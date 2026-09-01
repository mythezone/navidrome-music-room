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

	"github.com/mythezone/navidrome-music-room/gateway/internal/api"
	"github.com/mythezone/navidrome-music-room/gateway/internal/auth"
	"github.com/mythezone/navidrome-music-room/gateway/internal/config"
	"github.com/mythezone/navidrome-music-room/gateway/internal/store"
)

const restartExitCode = 75

func main() {
	os.Exit(run())
}

func run() int {
	_ = syscall.Umask(0o077)
	if len(os.Args) == 2 && os.Args[1] == "--healthcheck" {
		return healthcheck()
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		return 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	storage, err := store.Open(ctx, cfg.DatabasePath, cfg.DataDir)
	if err != nil {
		logger.Error("database failed", "error", err)
		return 1
	}
	defer storage.Close()
	navidrome := auth.NewNavidromeClient(cfg.NavidromeInternal)
	sessions := auth.NewSessionManager(
		storage, navidrome, cfg.SessionTTL, cfg.PluginLease, cfg.ExistingGrace,
		cfg.NavidromePublic.String(), cfg.GatewayPublic.String(),
	)
	roomServer, err := api.NewServer(cfg, storage, sessions, logger)
	if err != nil {
		logger.Error("server initialization failed", "error", err)
		return 1
	}
	restartRequested := make(chan struct{}, 1)
	roomServer.SetRestartCallback(func() {
		select {
		case restartRequested <- struct{}{}:
		default:
		}
	})
	httpServer := &http.Server{
		Addr: cfg.ListenAddress, Handler: roomServer.Handler(),
		ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 90 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go roomServer.RunClock(ctx)
	serverError := make(chan error, 1)
	go func() {
		logger.Info("gateway listening",
			"address", cfg.ListenAddress, "version", cfg.Version,
			"data_dir", cfg.DataDir, "pairing_token_file", cfg.DataDir+"/secrets/plugin-pairing-token",
		)
		serverError <- httpServer.ListenAndServe()
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	restart := false
	select {
	case signal := <-signals:
		logger.Info("shutdown requested", "signal", signal.String())
	case <-restartRequested:
		restart = true
		logger.Info("launcher restart requested")
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("gateway stopped unexpectedly", "error", err)
			return 1
		}
	}
	cancel()
	shutdownContext, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return 1
	}
	if restart {
		return restartExitCode
	}
	return 0
}

func healthcheck() int {
	endpoint := os.Getenv("MUSIC_ROOM_LAUNCHER_HEALTH_URL")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:4534/healthz"
	}
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(endpoint)
	if err != nil {
		return 1
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
