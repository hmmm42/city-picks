package shopservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/dal/query"
	redis2 "github.com/hmmm42/city-picks/internal/adapter/cache"
	"github.com/hmmm42/city-picks/pkg/code"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

const (
	ShopKeyPrefix = "cache:shop:"
	ShopTypeKey   = "cache:shopType:"
	CacheNullTTL  = 10 * time.Minute
	CacheShopTTL  = 2 * time.Hour
)

var sg singleflight.Group

func QueryShopByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		code.WriteResponse(c, code.ErrValidation, "Shop ID is required")
		return
	}

	shop, err, _ := sg.Do(idStr, func() (any, error) {
		return queryShopByIDSingle(c, idStr)
	})

	if errors.Is(err, gorm.ErrRecordNotFound) {
		code.WriteResponse(c, code.ErrDatabase, "Shop not found")
	} else if err != nil {
		code.WriteResponse(c, code.ErrDatabase, nil)
	} else {
		code.WriteResponse(c, code.ErrSuccess, shop)
	}
}

func queryShopByIDSingle(c context.Context, idStr string) (any, error) {
	slog.Debug("Querying shop by ID", "id", idStr)

	shopGot, err := redis2.RedisClient.Get(c, ShopKeyPrefix+idStr).Result()
	if err == nil {
		if shopGot == "" {
			return "", fmt.Errorf("%w: shop not found for ID %s", gorm.ErrRecordNotFound, idStr)
		}

		var shop model.TbShop
		err = json.Unmarshal([]byte(shopGot), &shop)
		if err != nil {
			slog.Error("Failed to decode shop data from cache", "err", err)
		}
		return shop, nil
	}
	if !errors.Is(err, redis.Nil) {
		slog.Error("Failed to get shop from Redis", "err", err)
		return "", err
	}

	slog.Warn("Cache miss for shop ID, querying database", "id", idStr)
	shopQuery := query.TbShop
	id, _ := strconv.Atoi(idStr)
	shop, err := shopQuery.Where(shopQuery.ID.Eq(uint64(id))).First()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 缓存 null 值，避免频繁查询数据库, 防止缓存穿透
		_, _ = redis2.RedisClient.Set(c, ShopKeyPrefix+idStr, "", CacheNullTTL).Result()
		return "", fmt.Errorf("%w: shop not found for ID %s", gorm.ErrRecordNotFound, idStr)
	}
	if err != nil {
		slog.Error("Failed to query shop from database", "err", err)
		return "", err
	}

	shopMarshalled, err := json.Marshal(shop)
	if err != nil {
		slog.Error("Failed to marshal shop data", "err", err)
		return nil, err
	}

	// 添加随机延迟，避免缓存雪崩
	_, err = redis2.RedisClient.Set(c, ShopKeyPrefix+idStr, shopMarshalled, CacheShopTTL+time.Duration(rand.Int31n(10000))).Result()
	if err != nil {
		slog.Error("Failed to cache shop data", "err", err)
		return "", err
	}

	return shop, nil
}

func QueryShopTypeList(c *gin.Context) {
	// 取出list中全部数据
	shopTypesGot, err := redis2.RedisClient.LRange(c, ShopTypeKey, 0, -1).Result()
	if errors.Is(err, redis.Nil) || len(shopTypesGot) == 0 {
		shopTypeQuery := query.TbShopType
		shopTypes, err := shopTypeQuery.Order(shopTypeQuery.Sort).Find()
		if err != nil {
			slog.Error("Failed to query shop types from database", "err", err)
			code.WriteResponse(c, code.ErrDatabase, nil)
			return
		}
		if len(shopTypes) == 0 {
			code.WriteResponse(c, code.ErrDatabase, "No shop types found")
			return
		}

		pipeline := redis2.RedisClient.Pipeline()
		for _, shopType := range shopTypes {
			v, _ := json.Marshal(shopType)
			pipeline.RPush(c, ShopTypeKey, string(v))
		}
		_, err = pipeline.Exec(c)
		if err != nil {
			slog.Error("Failed to cache shop types", "err", err)
			code.WriteResponse(c, code.ErrDatabase, nil)
			return
		}
		slog.Info("Cached shop types", "count", len(shopTypes))
		code.WriteResponse(c, code.ErrSuccess, shopTypes)
	} else if err != nil {
		slog.Error("Failed to get shop types from cache", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
	} else {
		shopTypesUnMarshalled := make([]model.TbShopType, len(shopTypesGot))
		for i, shopTypeStr := range shopTypesGot {
			_ = json.Unmarshal([]byte(shopTypeStr), &shopTypesUnMarshalled[i])
		}
		code.WriteResponse(c, code.ErrSuccess, shopTypesUnMarshalled)
	}
}

func UpdateShop(c *gin.Context) {
	// 保证数据库与缓存一致性: 先更新数据库，再删除缓存
	var shop model.TbShop
	if err := c.BindJSON(&shop); err != nil {
		slog.Error("Failed to bind shop data", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

	shopQuery := query.TbShop
	_, err := shopQuery.Where(shopQuery.ID.Eq(shop.ID)).Updates(shop)
	if err != nil {
		slog.Error("Failed to update shop in database", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}

	key := ShopKeyPrefix + strconv.FormatUint(shop.ID, 10)
	if err = redis2.RedisClient.Del(c, key).Err(); err != nil {
		slog.Error("Failed to delete shop cache", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, nil)
}

func CreateShop(c *gin.Context) {
	var shop model.TbShop
	if err := c.BindJSON(&shop); err != nil {
		slog.Error("Failed to bind shop data", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

	shopQuery := query.TbShop
	if err := shopQuery.Create(&shop); err != nil {
		slog.Error("Failed to create shop in database", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, nil)
}

func DeleteShop(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		code.WriteResponse(c, code.ErrValidation, "Shop ID is required")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		code.WriteResponse(c, code.ErrValidation, "Invalid Shop ID format")
		return
	}

	shopQuery := query.TbShop
	if _, err = shopQuery.Where(shopQuery.ID.Eq(uint64(id))).Delete(); err != nil {
		slog.Error("Failed to delete shop from database", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}

	key := ShopKeyPrefix + idStr
	if err = redis2.RedisClient.Del(c, key).Err(); err != nil {
		slog.Error("Failed to delete shop cache", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, nil)
}
