package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/store"
	updatemanager "github.com/mythezone/navidrome-music-room/gateway/internal/update"
)

const restartExitCode = 75

type launcherConfig struct {
	dataDir      string
	bootstrapDir string
	pluginDir    string
	healthURL    string
	version      string
	startupWait  time.Duration
}

type activation struct {
	NewRelease      string `json:"newRelease"`
	PreviousRelease string `json:"previousRelease"`
	DatabaseBackup  string `json:"databaseBackup"`
}

type releaseMetadata struct {
	Version string `json:"version"`
}

func main() {
	os.Exit(run())
}

func run() int {
	_ = syscall.Umask(0o077)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := loadConfig()
	if err != nil {
		logger.Error("launcher configuration failed", "error", err)
		return 1
	}
	if err := ensureBootstrap(cfg); err != nil {
		logger.Error("bootstrap release failed", "error", err)
		return 1
	}
	signalContext, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	var lastActivation *activation
	for {
		current, err := currentRelease(cfg)
		if err != nil {
			logger.Error("resolve current release failed", "error", err)
			return 1
		}
		releaseVersion, err := versionForRelease(current)
		if err != nil {
			logger.Error("read release metadata failed", "release", current, "error", err)
			return 1
		}
		command := exec.Command(filepath.Join(current, "music-room-gateway"))
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Stdin = os.Stdin
		command.Env = withEnvironment(os.Environ(), "MUSIC_ROOM_MANAGED_BY_LAUNCHER", "true")
		command.Env = withEnvironment(command.Env, "MUSIC_ROOM_VERSION", releaseVersion)
		command.Env = withEnvironment(command.Env, "MUSIC_ROOM_COSIGN_BINARY", filepath.Join(current, "cosign"))
		command.Env = withEnvironment(command.Env, "MUSIC_ROOM_SIGSTORE_TRUSTED_ROOT", filepath.Join(current, "sigstore-trusted-root.json"))
		if err := command.Start(); err != nil {
			logger.Error("start gateway failed", "release", current, "error", err)
			if lastActivation != nil && rollbackActivation(cfg, *lastActivation, logger) == nil {
				lastActivation = nil
				continue
			}
			return 1
		}
		logger.Info("gateway started", "pid", command.Process.Pid, "release", current, "version", releaseVersion)
		if err := waitHealthy(signalContext, cfg.healthURL, cfg.startupWait, command); err != nil {
			logger.Error("gateway health check failed", "release", current, "error", err)
			terminate(command)
			_ = command.Wait()
			if signalContext.Err() != nil {
				return 0
			}
			if lastActivation != nil && rollbackActivation(cfg, *lastActivation, logger) == nil {
				lastActivation = nil
				continue
			}
			return 1
		}
		if lastActivation != nil {
			if err := promoteActivation(cfg); err != nil {
				logger.Error("persist healthy activation failed", "error", err)
				terminate(command)
				_ = command.Wait()
				if rollbackActivation(cfg, *lastActivation, logger) == nil {
					lastActivation = nil
					continue
				}
				return 1
			}
			logger.Info("release health confirmed", "release", lastActivation.NewRelease)
		}
		lastActivation = nil
		waitResult := make(chan error, 1)
		go func() { waitResult <- command.Wait() }()
		select {
		case <-signalContext.Done():
			terminate(command)
			<-waitResult
			return 0
		case err := <-waitResult:
			exitCode := command.ProcessState.ExitCode()
			if exitCode != restartExitCode {
				logger.Error("gateway exited", "exit_code", exitCode, "error", err)
				return exitCodeOrOne(exitCode)
			}
			activationResult, changed, err := processLauncherRequest(cfg, logger)
			if err != nil {
				logger.Error("release switch failed", "error", err)
				return 1
			}
			if changed {
				lastActivation = &activationResult
			}
		}
	}
}

func loadConfig() (launcherConfig, error) {
	dataDir := envOr("MUSIC_ROOM_DATA_DIR", "/data")
	absData, err := filepath.Abs(dataDir)
	if err != nil {
		return launcherConfig{}, err
	}
	bootstrap := envOr("MUSIC_ROOM_BOOTSTRAP_RELEASE_DIR", "/opt/music-room/release")
	absBootstrap, err := filepath.Abs(bootstrap)
	if err != nil {
		return launcherConfig{}, err
	}
	return launcherConfig{
		dataDir: absData, bootstrapDir: absBootstrap,
		pluginDir:   strings.TrimSpace(os.Getenv("MUSIC_ROOM_PLUGIN_INSTALL_DIR")),
		healthURL:   envOr("MUSIC_ROOM_LAUNCHER_HEALTH_URL", "http://127.0.0.1:4534/healthz"),
		version:     envOr("MUSIC_ROOM_VERSION", "bootstrap"),
		startupWait: durationOr("MUSIC_ROOM_LAUNCHER_STARTUP_TIMEOUT", 30*time.Second),
	}, nil
}

