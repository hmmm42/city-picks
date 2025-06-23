package code

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func WriteResponse(c *gin.Context, code int, data any) {
	coder := ParseCoder(code)
	if coder.HTTPStatus() != http.StatusOK {
		c.JSON(coder.HTTPStatus(), Response{
			Code:    coder.Code(),
			Message: coder.String(),
			Data:    data,
		})
		return
	}
	c.JSON(http.StatusOK, Response{Data: data})
}
