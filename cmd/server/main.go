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

	"github.com/paulofh/code-reviewer/internal/config"
	"github.com/paulofh/code-reviewer/internal/infrastructure/ai"
	applogger "github.com/paulofh/code-reviewer/internal/infrastructure/logger"
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform"
	"github.com/paulofh/code-reviewer/internal/infrastructure/storage"
	transport "github.com/paulofh/code-reviewer/internal/transport/http"
	"github.com/paulofh/code-reviewer/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("loading config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log := applogger.New(cfg.Log.Level)

	// --- storage ---
	repo, err := storage.New(cfg.Database.Path)
	if err != nil {
		log.Error("initialising database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer repo.Close()

	// --- platform clients ---
	clientCfgs, err := config.LoadClients(cfg.ClientsConfigPath)
	if err != nil {
		log.Error("loading clients config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	aiTimeout := time.Duration(cfg.AI.TimeoutSeconds) * time.Second
	registry, err := platform.BuildAll(clientCfgs, aiTimeout)
	if err != nil {
		log.Error("building platform clients", slog.String("error", err.Error()))
		os.Exit(1)
	}

	for _, c := range registry.All() {
		log.Info("registered platform client",
			slog.String("name", c.ClientName()),
			slog.String("platform", string(c.Platform())),
			slog.String("watching", c.WatchedUsername()),
		)
	}

	// --- AI reviewer ---
	aiClient := ai.New(cfg.AI.BaseURL, cfg.AI.Token, cfg.AI.Model, aiTimeout)

	// --- usecase ---
	uc := usecase.NewProcessMergeRequest(registry, repo, aiClient, log)

	// --- HTTP server ---
	router := transport.NewRouter(uc, registry, log)
	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: aiTimeout + 30*time.Second, // AI call dominates write time
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server starting", slog.String("addr", srv.Addr), slog.String("env", cfg.App.Env))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gracefully")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", slog.String("error", err.Error()))
	}
}
