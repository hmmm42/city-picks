package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/internal/repository"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const CountBits = 32

type VoucherService interface {
	CreateVoucher(ctx context.Context, voucher *model.TbVoucher, seckillVoucher *model.TbSeckillVoucher) error
	SeckillVoucher(ctx context.Context, voucherID, userID uint64) (*model.TbVoucherOrder, error)
}

type voucherService struct {
	voucherRepo      repository.VoucherRepo
	voucherOrderRepo repository.VoucherOrderRepo
	redisClient      *redis.Client
	redsync          *redsync.Redsync // 用于分布式锁
	logger           *slog.Logger
}

func (s *voucherService) CreateVoucher(ctx context.Context, voucher *model.TbVoucher, seckillVoucher *model.TbSeckillVoucher) error {
	if seckillVoucher != nil {
		return s.voucherRepo.CreateSeckillVoucher(ctx, voucher, seckillVoucher)
	}
	return s.voucherRepo.CreateVoucher(ctx, voucher)
}

func (s *voucherService) SeckillVoucher(ctx context.Context, voucherID, userID uint64) (*model.TbVoucherOrder, error) {
	// 查询优惠券
	seckillVoucher, err := s.voucherRepo.GetSeckillVoucherByID(ctx, voucherID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("seckill voucher not found")
	}
	if err != nil {
		slog.Error("failed to query seckill voucher", "err", err)
		return nil, fmt.Errorf("failed to query seckill voucher: %w", err)
	}

	// 判断秒杀是否开始或结束
	now := time.Now()
	if seckillVoucher.BeginTime.After(now) || seckillVoucher.EndTime.Before(now) {
		return nil, fmt.Errorf("seckill not available at this time")
	}

	// 判断库存
	if seckillVoucher.Stock <= 0 {
		return nil, fmt.Errorf("seckill voucher out of stock")
	}

	// 检查用户是否已经购买过该优惠券
	exists, err := s.voucherOrderRepo.HasUserPurchasedVoucher(ctx, voucherID, userID)
	if err != nil {
		slog.Error("failed to check existing voucher order", "err", err)
		return nil, fmt.Errorf("failed to check existing voucher order: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user has already purchased this voucher")
	}

	// 分布式锁, 防止集群内重复下单
	mutex := s.redsync.NewMutex(fmt.Sprintf("order:%d", userID), redsync.WithTries(1))
	if err = mutex.LockContext(ctx); err != nil {
		return nil, fmt.Errorf("previous order logic is still processing/not allowing duplicate orders")
	}
	defer func() {
		_, err = mutex.Unlock()
		if err != nil {
			slog.Error("failed to unlock mutex", "err", err)
		}
	}()

	order, err := s.createVoucherOrder(ctx, voucherID, userID)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *voucherService) createVoucherOrder(ctx context.Context, voucherID, userID uint64) (*model.TbVoucherOrder, error) {
	// 创建订单并扣减库存
	orderID := s.nextID(ctx, "order")
	if orderID == -1 {
		return nil, fmt.Errorf("failed to generate order ID")
	}

	order := &model.TbVoucherOrder{
		ID:        orderID,
		VoucherID: voucherID,
		UserID:    userID,
	}

	err := s.voucherRepo.CreateVoucherOrderAndReduceStock(ctx, order, voucherID)
	if err != nil {
		slog.Error("failed to create voucher order and reduce stock", "err", err)
		return nil, fmt.Errorf("failed to create voucher order and reduce stock: %w", err)
	}

	return order, nil
}

func (s *voucherService) nextID(ctx context.Context, keyPrefix string) int64 {
	now := time.Now().Unix()

	date := time.Now().Format(time.DateOnly)
	count, err := s.redisClient.Incr(ctx, "incr:"+keyPrefix+":"+date).Result()
	if err != nil {
		slog.Error("failed to increment ID", "err", err)
		return -1
	}
	return (now << CountBits) | count
}

func NewVoucherService(voucherRepo repository.VoucherRepo, voucherOrderRepo repository.VoucherOrderRepo, redisClient *redis.Client, logger *slog.Logger) VoucherService {
	return &voucherService{
		voucherRepo:      voucherRepo,
		voucherOrderRepo: voucherOrderRepo,
		redisClient:      redisClient,
		logger:           logger,
	}
}
