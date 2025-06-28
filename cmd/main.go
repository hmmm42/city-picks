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
)

func main() {
	app, cleanup, err := InitApp()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	server := &http.Server{
		Addr:    ":" + config.ServerOptions.Port,
		Handler: app.Engine,
	}
	slog.Info("Listening on " + server.Addr)
	go func() {
		err = server.ListenAndServe()
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
