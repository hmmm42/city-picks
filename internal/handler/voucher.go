package handler

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/internal/service"
	"github.com/hmmm42/city-picks/pkg/code"
)

type VoucherHandler struct {
	voucherService service.VoucherService
}

func NewVoucherHandler(svc service.VoucherService) *VoucherHandler {
	return &VoucherHandler{
		voucherService: svc,
	}
}

type CreateVoucherRequest struct {
	ShopID      uint64 `json:"shop_id"` //关联的商店id
	Title       string `json:"title"`
	SubTitle    string `json:"subTitle"`
	Rules       string `json:"rules"`
	PayValue    uint64 `json:"pay_value"` //优惠的价格
	ActualValue int64  `json:"actual_value"`
	Type        uint8  `json:"type"`  //优惠卷类型
	Stock       int64  `json:"stock"` //库存
	BeginTime   string `json:"begin_time"`
	EndTime     string `json:"end_time"`
}

func (h *VoucherHandler) CreateVoucher(c *gin.Context) {
	var req CreateVoucherRequest
	err := c.BindJSON(&req)
	if err != nil {
		slog.Error("failed to bind voucher data", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

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
		layout := "2006-01-02 15:04:05"
		start, err := time.Parse(layout, req.BeginTime)
		if err != nil {
			slog.Error("failed to parse begin time", "err", err)
			code.WriteResponse(c, code.ErrValidation, "time format must be '2006-01-02 15:04:05'")
			return
		}
		end, err := time.Parse(layout, req.EndTime)
		if err != nil {
			slog.Error("failed to parse end time", "err", err)
			code.WriteResponse(c, code.ErrValidation, "time format must be '2006-01-02 15:04:05'")
			return
		}
		seckillVoucher = &model.TbSeckillVoucher{
			Stock:     req.Stock,
			BeginTime: start,
			EndTime:   end,
		}
	}

	err = h.voucherService.CreateVoucher(c.Request.Context(), voucher, seckillVoucher)
	if err != nil {
		slog.Error("failed to create voucher", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, nil)
}

type SeckillVoucherRequest struct {
	VoucherID string `json:"voucher_id"`
	UserID    string `json:"user_id"`
}

func (h *VoucherHandler) SeckillVoucher(c *gin.Context) {
	var req SeckillVoucherRequest
	err := c.BindJSON(&req)
	if err != nil {
		slog.Error("failed to bind seckill request", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

	// 从 JWT token 中获取用户 ID
	//userID, exists := c.Get("userID")
	//if !exists {
	//	slog.Error("userID not found in context")
	//	code.WriteResponse(c, code.ErrValidation, nil)
	//	return
	//}

	vid, err := strconv.ParseUint(req.VoucherID, 10, 64)
	if err != nil {
		slog.Error("voucherID in context is not of type uint64", "err", err)
		code.WriteResponse(c, code.ErrValidation, nil)
		return
	}
	uid, err := strconv.ParseUint(req.UserID, 10, 64)
	if err != nil {
		slog.Error("userID in context is not of type uint64", "err", err)
		code.WriteResponse(c, code.ErrValidation, nil)
		return
	}

	order, err := h.voucherService.SeckillVoucher(c.Request.Context(), vid, uid)
	if err != nil {
		slog.Error("failed to seckill voucher", "err", err)
		code.WriteResponse(c, code.ErrDatabase, err.Error()) // 返回具体的错误信息
		return
	}
	code.WriteResponse(c, code.ErrSuccess, gin.H{
		"order_id": order.ID,
	})
}
