package user

import (
	"log/slog"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/internal/middleware"
	"github.com/hmmm42/city-picks/internal/service"
	"github.com/hmmm42/city-picks/pkg/code"
)

type LoginRequest struct {
	Phone       string `json:"phone" binding:"required"`
	CodeOrPwd   string `json:"code_or_pwd" binding:"required"`
	LoginMethod string `json:"login_method" binding:"required"` // "phone" or "password"
}

type LoginHandler struct {
	userService service.UserService
}

func NewLoginHandler(userService service.UserService) *LoginHandler {
	return &LoginHandler{userService: userService}
}

func (h *LoginHandler) GetVerificationCode(c *gin.Context) {
	phone := c.Param("phone")
	if phone == "" || !isPhoneValid(phone) {
		code.WriteResponse(c, code.ErrValidation, "phone is empty or invalid")
		return
	}

	verificationCode, err := h.userService.GetVerificationCode(c.Request.Context(), phone)
	// 相比于传入 c, c.Request.Context() 不包含 http 请求细节, 更轻量
	if err != nil {
		code.WriteResponse(c, code.ErrDatabase, err)
		return
	}

	code.WriteResponse(c, code.ErrSuccess, gin.H{
		"VerificationCode": verificationCode,
	})
}

func (h *LoginHandler) Login(c *gin.Context) {
	var req LoginRequest
	err := c.Bind(&req)
	if err != nil {
		slog.Error("code login bind error", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

	if !isPhoneValid(req.Phone) {
		code.WriteResponse(c, code.ErrValidation, "phone is empty or invalid")
		return
	}

	var user *model.TbUser
	switch req.LoginMethod {
	case "phone":
		user, err = h.userService.LoginWithCode(c.Request.Context(), req.Phone, req.CodeOrPwd)
	case "password":
		user, err = h.userService.LoginWithPwd(c.Request.Context(), req.Phone, req.CodeOrPwd)
	default:
		code.WriteResponse(c, code.ErrValidation, "invalid login_method")
		return
	}

	if err != nil {
		// TODO: 精确的错误处理
		code.WriteResponse(c, code.ErrDatabase, err)
		return
	}

	token, err := middleware.GenerateToken(user.ID)
	if err != nil {
		slog.Error("generate token failed", "err", err)
		code.WriteResponse(c, code.ErrTokenGenerationFailed, nil)
		return
	}

	code.WriteResponse(c, code.ErrSuccess, gin.H{
		"token":   token,
		"user_id": user.ID,
	})
}

func isPhoneValid(phone string) bool {
	regRuler := `^1[1-9]\d{9}$`
	reg := regexp.MustCompile(regRuler)
	return reg.MatchString(phone)
}
