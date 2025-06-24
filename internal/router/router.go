package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/internal/shopservice"
	"github.com/hmmm42/city-picks/internal/user"
	"github.com/hmmm42/city-picks/pkg/code"
)

func NewRouter() *gin.Engine {
	//r := gin.New()
	//r.Use(gin.Recovery())
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		slog.Warn("Received ping request")
		code.WriteResponse(c, code.ErrSuccess, "pong")
	})

	r.GET("/user/verificationcode/:phone", user.GetVerificationCode)
	r.POST("/user/login", user.Login)

	protected := r.Group("/")
	//protected.Use(middleware.JWT())
	{
		protected.GET("/p_ping", func(c *gin.Context) {
			slog.Debug("Received protected ping request")
			code.WriteResponse(c, code.ErrSuccess, "pong from protected route")
		})

		protected.GET("/shop/:id", shopservice.QueryShopByID)
		protected.GET("/shop_type", shopservice.QueryShopTypeList)
		protected.POST("/shop/create", shopservice.CreateShop)
		protected.POST("/shop/update", shopservice.UpdateShop)
		protected.DELETE("/shop/:id", shopservice.DeleteShop)
	}
	return r
}
