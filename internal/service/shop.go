package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/internal/repository"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type ShopService interface {
	GetShopByID(ctx context.Context, id uint64) (*model.TbShop, error)
	UpdateShop(ctx context.Context, shop *model.TbShop) error
	CreateShop(ctx context.Context, shop *model.TbShop) error
	DeleteShop(ctx context.Context, id uint64) error
	GetShopTypeList(ctx context.Context) ([]*model.TbShopType, error)
}

type shopService struct {
	repo repository.ShopRepo
	sg   singleflight.Group
}

func (s *shopService) GetShopByID(ctx context.Context, id uint64) (*model.TbShop, error) {
	cacheShop, err := s.repo.GetShopCache(ctx, id)
	if err == nil {
		return cacheShop, nil
	}

	if !errors.Is(err, redis.Nil) {
		slog.Error("failed to get shop from cache", "err", err)
		return nil, err
	}

	// 优化: singleflight 只在缓存未命中时才会执行查询
	key := fmt.Sprintf("shop-singleflight:%d", id)
	dbShop, err, _ := s.sg.Do(key, func() (any, error) {
		shop, err := s.repo.GetShopByID(ctx, id)

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if cacheErr := s.repo.SetShopCacheNil(ctx, id); cacheErr != nil {
				slog.Error("failed to set shop cache nil", "err", cacheErr)
			}
			return nil, err
		}

		if err != nil {
			slog.Error("failed to get shop from database", "err", err)
			return nil, err
		}

		if err = s.repo.SetShopCache(ctx, shop); err != nil {
			slog.Error("failed to set shop cache", "err", err)
		}
		return shop, nil
	})

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("shop_id %v not found: %w", id, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get shop by ID %v: %w", id, err)
	}

	return dbShop.(*model.TbShop), nil
}

func (s *shopService) UpdateShop(ctx context.Context, shop *model.TbShop) error {
	if err := s.repo.UpdateShop(ctx, shop); err != nil {
		slog.Error("failed to update shop in database", "err", err)
		return fmt.Errorf("failed to update shop: %w", err)
	}

	if err := s.repo.DeleteShopCache(ctx, shop.ID); err != nil {
		slog.Warn("failed to delete shop cache", "err", err, "shop_id", shop.ID)
	}
	return nil
}

func (s *shopService) CreateShop(ctx context.Context, shop *model.TbShop) error {
	return s.repo.CreateShop(ctx, shop)
}

func (s *shopService) DeleteShop(ctx context.Context, id uint64) error {
	if err := s.repo.DeleteShop(ctx, id); err != nil {
		slog.Error("failed to delete shop from database", "err", err)
		return fmt.Errorf("failed to delete shop: %w", err)
	}

	if err := s.repo.DeleteShopCache(ctx, id); err != nil {
		slog.Warn("failed to delete shop cache", "err", err, "shop_id", id)
	}
	return nil
}

func (s *shopService) GetShopTypeList(ctx context.Context) ([]*model.TbShopType, error) {
	shopTypes, err := s.repo.GetShopTypeListCache(ctx)
	if err == nil && len(shopTypes) > 0 {
		return shopTypes, nil
	}

	if err != nil && !errors.Is(err, redis.Nil) {
		slog.Error("failed to get shop types from cache", "err", err)
	}

	shopTypes, err = s.repo.GetShopTypeList(ctx)
	if err != nil {
		slog.Error("failed to get shop types from database", "err", err)
		return nil, err
	}

	if err = s.repo.SetShopTypeListCache(ctx, shopTypes); err != nil {
		slog.Warn("failed to set shop type list cache", "err", err)
	}
	return shopTypes, nil
}

func NewShopService(repo repository.ShopRepo) ShopService {
	return &shopService{
		repo: repo,
	}
}
