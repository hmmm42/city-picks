package shopservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/dal/query"
	"github.com/hmmm42/city-picks/internal/db"
	"github.com/hmmm42/city-picks/pkg/code"
	"gorm.io/gorm"
)

const CountBits = 32

type Voucher struct {
	ShopID      int    `json:"shop_id"` //关联的商店id
	Title       string `json:"title"`
	SubTitle    string `json:"subTitle"`
	Rules       string `json:"rules"`
	PayValue    int    `json:"pay_value"` //优惠的价格
	ActualValue int    `json:"actual_value"`
	Type        int    `json:"type"`  //优惠卷类型
	Stock       int    `json:"stock"` //库存
	BeginTime   string `json:"begin_time"`
	EndTime     string `json:"end_time"`
}

func CreateVoucher(c *gin.Context) {
	var voucher Voucher
	err := c.BindJSON(voucher)
	if err != nil {
		slog.Error("failed to bind voucher data", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

	switch voucher.Type {
	case 0: // 平价券, 随时可以购买
		err = createOrdinaryVoucher(&voucher)
	case 1: // 特价券, 需要在指定时间段内购买
		err = createSeckillVoucher(&voucher)
	default:
		code.WriteResponse(c, code.ErrValidation, "type must be 0 or 1")
		return
	}
}

func createOrdinaryVoucher(voucher *Voucher) error {
	v := model.TbVoucher{
		ShopID:      uint64(voucher.ShopID),
		Title:       voucher.Title,
		SubTitle:    voucher.SubTitle,
		Rules:       voucher.Rules,
		PayValue:    uint64(voucher.PayValue),
		ActualValue: int64(voucher.ActualValue),
		Type:        uint8(voucher.Type),
	}

	err := query.TbVoucher.Create(&v)
	if err != nil {
		slog.Error("failed to create ordinary voucher", "err", err)
		return err
	}
	return nil
}

func createSeckillVoucher(voucher *Voucher) error {
	layout := "2006-01-02 15:04:05"
	start, err := time.Parse(layout, voucher.BeginTime)
	if err != nil {
		slog.Error("failed to parse begin time", "err", err)
		return fmt.Errorf("time format must be '2006-01-02 15:04:05'")
	}
	end, err := time.Parse(layout, voucher.EndTime)
	if err != nil {
		slog.Error("failed to parse end time", "err", err)
		return fmt.Errorf("time format must be '2006-01-02 15:04:05'")
	}

	v := model.TbVoucher{
		ShopID:      uint64(voucher.ShopID),
		Title:       voucher.Title,
		SubTitle:    voucher.SubTitle,
		Rules:       voucher.Rules,
		PayValue:    uint64(voucher.PayValue),
		ActualValue: int64(voucher.ActualValue),
		Type:        uint8(voucher.Type),
	}

	// 使用事务
	q := query.Use(db.DBEngine)
	return q.Transaction(func(tx *query.Query) error {
		err := tx.TbVoucher.Create(&v)
		if err != nil {
			slog.Error("failed to create seckill voucher", "err", err)
			return err
		}

		seckill := model.TbSeckillVoucher{
			VoucherID: v.ID,
			Stock:     int64(voucher.Stock),
			BeginTime: start,
			EndTime:   end,
		}
		err = tx.TbSeckillVoucher.Create(&seckill)
		if err != nil {
			slog.Error("failed to create seckill voucher", "err", err)
			return err
		}
		return nil
	})
}

func nextID(ctx context.Context, keyPrefix string) int64 {
	now := time.Now().Unix()

	date := time.Now().Format(time.DateOnly)
	count, err := db.RedisClient.Incr(ctx, "incr:"+keyPrefix+":"+date).Result()
	if err != nil {
		slog.Error("failed to increment ID", "err", err)
		return -1
	}
	return (now << CountBits) | count
}

type seckillRequest struct {
	VoucherID int64 `json:"voucher_id"`
	UserID    int64 `json:"user_id"`
}

func SeckillVoucher(c *gin.Context) {
	var req seckillRequest
	err := c.BindJSON(&req)
	if err != nil {
		slog.Error("failed to bind seckill request", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

	seckillQuery := query.TbSeckillVoucher
	voucher, err := seckillQuery.Where(seckillQuery.VoucherID.Eq(uint64(req.VoucherID))).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	if err != nil {
		slog.Error("failed to query seckill voucher", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}

	now := time.Now()
	if voucher.BeginTime.After(now) || voucher.EndTime.Before(now) {
		code.WriteResponse(c, code.ErrValidation, "seckill not available at this time")
		return
	}
	if voucher.Stock <= 0 {
		code.WriteResponse(c, code.ErrValidation, "seckill voucher out of stock")
		return
	}
	createVoucherOrder(c, req)
}

func createVoucherOrder(c *gin.Context, req seckillRequest) {
	orderQuery := query.TbVoucherOrder
	// 检查用户是否已经购买过该优惠券
	exists, err := orderQuery.Where(
		orderQuery.VoucherID.Eq(uint64(req.VoucherID)),
		orderQuery.UserID.Eq(uint64(req.UserID)),
	).Find()
	if err != nil {
		slog.Error("failed to check existing voucher order", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	if len(exists) > 0 {
		code.WriteResponse(c, code.ErrValidation, "user has already purchased this voucher")
		return
	}

	order := model.TbVoucherOrder{
		ID:        nextID(c, "order"),
		VoucherID: uint64(req.VoucherID),
		UserID:    uint64(req.UserID),
	}

	q := query.Use(db.DBEngine)
	err = q.Transaction(func(tx *query.Query) error {
		info, err := tx.TbSeckillVoucher.Where(
			tx.TbSeckillVoucher.VoucherID.Eq(uint64(req.VoucherID)),
			tx.TbSeckillVoucher.Stock.Gt(0), // 乐观锁 CAS
		).UpdateSimple(tx.TbSeckillVoucher.Stock.Add(-1))
		if err != nil {
			return err
		}
		if info.RowsAffected == 0 {
			return fmt.Errorf("row not found or stock insufficient")
		}

		// 只插入指定列
		return tx.TbVoucherOrder.Select(
			tx.TbVoucherOrder.ID,
			tx.TbVoucherOrder.VoucherID,
			tx.TbVoucherOrder.UserID).
			Create(&order)
	})

	if err != nil {
		slog.Error("failed to create voucher order", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, order.ID)
}
