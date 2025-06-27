package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	shopservice2 "github.com/hmmm42/city-picks/internal/handler/shopservice"
	"github.com/hmmm42/city-picks/internal/handler/user"
	"github.com/hmmm42/city-picks/pkg/code"
)

func NewRouter(userHandler *user.LoginHandler) *gin.Engine {
	//r := gin.New()
	//r.Use(gin.Recovery())
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		slog.Warn("Received ping request")
		code.WriteResponse(c, code.ErrSuccess, "pong")
	})

	r.GET("/user/verificationcode/:phone", userHandler.GetVerificationCode)
	r.POST("/user/login", userHandler.Login)

	protected := r.Group("/")
	//protected.Use(middleware.JWT())
	{
		protected.GET("/p_ping", func(c *gin.Context) {
			slog.Debug("Received protected ping request")
			code.WriteResponse(c, code.ErrSuccess, "pong from protected route")
		})

		protected.GET("/shop/:id", shopservice2.QueryShopByID)
		protected.GET("/shop_type", shopservice2.QueryShopTypeList)
		protected.POST("/shop/create", shopservice2.CreateShop)
		protected.POST("/shop/update", shopservice2.UpdateShop)
		protected.DELETE("/shop/:id", shopservice2.DeleteShop)

		protected.POST("/voucher/create", shopservice2.CreateVoucher)
		protected.POST("/voucher/seckill", shopservice2.SeckillVoucher)
	}
	return r
}
