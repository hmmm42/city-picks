package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/dal/query"
	"gorm.io/gorm"
)

type VoucherRepo interface {
	CreateVoucher(ctx context.Context, voucher *model.TbVoucher) error
	CreateSeckillVoucher(ctx context.Context, voucher *model.TbVoucher, seckillVoucher *model.TbSeckillVoucher) error
	GetSeckillVoucherByID(ctx context.Context, voucherID uint64) (*model.TbSeckillVoucher, error)
	CreateVoucherOrderAndReduceStock(ctx context.Context, order *model.TbVoucherOrder, voucherID uint64) error
}

type voucherRepo struct {
	q      *query.Query
	logger *slog.Logger
}

func (r *voucherRepo) CreateVoucher(ctx context.Context, voucher *model.TbVoucher) error {
	return r.q.TbVoucher.WithContext(ctx).Create(voucher)
}

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

func (r *voucherRepo) CreateVoucherOrderAndReduceStock(ctx context.Context, order *model.TbVoucherOrder, voucherID uint64) error {
	return r.q.Transaction(func(tx *query.Query) error {
		info, err := tx.TbSeckillVoucher.WithContext(ctx).Where(
			tx.TbSeckillVoucher.VoucherID.Eq(voucherID),
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

func NewVoucherRepo(db *gorm.DB, logger *slog.Logger) VoucherRepo {
	return &voucherRepo{
		q:      query.Use(db),
		logger: logger,
	}
}
