package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/internal/user"
	"github.com/hmmm42/city-picks/pkg/code"
)

func NewRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		slog.Debug("Received ping request")
		code.WriteResponse(c, code.ErrSuccess, "pong")
	})

	r.GET("/user/verificationcode/:phone", user.GetVerificationCode)
	r.POST("/user/login", user.Login)
	return r
}