func ensureBootstrap(cfg launcherConfig) error {
	releasesDir := filepath.Join(cfg.dataDir, "releases")
	versionsDir := filepath.Join(releasesDir, "versions")
	if err := os.MkdirAll(versionsDir, 0o700); err != nil {
		return err
	}
	if _, err := os.Lstat(filepath.Join(releasesDir, "current")); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	versionDir := filepath.Join(versionsDir, safeName(cfg.version)+"-bootstrap")
	if err := copyRelease(cfg.bootstrapDir, versionDir); err != nil {
		return err
	}
	if err := switchLink(filepath.Join(releasesDir, "current"), versionDir); err != nil {
		return err
	}
	return installPlugin(cfg, filepath.Join(versionDir, "navidrome-music-room.ndp"))
}

func processLauncherRequest(cfg launcherConfig, logger *slog.Logger) (activation, bool, error) {
	rollbackPath := filepath.Join(cfg.dataDir, "releases", "rollback-request.json")
	if _, err := os.Stat(rollbackPath); err == nil {
		result, err := switchToPrevious(cfg, logger)
		if err != nil {
			return activation{}, false, err
		}
		if err := stageActivation(cfg, result); err != nil {
			_ = rollbackActivation(cfg, result, logger)
			return activation{}, false, err
		}
		if err := os.Remove(rollbackPath); err != nil {
			_ = rollbackActivation(cfg, result, logger)
			return activation{}, false, err
		}
		return result, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return activation{}, false, err
	}
	pendingPath := filepath.Join(cfg.dataDir, "releases", "pending-update.json")
	body, err := os.ReadFile(pendingPath)
	if errors.Is(err, os.ErrNotExist) {
		return activation{}, false, nil
	}
	if err != nil {
		return activation{}, false, err
	}
	var pending updatemanager.Pending
	if err := json.Unmarshal(body, &pending); err != nil {
		return activation{}, false, fmt.Errorf("decode pending update: %w", err)
	}
	staging, err := filepath.Abs(pending.StagingDir)
	if err != nil || !inside(filepath.Join(cfg.dataDir, "releases"), staging) {
		return activation{}, false, fmt.Errorf("pending staging directory is outside release storage")
	}
	if err := verifyStagedRelease(staging, pending); err != nil {
		return activation{}, false, err
	}
	current, err := currentRelease(cfg)
	if err != nil {
		return activation{}, false, err
	}
	backup, err := backupDatabase(cfg)
	if err != nil {
		return activation{}, false, err
	}
	versionDir := filepath.Join(cfg.dataDir, "releases", "versions", safeName(pending.Version)+"-"+time.Now().UTC().Format("20060102T150405Z"))
	if err := copyRelease(staging, versionDir); err != nil {
		return activation{}, false, err
	}
	if err := switchLink(filepath.Join(cfg.dataDir, "releases", "previous"), current); err != nil {
		return activation{}, false, err
	}
	if err := switchLink(filepath.Join(cfg.dataDir, "releases", "current"), versionDir); err != nil {
		return activation{}, false, err
	}
	if err := installPlugin(cfg, filepath.Join(versionDir, "navidrome-music-room.ndp")); err != nil {
		_ = switchLink(filepath.Join(cfg.dataDir, "releases", "current"), current)
		return activation{}, false, err
	}
	result := activation{NewRelease: versionDir, PreviousRelease: current, DatabaseBackup: backup}
	if err := stageActivation(cfg, result); err != nil {
		_ = rollbackActivation(cfg, result, logger)
		return activation{}, false, err
	}
	if err := os.Remove(pendingPath); err != nil {
		_ = rollbackActivation(cfg, result, logger)
		return activation{}, false, err
	}
	logger.Info("release switch staged", "release", versionDir, "previous", current, "database_backup", backup)
	return result, true, nil
}

func rollbackActivation(cfg launcherConfig, item activation, logger *slog.Logger) error {
	versionsDir := filepath.Join(cfg.dataDir, "releases", "versions")
	if item.NewRelease == "" || !inside(versionsDir, item.NewRelease) ||
		item.PreviousRelease == "" || !inside(versionsDir, item.PreviousRelease) {
		return fmt.Errorf("activation release paths are invalid")
	}
	if err := switchLink(filepath.Join(cfg.dataDir, "releases", "current"), item.PreviousRelease); err != nil {
		return err
	}
	if err := switchLink(filepath.Join(cfg.dataDir, "releases", "previous"), item.NewRelease); err != nil {
		return err
	}
	if err := installPlugin(cfg, filepath.Join(item.PreviousRelease, "navidrome-music-room.ndp")); err != nil {
		return err
	}
	if item.DatabaseBackup != "" {
		if err := restoreDatabase(cfg, item.DatabaseBackup); err != nil {
			return err
		}
	}
	_ = os.Remove(filepath.Join(cfg.dataDir, "releases", "pending-activation.json"))
	logger.Warn("release rolled back after failed health check", "release", item.PreviousRelease)
	return nil
}

