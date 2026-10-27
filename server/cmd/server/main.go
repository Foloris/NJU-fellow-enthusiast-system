package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"campus-fellow/server/internal/config"
	"campus-fellow/server/internal/httpapi"
	"campus-fellow/server/internal/mailer"
	"campus-fellow/server/internal/realtime"
	"campus-fellow/server/internal/repository"
	"campus-fellow/server/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	root := flag.String("root", ".", "project root containing .env and client/dist")
	migrateOnly := flag.Bool("migrate-only", false, "apply schema migrations and exit")
	flag.Parse()
	if err := os.Chdir(*root); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	repo, err := repository.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer repo.Close()
	if err = repo.Migrate(ctx); err != nil {
		return err
	}
	if *migrateOnly {
		slog.Info("MySQL migrations applied")
		return nil
	}
	if err = repo.Seed(ctx, cfg); err != nil {
		return err
	}
	if err = os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		return err
	}
	hub := realtime.New()
	defer hub.Close()
	var sender service.VerificationMailer
	if cfg.SMTPHost != "" {
		sender = mailer.NewSMTP(cfg)
	} else {
		slog.Warn("SMTP not configured; email-code registration is unavailable")
	}
	app := service.New(repo, service.NewTokens(cfg.JWTSecret), hub, service.WithEmailVerification(cfg.CampusEmailDomain, sender))
	server := &http.Server{Addr: cfg.Address, Handler: httpapi.New(app, hub, cfg), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	stopping, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() {
		slog.Info("Go API listening", "address", cfg.Address, "database", cfg.DBName)
		done <- server.ListenAndServe()
	}()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-stopping.Done():
	}
	hub.Close()
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return server.Shutdown(shutdown)
}
