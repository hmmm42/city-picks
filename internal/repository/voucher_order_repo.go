package repository

import (
	"context"
	"log/slog"

	"github.com/hmmm42/city-picks/dal/query"
	"gorm.io/gorm"
)

type VoucherOrderRepo interface {
	HasUserPurchasedVoucher(ctx context.Context, voucherID, userID uint64) (bool, error)
}

type voucherOrderRepo struct {
	q      *query.Query
	logger *slog.Logger
}

func (r *voucherOrderRepo) HasUserPurchasedVoucher(ctx context.Context, voucherID, userID uint64) (bool, error) {
	count, err := r.q.TbVoucherOrder.WithContext(ctx).Where(
		r.q.TbVoucherOrder.VoucherID.Eq(voucherID),
		r.q.TbVoucherOrder.UserID.Eq(userID),
	).Count()
	if err != nil {
		r.logger.Error("failed to check existing voucher order", "err", err)
		return false, err
	}
	return count > 0, nil
}

func NewVoucherOrderRepo(db *gorm.DB, logger *slog.Logger) VoucherOrderRepo {
	return &voucherOrderRepo{
		q:      query.Use(db),
		logger: logger,
	}
}