func switchToPrevious(cfg launcherConfig, logger *slog.Logger) (activation, error) {
	last, err := readActivation(filepath.Join(cfg.dataDir, "releases", "last-activation.json"))
	if err != nil {
		return activation{}, fmt.Errorf("read rollback state: %w", err)
	}
	previousLink := filepath.Join(cfg.dataDir, "releases", "previous")
	previous, err := filepath.EvalSymlinks(previousLink)
	if err != nil {
		return activation{}, fmt.Errorf("no previous release is available: %w", err)
	}
	current, err := currentRelease(cfg)
	if err != nil {
		return activation{}, err
	}
	if !samePath(last.NewRelease, current) || !samePath(last.PreviousRelease, previous) ||
		!inside(filepath.Join(cfg.dataDir, "backups"), last.DatabaseBackup) {
		return activation{}, fmt.Errorf("rollback state does not match the active release")
	}
	if info, err := os.Stat(last.DatabaseBackup); err != nil || !info.Mode().IsRegular() {
		return activation{}, fmt.Errorf("rollback database backup is unavailable")
	}
	backup, err := backupDatabase(cfg)
	if err != nil {
		return activation{}, err
	}
	inverse := activation{NewRelease: previous, PreviousRelease: current, DatabaseBackup: backup}
	if err := switchLink(filepath.Join(cfg.dataDir, "releases", "current"), previous); err != nil {
		return activation{}, err
	}
	if err := switchLink(previousLink, current); err != nil {
		_ = rollbackActivation(cfg, inverse, logger)
		return activation{}, err
	}
	if err := installPlugin(cfg, filepath.Join(previous, "navidrome-music-room.ndp")); err != nil {
		_ = rollbackActivation(cfg, inverse, logger)
		return activation{}, err
	}
	if err := restoreDatabase(cfg, last.DatabaseBackup); err != nil {
		_ = rollbackActivation(cfg, inverse, logger)
		return activation{}, err
	}
	logger.Info("manual rollback staged", "release", previous, "restored_database_backup", last.DatabaseBackup, "forward_database_backup", backup)
	return inverse, nil
}

func backupDatabase(cfg launcherConfig) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	databasePath := filepath.Join(cfg.dataDir, "rooms.sqlite3")
	storage, err := store.Open(ctx, databasePath, cfg.dataDir)
	if err != nil {
		return "", err
	}
	defer storage.Close()
	return storage.Backup(ctx, "pre-launcher-switch")
}

