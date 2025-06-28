package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/internal/handler"
	"github.com/hmmm42/city-picks/pkg/code"
)

func NewRouter(
	userHandler *handler.LoginHandler,
	shopHandler *handler.ShopService,
	voucherHandler *handler.VoucherHandler,
) *gin.Engine {
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
	//protected.Use(jwtMiddleware.JWT())
	{
		protected.GET("/p_ping", func(c *gin.Context) {
			slog.Debug("Received protected ping request")
			code.WriteResponse(c, code.ErrSuccess, "pong from protected route")
		})

		protected.GET("/shop/:id", shopHandler.QueryShopByID)
		protected.GET("/shop_type", shopHandler.QueryShopTypeList)
		protected.POST("/shop/create", shopHandler.CreateShop)
		protected.POST("/shop/update", shopHandler.UpdateShop)
		protected.DELETE("/shop/:id", shopHandler.DeleteShop)

		protected.POST("/voucher/create", voucherHandler.CreateVoucher)
		protected.POST("/voucher/seckill", voucherHandler.SeckillVoucher)
	}
	return r
}
