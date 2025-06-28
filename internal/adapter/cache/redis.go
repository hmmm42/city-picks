package cache

import (
	"context"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/hmmm42/city-picks/internal/config"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(setting *config.RedisSetting) (*redis.Client, func(), error) {
	client := redis.NewClient(&redis.Options{
		Addr:     setting.Host + ":" + setting.Port,
		PoolSize: setting.PoolSize,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = client.Close()
	}
	return client, cleanup, nil
}

func NewRedsync(client *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(client)
	return redsync.New(pool)
}
