//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/hmmm42/city-picks/internal/adapter/cache"
	"github.com/hmmm42/city-picks/internal/adapter/persistent"
	"github.com/hmmm42/city-picks/internal/config"
	"github.com/hmmm42/city-picks/internal/handler"
	"github.com/hmmm42/city-picks/internal/mq"
	"github.com/hmmm42/city-picks/internal/repository"
	"github.com/hmmm42/city-picks/internal/router"
	"github.com/hmmm42/city-picks/internal/service"
	"github.com/hmmm42/city-picks/pkg/logger"
)

type App struct {
	Engine        *gin.Engine
	OrderConsumer *mq.OrderConsumer
}

var configSet = wire.NewSet(config.NewOptions,
	wire.FieldsOf(new(*config.Options),
		// 从 *Options 中提取出子结构体，供其他Provider使用
		"MySQL", "Redis", "Log", "JWT", "Server"))
var dbSet = wire.NewSet(persistent.NewMySQL, cache.NewRedisClient)
var loggerSet = wire.NewSet(logger.NewLogger)

var repositorySet = wire.NewSet(
	repository.NewUserRepo,
	repository.NewShopRepo,
	repository.NewVoucherRepo,
	repository.NewVoucherOrderRepo,
	repository.NewMessageQueue,
)

var serviceSet = wire.NewSet(
	service.NewUserService,
	service.NewShopService,
	service.NewVoucherService,
)

var handlerSet = wire.NewSet(
	handler.NewLoginHandler,
	handler.NewShopService,
	handler.NewVoucherHandler,
)

var routerSet = wire.NewSet(router.NewRouter)

var mqSet = wire.NewSet(mq.NewOrderConsumer)

func InitApp() (*App, func(), error) {
	wire.Build(
		configSet,
		dbSet,
		loggerSet,
		repositorySet,
		serviceSet,
		handlerSet,
		routerSet,
		mqSet,
		wire.Struct(new(App), "*"),
	)
	return nil, nil, nil
}
