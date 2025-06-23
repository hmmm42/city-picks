package main

import (
	"log/slog"

	"github.com/hmmm42/city-picks/internal/config"
	"github.com/hmmm42/city-picks/pkg/logger"
	"github.com/spf13/pflag"
)

func init() {
	defaultConfigPath := config.GetDefaultConfigPath()
	configPath := pflag.StringP("config", "c", defaultConfigPath, "path to config file")
	pflag.Parse()

	config.InitConfig(*configPath)
	logger.InitLogger(config.LogOptions)
}

func main() {
	slog.Info("test", "user_id", 123)
	slog.Error(config.MySQLOptions.DBName)
}
