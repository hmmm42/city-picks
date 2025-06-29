package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/dal/query"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type VoucherRepo interface {
	CreateVoucher(ctx context.Context, voucher *model.TbVoucher) error
	CreateSeckillVoucher(ctx context.Context, voucher *model.TbVoucher, seckillVoucher *model.TbSeckillVoucher) error
	GetSeckillVoucherByID(ctx context.Context, voucherID uint64) (*model.TbSeckillVoucher, error)
	CreateVoucherOrderAndReduceStock(ctx context.Context, order *model.TbVoucherOrder) error
	SetVoucherStockCache(ctx context.Context, voucher *model.TbSeckillVoucher) error
	ExecScript(ctx context.Context, script string, keys []string, args ...any) (int64, error)
}

type voucherRepo struct {
	q      *query.Query
	rdb    *redis.Client
	logger *slog.Logger
}

func (r *voucherRepo) CreateVoucher(ctx context.Context, voucher *model.TbVoucher) error {
	return r.q.TbVoucher.WithContext(ctx).Create(voucher)
}

// CreateSeckillVoucher 同时创建优惠券信息和秒杀信息
func (r *voucherRepo) CreateSeckillVoucher(ctx context.Context, voucher *model.TbVoucher, seckillVoucher *model.TbSeckillVoucher) error {
	return r.q.Transaction(func(tx *query.Query) error {
		err := tx.TbVoucher.WithContext(ctx).Create(voucher)
		if err != nil {
			return err
		}
		seckillVoucher.VoucherID = voucher.ID
		return tx.TbSeckillVoucher.WithContext(ctx).Create(seckillVoucher)
	})
}

func (r *voucherRepo) GetSeckillVoucherByID(ctx context.Context, voucherID uint64) (*model.TbSeckillVoucher, error) {
	return r.q.TbSeckillVoucher.WithContext(ctx).Where(r.q.TbSeckillVoucher.VoucherID.Eq(voucherID)).First()
}

func (r *voucherRepo) CreateVoucherOrderAndReduceStock(ctx context.Context, order *model.TbVoucherOrder) error {
	return r.q.Transaction(func(tx *query.Query) error {
		info, err := tx.TbSeckillVoucher.WithContext(ctx).Where(
			tx.TbSeckillVoucher.VoucherID.Eq(order.VoucherID),
			tx.TbSeckillVoucher.Stock.Gt(0),
		).UpdateSimple(tx.TbSeckillVoucher.Stock.Add(-1))
		if err != nil {
			return err
		}
		if info.RowsAffected == 0 {
			return errors.New("row not found or stock insufficient")
		}

		o := r.q.TbVoucherOrder
		return tx.TbVoucherOrder.WithContext(ctx).
			Omit(o.PayTime, o.UseTime, o.RefundTime).
			Create(order)
	})
}

func getVoucherKey(voucherID uint64) string {
	return fmt.Sprintf("seckill:stock:%d", voucherID)
}

func (r *voucherRepo) SetVoucherStockCache(ctx context.Context, voucher *model.TbSeckillVoucher) error {
	return r.rdb.Set(ctx, getVoucherKey(voucher.VoucherID), voucher.Stock, 0).Err()
}

func (r *voucherRepo) GetVoucherStockCache(ctx context.Context, voucherID uint64) (int64, error) {
	return r.rdb.Get(ctx, getVoucherKey(voucherID)).Int64()
}

func (r *voucherRepo) ExecScript(ctx context.Context, script string, keys []string, args ...any) (int64, error) {
	// 使用 Lua 脚本执行 Redis 命令
	result, err := r.rdb.Eval(ctx, script, keys, args...).Result()
	if err != nil {
		r.logger.Error("failed to execute Lua script", "err", err)
		return 0, err
	}

	// 将结果转换为 int64
	if count, ok := result.(int64); ok {
		return count, nil
	}

	return 0, fmt.Errorf("unexpected result type: %T", result)
}

func NewVoucherRepo(db *gorm.DB, rdb *redis.Client, logger *slog.Logger) VoucherRepo {
	return &voucherRepo{
		q:      query.Use(db),
		rdb:    rdb,
		logger: logger,
	}
}
