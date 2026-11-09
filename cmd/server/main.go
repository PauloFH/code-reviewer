package main

import (
	"context"
	"crypto/rand"
	"errors"
	"io/fs"
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
	"github.com/paulofh/code-reviewer/internal/infrastructure/platform/gitfetch"
	"github.com/paulofh/code-reviewer/internal/infrastructure/sandbox"
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
	slog.SetDefault(log)
	fail := func(msg string, err error) {
		log.Error(msg, slog.String("error", err.Error()))
		os.Exit(1)
	}

	repo, err := storage.Open(cfg.Database.Path, cfg.AIConfigEncKey, log)
	if errors.Is(err, storage.ErrNoEncryptionKey) {
		fail("opening database — set AI_CONFIG_ENC_KEY", err)
	}
	if err != nil {
		fail("opening database — is AI_CONFIG_ENC_KEY correct?", err)
	}
	defer repo.Close()

	if err := seedSuperUser(repo, cfg.Dashboard, log); err != nil {
		fail("seeding super user", err)
	}

	factory := platform.Factory{
		Timeout:         time.Duration(cfg.AI.TimeoutSeconds) * time.Second, // AI_TIMEOUT_SECONDS also bounds platform calls
		GitLabReposPath: cfg.Agentic.GitLabReposPath,
	}
	registry, err := loadClients(repo, factory, cfg.ClientsConfigPath, log)
	if err != nil {
		fail("loading platform clients", err)
	}

	var execSandbox ai.Sandbox
	if cfg.Sandbox.Image != "" {
		execSandbox = sandbox.NewDockerSandbox(
			cfg.Sandbox.Image, cfg.Sandbox.CPUs, cfg.Sandbox.Memory,
			time.Duration(cfg.Sandbox.TimeoutSeconds)*time.Second,
		)
		log.Info("execution sandbox enabled", slog.String("image", cfg.Sandbox.Image))
	}
	claudeOAuth := ai.NewClaudeOAuth()
	reviewer := ai.NewReviewer(repo.AIConfigs(), repo.Prompts(), claudeOAuth, cfg.AI, execSandbox, log)

	broadcaster := transport.NewEventBroadcaster()
	reviews := usecase.NewProcessMergeRequest(usecase.ReviewDeps{
		Registry: registry,
		Claims:   repo,
		Events:   repo,
		Users:    repo,
		Profiles: repo.ReviewProfiles(),
		Fetcher: gitfetch.NewGitWorkspaceFetcher(
			cfg.Agentic.WorkspaceDir, cfg.Agentic.CloneDepth,
			time.Duration(cfg.Agentic.TimeoutSeconds)*time.Second, log,
		),
		Reviewer:  reviewer,
		Publisher: broadcaster,
		Retry: usecase.RetryConfig{
			MaxAttempts: cfg.Retry.MaxAttempts,
			BaseDelay:   time.Duration(cfg.Retry.BaseDelaySeconds) * time.Second,
		},
		Log: log,
	})

	jwtSecret := []byte(cfg.Dashboard.JWTSecret)
	if len(jwtSecret) == 0 {
		log.Warn("DASHBOARD_JWT_SECRET is not set; using a random secret, so sessions end on restart")
		jwtSecret = make([]byte, 32)
		_, _ = rand.Read(jwtSecret) // never fails, see crypto/rand.Read
	}

	server := transport.NewServer(transport.Deps{
		Reviews:        reviews,
		Registry:       registry,
		Clients:        usecase.NewClientManagementService(repo, registry, factory, log),
		Users:          repo,
		Events:         repo,
		Prompts:        repo.Prompts(),
		ReviewProfiles: repo.ReviewProfiles(),
		AIConfigs:      repo.AIConfigs(),
		ClaudeOAuth:    claudeOAuth,
		ConfigTester:   reviewer,
		Broadcaster:    broadcaster,
		JWTSecret:      jwtSecret,
		Log:            log,
	})

	srv := &http.Server{
		Addr:        ":" + cfg.App.Port,
		Handler:     server.Handler,
		ReadTimeout: 15 * time.Second,
		// No write timeout: the SSE stream is long-lived and ends with its request context.
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	go purgeOldEvents(repo, log)

	go func() {
		log.Info("server starting", slog.String("addr", srv.Addr), slog.String("env", cfg.App.Env))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fail("server error", err)
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
	// Reviews run detached from their request; wait so a deploy does not drop one.
	server.Shutdown(ctx)
}

func seedSuperUser(repo *storage.SQLiteRepository, d config.DashboardConfig, log *slog.Logger) error {
	n, err := repo.CountUsers()
	if err != nil || n > 0 {
		return err
	}
	hash, err := transport.HashPassword(d.Password)
	if err != nil {
		return err
	}
	if _, err := repo.CreateUser(domain.User{Username: d.User, Password: hash, Role: domain.RoleSuper}); err != nil {
		return err
	}
	log.Info("seeded super user", slog.String("username", d.User))
	return nil
}

// loadClients seeds the database from clients.json on first boot and builds
// the live registry from the stored clients.
func loadClients(repo *storage.SQLiteRepository, factory platform.Factory, seedPath string, log *slog.Logger) (*platform.Registry, error) {
	count, err := repo.CountClients()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		seed, err := config.LoadClients(seedPath)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			log.Warn("could not load clients.json for seeding", slog.String("error", err.Error()))
		default:
			if err := repo.SeedClients(seed); err != nil {
				return nil, err
			}
			log.Info("seeded clients from clients.json", slog.Int("count", len(seed)))
		}
	}

	stored, err := repo.ListClients()
	if err != nil {
		return nil, err
	}
	registry := platform.NewRegistry()
	for _, c := range stored {
		pc, err := factory.Build(c)
		if err != nil {
			// A row saved before clients were validated must not take the whole service down.
			log.Error("skipping invalid client; fix it in the dashboard", slog.String("name", c.Name), slog.String("error", err.Error()))
			continue
		}
		registry.Register(pc)
		log.Info("registered platform client",
			slog.String("name", pc.ClientName()),
			slog.String("platform", string(pc.Platform())),
			slog.String("watching", pc.WatchedUsername()),
		)
	}
	return registry, nil
}

// purgeOldEvents enforces the 12-month retention promised by the privacy policy.
func purgeOldEvents(repo *storage.SQLiteRepository, log *slog.Logger) {
	for {
		if n, err := repo.PurgeEventsBefore(time.Now().AddDate(-1, 0, 0)); err != nil {
			log.Error("review events purge failed", slog.String("error", err.Error()))
		} else if n > 0 {
			log.Info("review events purged", slog.Int64("count", n))
		}
		time.Sleep(24 * time.Hour)
	}
}
