package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const generatedSecretBytes = 32

type Config struct {
	ListenAddress       string
	DataDir             string
	DatabasePath        string
	NavidromeInternal   *url.URL
	NavidromePublic     *url.URL
	GatewayPublic       *url.URL
	PluginPairingToken  string
	AllowedOrigins      []string
	TrustProxy          bool
	Version             string
	ReleaseRepository   string
	CosignBinary        string
	UpdateIdentity      string
	LicensePublicKey    string
	ManagedByLauncher   bool
	PluginLease         time.Duration
	ExistingGrace       time.Duration
	SessionTTL          time.Duration
	WebSocketTicketTTL  time.Duration
	EmptyRoomPauseDelay time.Duration
	ShutdownTimeout     time.Duration
}

func Load() (Config, error) {
	dataDir := strings.TrimSpace(os.Getenv("MUSIC_ROOM_DATA_DIR"))
	if dataDir == "" {
		dataDir = "./room-data"
	}
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return Config{}, fmt.Errorf("resolve data directory: %w", err)
	}
	for _, dir := range []string{"", "secrets", "backups", "releases", "logs"} {
		path := filepath.Join(absDataDir, dir)
		if err := os.MkdirAll(path, 0o700); err != nil {
			return Config{}, fmt.Errorf("create %s: %w", path, err)
		}
		if err := os.Chmod(path, 0o700); err != nil {
			return Config{}, fmt.Errorf("secure %s: %w", path, err)
		}
	}

	internal, err := requiredURL("MUSIC_ROOM_NAVIDROME_INTERNAL_URL")
	if err != nil {
		return Config{}, err
	}
	public, err := optionalURL("MUSIC_ROOM_NAVIDROME_PUBLIC_URL", internal.String())
	if err != nil {
		return Config{}, err
	}
	gatewayPublic, err := optionalURL("MUSIC_ROOM_PUBLIC_URL", "http://localhost:4534")
	if err != nil {
		return Config{}, err
	}
	pairingToken, err := loadOrCreateSecret(
		strings.TrimSpace(os.Getenv("MUSIC_ROOM_PLUGIN_PAIRING_TOKEN")),
		filepath.Join(absDataDir, "secrets", "plugin-pairing-token"),
	)
	if err != nil {
		return Config{}, fmt.Errorf("load plugin pairing token: %w", err)
	}

	repository := envOr("MUSIC_ROOM_RELEASE_REPOSITORY", "mythezone/navidrome-music-room")
	return Config{
		ListenAddress:       envOr("MUSIC_ROOM_LISTEN_ADDRESS", ":4534"),
		DataDir:             absDataDir,
		DatabasePath:        filepath.Join(absDataDir, "rooms.sqlite3"),
		NavidromeInternal:   internal,
		NavidromePublic:     public,
		GatewayPublic:       gatewayPublic,
		PluginPairingToken:  pairingToken,
		AllowedOrigins:      splitCSV(os.Getenv("MUSIC_ROOM_ALLOWED_ORIGINS")),
		TrustProxy:          envBool("MUSIC_ROOM_TRUST_PROXY", false),
		Version:             envOr("MUSIC_ROOM_VERSION", "dev"),
		ReleaseRepository:   repository,
		CosignBinary:        envOr("MUSIC_ROOM_COSIGN_BINARY", "cosign"),
		UpdateIdentity:      envOr("MUSIC_ROOM_UPDATE_IDENTITY", "https://github\\.com/"+regexp.QuoteMeta(repository)+"/\\.github/workflows/release\\.yml@refs/tags/.*"),
		LicensePublicKey:    strings.TrimSpace(os.Getenv("MUSIC_ROOM_LICENSE_PUBLIC_KEY")),
		ManagedByLauncher:   envBool("MUSIC_ROOM_MANAGED_BY_LAUNCHER", false),
		PluginLease:         envDuration("MUSIC_ROOM_PLUGIN_LEASE", 90*time.Second),
		ExistingGrace:       envDuration("MUSIC_ROOM_EXISTING_SESSION_GRACE", 60*time.Second),
		SessionTTL:          envDuration("MUSIC_ROOM_SESSION_TTL", 15*time.Minute),
		WebSocketTicketTTL:  envDuration("MUSIC_ROOM_WS_TICKET_TTL", 60*time.Second),
		EmptyRoomPauseDelay: envDuration("MUSIC_ROOM_EMPTY_PAUSE_DELAY", 15*time.Second),
		ShutdownTimeout:     envDuration("MUSIC_ROOM_SHUTDOWN_TIMEOUT", 15*time.Second),
	}, nil
}

func requiredURL(key string) (*url.URL, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil, fmt.Errorf("%s is required", key)
	}
	return parseHTTPURL(key, raw)
}

func optionalURL(key, fallback string) (*url.URL, error) {
	raw := envOr(key, fallback)
	return parseHTTPURL(key, raw)
}

func parseHTTPURL(key, raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(raw), "/"))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("%s must be an absolute http(s) URL", key)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed, nil
}

func loadOrCreateSecret(configured, path string) (string, error) {
	if configured != "" {
		if len(configured) < 32 {
			return "", errors.New("configured secret must contain at least 32 characters")
		}
		if err := os.WriteFile(path, []byte(configured+"\n"), 0o600); err != nil {
			return "", err
		}
		if err := os.Chmod(path, 0o600); err != nil {
			return "", err
		}
		return configured, nil
	}
	if value, err := os.ReadFile(path); err == nil {
		secret := strings.TrimSpace(string(value))
		if len(secret) < 32 {
			return "", errors.New("stored secret is too short")
		}
		return secret, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	raw := make([]byte, generatedSecretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return "", err
	}
	return secret, nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, raw := range strings.Split(value, ",") {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
