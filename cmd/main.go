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
)

//func init() {
//	defaultConfigPath := config.GetDefaultConfigPath()
//	configPath := pflag.StringP("config", "c", defaultConfigPath, "path to config file")
//	pflag.Parse()
//
//	config.InitConfig(*configPath)
//
//	var err error
//	persistent.DBEngine, err = persistent.NewMySQL(config.MySQLOptions)
//	if err != nil {
//		panic(err)
//	}
//	cache.RedisClient, err = cache.NewRedisClient(config.RedisOptions)
//	if err != nil {
//		panic(err)
//	}
//}

func main() {
	app, err := InitApp()
	if err != nil {
		panic(err)
	}
	server := &http.Server{
		Addr:    ":" + app.Config.Server.Port,
		Handler: app.Engine,
	}
	slog.Info("Listening on " + app.Config.Server.Port)
	go func() {
		err := server.ListenAndServe()
		//err = app.Engine.Run(config.ServerOptions.Port)
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
	if err = server.Shutdown(ctx); err != nil {
		panic(err)
	}
}
