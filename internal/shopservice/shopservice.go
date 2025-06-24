package shopservice

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/dal/query"
	"github.com/hmmm42/city-picks/internal/db"
	"github.com/hmmm42/city-picks/pkg/code"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	ShopKeyPrefix = "cache:shop:"
	ShopTypeKey   = "cache:shopType:"
)

func QueryShopByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		code.WriteResponse(c, code.ErrValidation, "Shop ID is required")
		return
	}

	shopGot, err := db.RedisClient.Get(c, ShopKeyPrefix+idStr).Result()
	if err == nil {
		slog.Info("Cache hit for shop ID", "id", idStr)
		var shop model.TbShop
		err = json.Unmarshal([]byte(shopGot), &shop)
		if err != nil {
			slog.Error("Failed to decode shop data from cache", "err", err)
			code.WriteResponse(c, code.ErrDecodingJSON, "Failed to decode shop data from cache")
		} else {
			code.WriteResponse(c, code.ErrSuccess, shop)
		}
		return
	}
	if !errors.Is(err, redis.Nil) {
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}

	slog.Warn("Cache miss for shop ID", "id", idStr)
	shopQuery := query.TbShop
	id, err := strconv.Atoi(idStr)
	if err != nil {
		code.WriteResponse(c, code.ErrValidation, "Invalid Shop ID format")
		return
	}
	shop, err := shopQuery.Where(shopQuery.ID.Eq(uint64(id))).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		code.WriteResponse(c, code.ErrDatabase, "Shop not found")
		return
	}
	if err != nil {
		slog.Error("Failed to query shop from database", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}

	shopMarshalled, err := json.Marshal(shop)
	if err != nil {
		slog.Error("Failed to marshal shop data", "err", err)
		code.WriteResponse(c, code.ErrEncodingJSON, "Failed to encode shop data")
		return
	}
	_, err = db.RedisClient.Set(c, ShopKeyPrefix+idStr, shopMarshalled, 0).Result()
	if err != nil {
		slog.Error("Failed to cache shop data", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, shop)
}

func QueryShopTypeList(c *gin.Context) {
	// 取出list中全部数据
	shopTypesGot, err := db.RedisClient.LRange(c, ShopTypeKey, 0, -1).Result()
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

		pipeline := db.RedisClient.Pipeline()
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
	if err = db.RedisClient.Del(c, key).Err(); err != nil {
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
	if err = db.RedisClient.Del(c, key).Err(); err != nil {
		slog.Error("Failed to delete shop cache", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, nil)
}
