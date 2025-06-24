package middleware

import (
	"crypto/sha1"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hmmm42/city-picks/internal/config"
	"github.com/hmmm42/city-picks/pkg/code"
)

type UserClaims struct {
	Phone string
	jwt.RegisteredClaims
}

func getJWTSecret() []byte {
	return []byte(config.JWTOptions.Secret)
}

func GenerateToken(phone string) (string, error) {
	hash := sha1.New()
	hash.Write([]byte(phone))
	claims := UserClaims{
		Phone: string(hash.Sum(nil)),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(config.JWTOptions.Expire)),
			Issuer:    config.JWTOptions.Issuer,
			NotBefore: jwt.NewNumericDate(time.Now()), // 生效时间
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

func ParseToken(token string) (*UserClaims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &UserClaims{}, func(token *jwt.Token) (any, error) {
		return getJWTSecret(), nil
	})
	if err != nil {
		slog.Debug("Error parsing token", "err", err)
		return nil, err
	}
	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*UserClaims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}
	return nil, err
}

func JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Info("JWT middleware invoked", "method", c.Request.Method, "path", c.Request.URL.Path)
		ecode := code.ErrSuccess
		token := c.GetHeader("token")
		if token == "" {
			ecode = code.ErrInvalidAuthHeader
		} else {
			_, err := ParseToken(token)
			if err != nil {
				ecode = code.ErrTokenInvalid
			}
		}
		if ecode != code.ErrSuccess {
			code.WriteResponse(c, ecode, nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
