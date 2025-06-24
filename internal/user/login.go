package user

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/dal/query"
	"github.com/hmmm42/city-picks/internal/db"
	"github.com/hmmm42/city-picks/internal/middleware"
	"github.com/hmmm42/city-picks/pkg/code"
	"github.com/redis/go-redis/v9"
)

const (
	UserNickNamePrefix = "user"
	phoneKeyPrefix     = "phone:"
)

type LoginRequest struct {
	Phone       string `json:"phone" binding:"required"`
	CodeOrPwd   string `json:"code_or_pwd" binding:"required"`
	LoginMethod string `json:"login_method" binding:"required"` // "phone" or "password"
}

func GetVerificationCode(c *gin.Context) {
	phone := c.Param("phone")
	if phone == "" || !isPhoneValid(phone) {
		code.WriteResponse(c, code.ErrValidation, "phone is empty or invalid")
		return
	}

	num := rand.Intn(1000000) + 100000 // 6-digit code
	key := phoneKeyPrefix + phone

	success, err := db.RedisClient.SetNX(context.Background(), key, num, 10*time.Minute).Result()
	if !success || err != nil {
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}

	code.WriteResponse(c, code.ErrSuccess, gin.H{"VerificationCode": num})
}

func Login(c *gin.Context) {
	var login LoginRequest
	err := c.Bind(&login)
	if err != nil {
		slog.Error("code login bind error", "err", err)
		code.WriteResponse(c, code.ErrBind, nil)
		return
	}

	if !isPhoneValid(login.Phone) {
		code.WriteResponse(c, code.ErrValidation, "phone is empty or invalid")
		return
	}

	slog.Debug("code login", "data", login)
	switch login.LoginMethod {
	case "phone":
		loginCode(c, login)
	case "password":
		loginPassword(c, login)
	default:
		code.WriteResponse(c, code.ErrValidation, "invalid login_method")
	}
}

func isPhoneValid(phone string) bool {
	regRuler := `^1[1-9]\d{9}$`
	reg := regexp.MustCompile(regRuler)
	return reg.MatchString(phone)
}

func loginCode(c *gin.Context, login LoginRequest) {
	valid, err := db.RedisClient.Get(context.Background(), phoneKeyPrefix+login.Phone).Result()
	if errors.Is(err, redis.Nil) {
		code.WriteResponse(c, code.ErrValidation, "verification code expired or not found")
		return
	}
	if err != nil {
		slog.Error("redis get error", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	if valid != login.CodeOrPwd {
		code.WriteResponse(c, code.ErrValidation, "verification code is incorrect")
		return
	}
	u := query.TbUser
	count, err := u.Where(u.Phone.Eq(login.Phone)).Count()
	if err != nil {
		slog.Error("find by phone bad", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	if count == 0 {
		err = u.Create(&model.TbUser{
			Phone:    login.Phone,
			NickName: UserNickNamePrefix + strconv.Itoa(rand.Intn(100000)),
		})
		if err != nil {
			slog.Error("create user failed", "err", err)
			code.WriteResponse(c, code.ErrDatabase, "create user failed")
			return
		}
	}
	generateTokenResponse(c, login.Phone)
}

func loginPassword(c *gin.Context, login LoginRequest) {
	u := query.TbUser
	count, err := u.Where(u.Phone.Eq(login.Phone)).Count()
	if err != nil {
		slog.Error("find by phone and password bad", "err", err)
		code.WriteResponse(c, code.ErrDatabase, nil)
		return
	}
	if count == 0 {
		code.WriteResponse(c, code.ErrPasswordIncorrect, "phone or password is incorrect")
		return
	}
	generateTokenResponse(c, login.Phone)
}

func generateTokenResponse(c *gin.Context, phone string) {
	token, err := middleware.GenerateToken(phone)
	if err != nil {
		slog.Error("generate token failed", "err", err)
		code.WriteResponse(c, code.ErrTokenGenerationFailed, nil)
		return
	}
	code.WriteResponse(c, code.ErrSuccess, gin.H{
		"token": token,
	})
}
