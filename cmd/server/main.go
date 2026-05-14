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
	"github.com/paulofh/code-reviewer/internal/domain"
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

	// ── Storage ───────────────────────────────────────────────────────────────
	repo, err := storage.New(cfg.Database.Path)
	if err != nil {
		log.Error("initialising database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer repo.Close()

	// ── Platform clients ──────────────────────────────────────────────────────
	// On first boot: if the clients table is empty, seed from clients.json.
	count, err := repo.CountClients()
	if err != nil {
		log.Error("counting clients in db", slog.String("error", err.Error()))
		os.Exit(1)
	}

	aiTimeout := time.Duration(cfg.AI.TimeoutSeconds) * time.Second

	if count == 0 {
		clientCfgs, err := config.LoadClients(cfg.ClientsConfigPath)
		if err != nil && !os.IsNotExist(err) {
			log.Warn("could not load clients.json for seeding", slog.String("error", err.Error()))
		} else if err == nil {
			domainClients := configClientsToDomain(clientCfgs)
			if err := repo.SeedFromConfigs(domainClients); err != nil {
				log.Error("seeding clients from config", slog.String("error", err.Error()))
				os.Exit(1)
			}
			log.Info("seeded clients from clients.json", slog.Int("count", len(domainClients)))
		}
	}

	// Load all clients from DB and build the platform registry.
	dbClients, err := repo.ListClients()
	if err != nil {
		log.Error("loading clients from db", slog.String("error", err.Error()))
		os.Exit(1)
	}

	factory := platform.NewFactory()
	registry := platform.NewRegistry()
	for _, c := range dbClients {
		pc, err := factory.BuildFromDomain(c, aiTimeout)
		if err != nil {
			log.Error("building platform client",
				slog.String("name", c.Name),
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
		registry.Register(pc)
		log.Info("registered platform client",
			slog.String("name", pc.ClientName()),
			slog.String("platform", string(pc.Platform())),
			slog.String("watching", pc.WatchedUsername()),
		)
	}

	// ── AI reviewer ───────────────────────────────────────────────────────────
	aiClient := ai.New(cfg.AI.BaseURL, cfg.AI.Token, cfg.AI.Model, aiTimeout, repo)

	// ── SSE broadcaster ───────────────────────────────────────────────────────
	broadcaster := transport.NewEventBroadcaster()

	// ── Usecases ──────────────────────────────────────────────────────────────
	uc := usecase.NewProcessMergeRequest(
		registry, repo, repo, aiClient, broadcaster,
		cfg.AI.PricePerMInputTokens, cfg.AI.PricePerMOutputTokens,
		log,
	)

	clientMgmtSvc := usecase.NewClientManagementService(
		repo, registry, factory, aiTimeout, log,
	)

	// ── HTTP handlers ─────────────────────────────────────────────────────────
	authHandler := transport.NewAuthHandler(
		cfg.Dashboard.User,
		cfg.Dashboard.Password,
		cfg.Dashboard.JWTSecret,
	)
	apiHandler := transport.NewAPIHandler(repo, clientMgmtSvc)
	promptHandler := transport.NewPromptHandler(repo)

	router := transport.NewRouter(uc, registry, authHandler, apiHandler, promptHandler, broadcaster, log)

	srv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: router,
		// SSE connections can be long-lived; write timeout must not kill them.
		// Set a large but finite value; individual SSE handlers respect ctx.Done().
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
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

// configClientsToDomain converts config.ClientConfig slice to domain.Client slice.
func configClientsToDomain(cfgs []config.ClientConfig) []domain.Client {
	out := make([]domain.Client, len(cfgs))
	for i, c := range cfgs {
		out[i] = domain.Client{
			Name:          c.Name,
			Platform:      domain.PlatformType(c.Platform),
			BaseURL:       c.BaseURL,
			Token:         c.Token,
			WebhookSecret: c.WebhookSecret,
			Username:      c.Username,
		}
	}
	return out
}
