package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"codeatlas.dev/internal/content"
	"codeatlas.dev/internal/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	contentDir := envOrDefault("CONTENT_DIR", "content")
	templatesDir := envOrDefault("TEMPLATES_DIR", "templates")
	staticDir := envOrDefault("STATIC_DIR", "static")
	baseURL := envOrDefault("BASE_URL", envOrDefault("SITE_URL", "https://codeatlas.dev"))
	siteURL := strings.TrimRight(baseURL, "/")

	store, err := content.Load(contentDir)
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}

	app, err := web.New(store, web.Config{
		TemplatesDir: templatesDir,
		StaticDir:    staticDir,
		SiteURL:      siteURL,
	})
	if err != nil {
		return fmt.Errorf("initialize web app: %w", err)
	}

	port := envOrDefault("PORT", "8080")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("serving codeatlas.dev", "addr", srv.Addr, "guides", store.Count())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.Info("shutting down server")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	return <-errCh
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
