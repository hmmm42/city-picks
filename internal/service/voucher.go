package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/internal/repository"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const CountBits = 32

type VoucherService interface {
	CreateVoucher(ctx context.Context, voucher *model.TbVoucher, seckillVoucher *model.TbSeckillVoucher) error
	SeckillVoucher(ctx context.Context, voucherID, userID uint64) error
}

type voucherService struct {
	voucherRepo      repository.VoucherRepo
	voucherOrderRepo repository.VoucherOrderRepo
	redisClient      *redis.Client
	logger           *slog.Logger
}

func (s *voucherService) CreateVoucher(ctx context.Context, voucher *model.TbVoucher, seckillVoucher *model.TbSeckillVoucher) error {
	if seckillVoucher != nil {
		return s.voucherRepo.CreateSeckillVoucher(ctx, voucher, seckillVoucher)
	}
	return s.voucherRepo.CreateVoucher(ctx, voucher)
}

func (s *voucherService) SeckillVoucher(ctx context.Context, voucherID, userID uint64) error {
	// 1. 查询优惠券
	seckillVoucher, err := s.voucherRepo.GetSeckillVoucherByID(ctx, voucherID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("seckill voucher not found")
	}
	if err != nil {
		slog.Error("failed to query seckill voucher", "err", err)
		return fmt.Errorf("failed to query seckill voucher: %w", err)
	}

	// 2. 判断秒杀是否开始或结束
	now := time.Now()
	if seckillVoucher.BeginTime.After(now) || seckillVoucher.EndTime.Before(now) {
		return fmt.Errorf("seckill not available at this time")
	}

	// 3. 判断库存
	if seckillVoucher.Stock <= 0 {
		return fmt.Errorf("seckill voucher out of stock")
	}

	// 4. 检查用户是否已经购买过该优惠券
	exists, err := s.voucherOrderRepo.HasUserPurchasedVoucher(ctx, voucherID, userID)
	if err != nil {
		slog.Error("failed to check existing voucher order", "err", err)
		return fmt.Errorf("failed to check existing voucher order: %w", err)
	}
	if exists {
		return fmt.Errorf("user has already purchased this voucher")
	}

	// 5. 创建订单并扣减库存
	orderID := s.nextID(ctx, "order")
	if orderID == -1 {
		return fmt.Errorf("failed to generate order ID")
	}

	order := &model.TbVoucherOrder{
		ID:        orderID,
		VoucherID: voucherID,
		UserID:    userID,
	}

	err = s.voucherRepo.CreateVoucherOrderAndReduceStock(ctx, order, voucherID)
	if err != nil {
		slog.Error("failed to create voucher order and reduce stock", "err", err)
		return fmt.Errorf("failed to create voucher order and reduce stock: %w", err)
	}

	return nil
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
