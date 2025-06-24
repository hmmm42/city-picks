package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hmmm42/city-picks/internal/config"
	"github.com/hmmm42/city-picks/internal/db"
	"github.com/hmmm42/city-picks/internal/router"
	"github.com/spf13/pflag"
)

func init() {
	defaultConfigPath := config.GetDefaultConfigPath()
	configPath := pflag.StringP("config", "c", defaultConfigPath, "path to config file")
	pflag.Parse()

	config.InitConfig(*configPath)

	var err error
	db.DBEngine, err = db.NewMySQL(config.MySQLOptions)
	if err != nil {
		panic(err)
	}
	db.RedisClient, err = db.NewRedisClient(config.RedisOptions)
	if err != nil {
		panic(err)
	}
}

func main() {
	r := router.NewRouter()
	server := &http.Server{
		Addr:    ":" + config.ServerOptions.Port,
		Handler: r,
	}
	slog.Info("Listening on " + config.ServerOptions.Port)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		panic(err)
	}
}
