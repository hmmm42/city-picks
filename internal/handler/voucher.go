package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
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

func (h *VoucherHandler) CreateVoucher(c *gin.Context) {
	var req service.VoucherDTO
	err := c.BindJSON(&req)
	if err != nil {
		slog.Error("failed to bind voucher data", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

	err = h.voucherService.CreateVoucher(c.Request.Context(), &req)
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

	orderID, err := h.voucherService.SeckillVoucher(c.Request.Context(), vid, uid)
	if err != nil {
		slog.Error("failed to seckill voucher", "err", err)
		code.WriteResponse(c, code.ErrDatabase, err.Error()) // 返回具体的错误信息
		return
	}
	code.WriteResponse(c, code.ErrSuccess, gin.H{
		"order_id": orderID,
	})
}
