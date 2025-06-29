package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/internal/repository"
	"github.com/hmmm42/city-picks/pkg/json_time"
	sf "github.com/hmmm42/city-picks/pkg/sonyflake"
	"github.com/sony/sonyflake"
)

// VoucherDTO defines the request structure for creating a voucher.
type VoucherDTO struct {
	ShopID      uint64               `json:"shop_id"` //关联的商店id
	Title       string               `json:"title"`
	SubTitle    string               `json:"subTitle"`
	Rules       string               `json:"rules"`
	PayValue    uint64               `json:"pay_value"` //优惠的价格
	ActualValue int64                `json:"actual_value"`
	Type        uint8                `json:"type"`  //优惠卷类型
	Stock       int64                `json:"stock"` //库存
	BeginTime   json_time.CustomTime `json:"begin_time"`
	EndTime     json_time.CustomTime `json:"end_time"`
}

type VoucherService interface {
	CreateVoucher(ctx context.Context, req *VoucherDTO) error
	SeckillVoucher(ctx context.Context, voucherID, userID uint64) (int64, error)
	CreateVoucherOrderDB(ctx context.Context, order *model.TbVoucherOrder) error
}

type voucherService struct {
	voucherRepo      repository.VoucherRepo
	voucherOrderRepo repository.VoucherOrderRepo
	sf               *sonyflake.Sonyflake    // 用于唯一ID
	mq               repository.MessageQueue // 用于消息队列
	logger           *slog.Logger
}

func (s *voucherService) CreateVoucher(ctx context.Context, req *VoucherDTO) error {
	voucher := &model.TbVoucher{
		ShopID:      req.ShopID,
		Title:       req.Title,
		SubTitle:    req.SubTitle,
		Rules:       req.Rules,
		PayValue:    req.PayValue,
		ActualValue: req.ActualValue,
		Type:        req.Type,
	}

	var seckillVoucher *model.TbSeckillVoucher
	if req.Type == 1 { // 特价券
		seckillVoucher = &model.TbSeckillVoucher{
			Stock:     req.Stock,
			BeginTime: time.Time(req.BeginTime),
			EndTime:   time.Time(req.EndTime),
		}
	}

	if seckillVoucher == nil {
		return s.voucherRepo.CreateVoucher(ctx, voucher)
	}
	err := s.voucherRepo.CreateSeckillVoucher(ctx, voucher, seckillVoucher)
	if err != nil {
		slog.Error("failed to create seckill voucher", "err", err)
		return fmt.Errorf("failed to create seckill voucher: %w", err)
	}
	return s.voucherRepo.SetVoucherStockCache(ctx, seckillVoucher)
}

//func (s *voucherService) SeckillVoucher(ctx context.Context, voucherID, userID uint64) (*model.TbVoucherOrder, error) {
//	// 查询优惠券
//	seckillVoucher, err := s.voucherRepo.GetSeckillVoucherByID(ctx, voucherID)
//	if errors.Is(err, gorm.ErrRecordNotFound) {
//		return nil, fmt.Errorf("seckill voucher not found")
//	}
//	if err != nil {
//		slog.Error("failed to query seckill voucher", "err", err)
//		return nil, fmt.Errorf("failed to query seckill voucher: %w", err)
//	}
//
//	// 判断秒杀是否开始或结束
//	now := time.Now()
//	if seckillVoucher.BeginTime.After(now) || seckillVoucher.EndTime.Before(now) {
//		return nil, fmt.Errorf("seckill not available at this time")
//	}
//
//	// 判断库存
//	if seckillVoucher.Stock <= 0 {
//		return nil, fmt.Errorf("seckill voucher out of stock")
//	}
//
//	// 检查用户是否已经购买过该优惠券
//	exists, err := s.voucherOrderRepo.HasUserPurchasedVoucher(ctx, voucherID, userID)
//	if err != nil {
//		slog.Error("failed to check existing voucher order", "err", err)
//		return nil, fmt.Errorf("failed to check existing voucher order: %w", err)
//	}
//	if exists {
//		return nil, fmt.Errorf("user has already purchased this voucher")
//	}
//
//	// 分布式锁, 防止集群内重复下单
//	mutex := s.redsync.NewMutex(fmt.Sprintf("order:%d", userID), redsync.WithTries(1))
//	if err = mutex.LockContext(ctx); err != nil {
//		return nil, fmt.Errorf("previous order logic is still processing/not allowing duplicate orders")
//	}
//	defer func() {
//		_, err = mutex.Unlock()
//		if err != nil {
//			slog.Error("failed to unlock mutex", "err", err)
//		}
//	}()
//
//	order, err := s.createVoucherOrder(ctx, voucherID, userID)
//	if err != nil {
//		return nil, err
//	}
//	return order, nil
//}

func (s *voucherService) SeckillVoucher(ctx context.Context, voucherID, userID uint64) (int64, error) {
	//script := redis.NewScript(adjustSeckill)
	orderID, err := s.sf.NextID()
	if err != nil {
		s.logger.Error("failed to generate unique ID", "err", err)
		return 0, fmt.Errorf("failed to generate unique ID: %w", err)
	}
	keys := []string{
		strconv.FormatUint(voucherID, 10),
		strconv.FormatUint(userID, 10),
		strconv.FormatUint(orderID, 10),
	}

	res, err := s.voucherRepo.ExecScript(ctx, adjustSeckill, keys)
	if err != nil {
		slog.Error("failed to execute seckill script", "err", err)
		return 0, err
	}

	switch res {
	case 0:
		return int64(orderID), nil
	case 1:
		return 0, fmt.Errorf("seckill voucher not found or out of stock")
	case 2:
		return 0, fmt.Errorf("user has already purchased this voucher")
	default:
		return 0, fmt.Errorf("unexpected result from seckill script: %d", res)
	}
}

func (s *voucherService) CreateVoucherOrderDB(ctx context.Context, order *model.TbVoucherOrder) error {
	// 创建订单并扣减库存
	err := s.voucherRepo.CreateVoucherOrderAndReduceStock(ctx, order)
	if err != nil {
		slog.Error("failed to create voucher order and reduce stock", "err", err)
		return err
	}
	return nil
}

func NewVoucherService(voucherRepo repository.VoucherRepo, voucherOrderRepo repository.VoucherOrderRepo, logger *slog.Logger) VoucherService {
	nsf, _ := sf.NewSonyflake()
	return &voucherService{
		voucherRepo:      voucherRepo,
		voucherOrderRepo: voucherOrderRepo,
		sf:               nsf,
		logger:           logger,
	}
}
