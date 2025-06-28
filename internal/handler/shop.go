package handler

import (
	"errors"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/internal/service"
	"github.com/hmmm42/city-picks/pkg/code"
	"gorm.io/gorm"
)

type ShopService struct {
	service service.ShopService
}

func NewShopService(svc service.ShopService) *ShopService {
	return &ShopService{
		service: svc,
	}
}

func (s *ShopService) QueryShopByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		code.WriteResponse(c, code.ErrValidation, "Shop ID is required")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		code.WriteResponse(c, code.ErrValidation, "Invalid Shop ID format")
		return
	}

	shop, err := s.service.GetShopByID(c.Request.Context(), id)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		code.WriteResponse(c, code.ErrDatabase, "Shop not found")
	} else if err != nil {
		slog.Error("Failed to query shop by ID", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
	} else {
		code.WriteResponse(c, code.ErrSuccess, shop)
	}
}

func (s *ShopService) QueryShopTypeList(c *gin.Context) {
	shopTypes, err := s.service.GetShopTypeList(c.Request.Context())
	if err != nil {
		slog.Error("Failed to query shop type list", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, shopTypes)
}

func (s *ShopService) UpdateShop(c *gin.Context) {
	var shop model.TbShop
	if err := c.BindJSON(&shop); err != nil {
		slog.Error("Failed to bind shop data", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

	if err := s.service.UpdateShop(c.Request.Context(), &shop); err != nil {
		slog.Error("Failed to update shop", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, nil)
}

func (s *ShopService) CreateShop(c *gin.Context) {
	var shop model.TbShop
	if err := c.BindJSON(&shop); err != nil {
		slog.Error("Failed to bind shop data", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

	if err := s.service.CreateShop(c.Request.Context(), &shop); err != nil {
		slog.Error("Failed to create shop", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, nil)
}

func (s *ShopService) DeleteShop(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		code.WriteResponse(c, code.ErrValidation, "Shop ID is required")
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		code.WriteResponse(c, code.ErrValidation, "Invalid Shop ID format")
		return
	}

	if err := s.service.DeleteShop(c.Request.Context(), id); err != nil {
		slog.Error("Failed to delete shop", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, nil)
}
