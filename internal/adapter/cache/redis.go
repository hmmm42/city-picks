package cache

import (
	"context"

	"github.com/hmmm42/city-picks/internal/config"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func NewRedisClient(setting *config.RedisSetting) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     setting.Host + ":" + setting.Port,
		PoolSize: setting.PoolSize,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	//slog.Debug("ping redis", "host", setting.Host, "port", setting.Port)
	return client, nil
}
