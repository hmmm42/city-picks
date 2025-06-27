package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/dal/query"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	shopKeyPrefix = "cache:shop:"
	shopTypeKey   = "cache:shopType:"
	cacheNullTTL  = 10 * time.Minute
	cacheShopTTL  = 2 * time.Hour
)

type ShopRepo interface {
	GetShopByID(ctx context.Context, id uint64) (*model.TbShop, error)
	SetShopCache(ctx context.Context, shop *model.TbShop) error
	GetShopCache(ctx context.Context, id uint64) (*model.TbShop, error)
	SetShopCacheNil(ctx context.Context, id uint64) error
	GetShopTypeList(ctx context.Context) ([]*model.TbShopType, error)
	GetShopTypeListCache(ctx context.Context) ([]*model.TbShopType, error)
	SetShopTypeListCache(ctx context.Context, types []*model.TbShopType) error
	UpdateShop(ctx context.Context, shop *model.TbShop) error
	DeleteShopCache(ctx context.Context, id uint64) error
	CreateShop(ctx context.Context, shop *model.TbShop) error
	DeleteShop(ctx context.Context, id uint64) error
}

type shopRepo struct {
	q   *query.Query
	rdb *redis.Client
}

func (r *shopRepo) GetShopByID(ctx context.Context, id uint64) (*model.TbShop, error) {
	s := r.q.TbShop
	return s.WithContext(ctx).Where(s.ID.Eq(id)).First()
}

func (r *shopRepo) SetShopCache(ctx context.Context, shop *model.TbShop) error {
	key := getShopKey(shop.ID)
	data, err := json.Marshal(shop)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, key, data, cacheShopTTL).Err()
}

func (r *shopRepo) GetShopCache(ctx context.Context, id uint64) (*model.TbShop, error) {
	key := getShopKey(id)
	data, err := r.rdb.Get(ctx, key).Result()
	if err != nil { // repo 层不处理业务逻辑, 返回 redis.Nil 错误
		return nil, err
	}

	var shop model.TbShop
	if err = json.Unmarshal([]byte(data), &shop); err != nil {
		return nil, err
	}
	return &shop, nil
}

func (r *shopRepo) SetShopCacheNil(ctx context.Context, id uint64) error {
	key := getShopKey(id)
	return r.rdb.Set(ctx, key, "", cacheNullTTL).Err()
}

func (r *shopRepo) GetShopTypeList(ctx context.Context) ([]*model.TbShopType, error) {
	st := r.q.TbShopType
	return st.WithContext(ctx).Order(st.Sort).Find()
}

func (r *shopRepo) GetShopTypeListCache(ctx context.Context) (res []*model.TbShopType, err error) {
	typeListCached, err := r.rdb.LRange(ctx, shopTypeKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	res = make([]*model.TbShopType, len(typeListCached))
	for i, t := range typeListCached {
		if err = json.Unmarshal([]byte(t), &res[i]); err != nil {
			return nil, err
		}
	}
	return
}

func (r *shopRepo) SetShopTypeListCache(ctx context.Context, types []*model.TbShopType) error {
	if len(types) == 0 {
		return nil
	}

	pipe := r.rdb.TxPipeline()
	for _, t := range types {
		data, err := json.Marshal(t)
		if err != nil {
			return err
		}
		pipe.RPush(ctx, shopTypeKey, string(data))
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (r *shopRepo) UpdateShop(ctx context.Context, shop *model.TbShop) error {
	s := r.q.TbShop
	_, err := s.WithContext(ctx).Where(s.ID.Eq(shop.ID)).Updates(shop)
	return err
}

func (r *shopRepo) DeleteShopCache(ctx context.Context, id uint64) error {
	return r.rdb.Del(ctx, getShopKey(id)).Err()
}

func (r *shopRepo) CreateShop(ctx context.Context, shop *model.TbShop) error {
	return r.q.TbShop.WithContext(ctx).Create(shop)
}

func (r *shopRepo) DeleteShop(ctx context.Context, id uint64) error {
	s := r.q.TbShop
	_, err := s.WithContext(ctx).Where(s.ID.Eq(id)).Delete()
	return err
}

func getShopKey(id uint64) string {
	return fmt.Sprintf("%v%d", shopKeyPrefix, id)
}

func NewShopRepo(db *gorm.DB, rdb *redis.Client) ShopRepo {
	return &shopRepo{
		q:   query.Use(db),
		rdb: rdb,
	}
}