func restoreDatabase(cfg launcherConfig, backup string) error {
	absBackup, err := filepath.Abs(backup)
	if err != nil || !inside(filepath.Join(cfg.dataDir, "backups"), absBackup) {
		return fmt.Errorf("database backup path is invalid")
	}
	destination := filepath.Join(cfg.dataDir, "rooms.sqlite3")
	if err := copyFile(absBackup, destination, 0o600); err != nil {
		return err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(destination + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func currentRelease(cfg launcherConfig) (string, error) {
	resolved, err := filepath.EvalSymlinks(filepath.Join(cfg.dataDir, "releases", "current"))
	if err != nil {
		return "", err
	}
	if !inside(filepath.Join(cfg.dataDir, "releases", "versions"), resolved) {
		return "", fmt.Errorf("current release symlink is outside versions directory")
	}
	return resolved, nil
}

func versionForRelease(releaseDir string) (string, error) {
	payload, err := os.ReadFile(filepath.Join(releaseDir, "release.json"))
	if err != nil {
		return "", err
	}
	var metadata releaseMetadata
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return "", fmt.Errorf("decode release.json: %w", err)
	}
	version := strings.TrimSpace(metadata.Version)
	if version == "" || len(version) > 128 || strings.ContainsAny(version, "\r\n\x00") {
		return "", fmt.Errorf("release.json contains an invalid version")
	}
	return version, nil
}

func installPlugin(cfg launcherConfig, source string) error {
	if cfg.pluginDir == "" {
		return nil
	}
	if err := os.MkdirAll(cfg.pluginDir, 0o755); err != nil {
		return err
	}
	destination := filepath.Join(cfg.pluginDir, "navidrome-music-room.ndp")
	if _, err := os.Stat(destination); err == nil {
		backupDir := filepath.Join(cfg.dataDir, "releases", "plugin-backups")
		if err := os.MkdirAll(backupDir, 0o700); err != nil {
			return err
		}
		backup := filepath.Join(backupDir, "navidrome-music-room-"+time.Now().UTC().Format("20060102T150405.000000000Z")+".ndp")
		if err := copyFile(destination, backup, 0o600); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	part := destination + ".new"
	if err := copyFile(source, part, 0o644); err != nil {
		return err
	}
	return os.Rename(part, destination)
}

func copyRelease(source, destination string) error {
	if err := os.MkdirAll(destination, 0o700); err != nil {
		return err
	}
	for _, name := range []string{"music-room-gateway", "cosign", "sigstore-trusted-root.json", "navidrome-music-room.ndp", "release.json"} {
		mode := os.FileMode(0o600)
		if name == "music-room-gateway" || name == "cosign" {
			mode = 0o700
		}
		if err := copyFile(filepath.Join(source, name), filepath.Join(destination, name), mode); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	temporary := destination + ".tmp"
	output, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	syncErr := output.Sync()
	closeErr := output.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("copy %s failed", filepath.Base(source))
	}
	if err := os.Chmod(temporary, mode); err != nil {
		return err
	}
	return os.Rename(temporary, destination)
}

func switchLink(link, target string) error {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	temporary := link + ".new"
	_ = os.Remove(temporary)
	if err := os.Symlink(absTarget, temporary); err != nil {
		return err
	}
	return os.Rename(temporary, link)
}

func waitHealthy(ctx context.Context, healthURL string, timeout time.Duration, command *exec.Cmd) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if command.ProcessState != nil && command.ProcessState.Exited() {
			return fmt.Errorf("gateway exited before becoming healthy")
		}
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		response, err := client.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == 200 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("health check timed out")
}

func terminate(command *exec.Cmd) {
	if command.Process == nil {
		return
	}
	_ = command.Process.Signal(syscall.SIGTERM)
	time.AfterFunc(8*time.Second, func() {
		_ = command.Process.Kill()
	})
}

func inside(parent, child string) bool {
	absParent, err := filepath.Abs(parent)
	if err != nil {
		return false
	}
	absChild, err := filepath.Abs(child)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(absParent, absChild)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func withEnvironment(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, item := range environment {
		if !strings.HasPrefix(item, prefix) {
			result = append(result, item)
		}
	}
	return append(result, prefix+value)
}

func stageActivation(cfg launcherConfig, item activation) error {
	return writeJSON(filepath.Join(cfg.dataDir, "releases", "pending-activation.json"), item)
}

func promoteActivation(cfg launcherConfig) error {
	releasesDir := filepath.Join(cfg.dataDir, "releases")
	pending := filepath.Join(releasesDir, "pending-activation.json")
	if _, err := readActivation(pending); err != nil {
		return err
	}
	return os.Rename(pending, filepath.Join(releasesDir, "last-activation.json"))
}

func readActivation(path string) (activation, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return activation{}, err
	}
	if len(payload) > 16<<10 {
		return activation{}, fmt.Errorf("activation state is too large")
	}
	var item activation
	if err := json.Unmarshal(payload, &item); err != nil {
		return activation{}, err
	}
	if item.NewRelease == "" || item.PreviousRelease == "" || item.DatabaseBackup == "" {
		return activation{}, fmt.Errorf("activation state is incomplete")
	}
	return item, nil
}

func writeJSON(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Chmod(temporary, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func safeName(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, value)
}

func samePath(left, right string) bool {
	leftPath, leftErr := filepath.Abs(left)
	rightPath, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && filepath.Clean(leftPath) == filepath.Clean(rightPath)
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationOr(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func exitCodeOrOne(code int) int {
	if code > 0 && code < 126 {
		return code
	}
	return 1
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func verifyStagedRelease(staging string, pending updatemanager.Pending) error {
	required := map[string]string{
		"music-room-gateway":         pending.GatewaySHA256,
		"cosign":                     pending.CosignSHA256,
		"sigstore-trusted-root.json": pending.TrustedRootSHA256,
		"navidrome-music-room.ndp":   pending.PluginSHA256,
		"release.json":               pending.ReleaseMetadataSHA256,
	}
	for name, expected := range required {
		path := filepath.Join(staging, name)
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("pending update is missing %s", name)
		}
		if len(expected) != sha256.Size*2 {
			return fmt.Errorf("pending update has no verified digest for %s", name)
		}
		actual, err := sha256File(path)
		if err != nil || !strings.EqualFold(actual, expected) {
			return fmt.Errorf("pending update digest changed for %s", name)
		}
	}
	return nil
}
