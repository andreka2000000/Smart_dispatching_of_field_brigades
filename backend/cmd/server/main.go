package main

import (
	"context"
	"errors"
	"log/slog"
	"magistral/backend/internal/application"
	"magistral/backend/internal/contract"
	"magistral/backend/internal/storage/postgres"
	"magistral/backend/internal/transport"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		slog.Error("server stopped", "error", e)
		os.Exit(1)
	}
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL is required")
	}
	boot, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	store, e := postgres.Open(boot, url)
	if e != nil {
		return e
	}
	defer store.Pool.Close()
	v, e := contract.New()
	if e != nil {
		return e
	}
	api := &transport.API{Service: &application.Service{Store: store}, Validator: v}
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{Addr: addr, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 65 * time.Second, WriteTimeout: 330 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		srv.Shutdown(shutdown)
	}()
	slog.Info("API ready", "address", addr)
	e = srv.ListenAndServe()
	if errors.Is(e, http.ErrServerClosed) {
		return nil
	}
	return e
}
