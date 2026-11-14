package main

import (
	"context"
	"errors"
	"log/slog"
	"magistral/backend/internal/application"
	"magistral/backend/internal/auth"
	"magistral/backend/internal/contract"
	"magistral/backend/internal/optimization"
	"magistral/backend/internal/routing"
	"magistral/backend/internal/storage/postgres"
	"magistral/backend/internal/transport"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	if target := os.Getenv("OPTIMIZER_URL"); target != "" {
		api.Service.Optimizer, e = optimization.New(target)
		if e != nil {
			return e
		}
	}
	if target := os.Getenv("OSRM_URL"); target != "" {
		api.Router, e = routing.New(target)
		if e != nil {
			return e
		}
	}
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	origins := os.Getenv("APP_ORIGINS")
	if origins == "" {
		origins = "http://127.0.0.1:3000,http://localhost:3000,http://127.0.0.1:3001,http://localhost:3001"
	}
	srv := &http.Server{Addr: addr, Handler: api.ApplicationHandler(&auth.Manager{Store: store}, strings.Split(origins, ","), os.Getenv("COOKIE_SECURE") == "true"), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 65 * time.Second, WriteTimeout: 330 * time.Second, IdleTimeout: 60 * time.Second}
	// No host port for this listener in Compose. Evaluation and explicit local
	// import tools retain their anonymous organizer contract on the private net.
	var evaluation *http.Server
	if internal := os.Getenv("EVALUATION_ADDR"); internal != "" {
		evaluation = &http.Server{Addr: internal, Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 65 * time.Second, WriteTimeout: 330 * time.Second}
		go func() {
			if err := evaluation.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("evaluation listener failed", "error", err)
				stop()
			}
		}()
	}
	go func() {
		<-ctx.Done()
		shutdown, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		srv.Shutdown(shutdown)
		if evaluation != nil {
			evaluation.Shutdown(shutdown)
		}
	}()
	slog.Info("API ready", "address", addr)
	e = srv.ListenAndServe()
	if errors.Is(e, http.ErrServerClosed) {
		return nil
	}
	return e
}
