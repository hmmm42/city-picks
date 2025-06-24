package middleware

import (
	"testing"

	"github.com/hmmm42/city-picks/internal/config"
)

func Test_JWT(t *testing.T) {
	t.Log("Testing JWT secret retrieval")
	config.InitConfig(config.GetDefaultConfigPath())

	//t.Log(viper.Get("jwt.Secret"))
	t.Log(string(getJWTSecret()))
}
