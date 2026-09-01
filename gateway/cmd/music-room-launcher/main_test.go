package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mythezone/navidrome-music-room/gateway/internal/store"
	updatemanager "github.com/mythezone/navidrome-music-room/gateway/internal/update"
)

func TestVersionForRelease(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "release.json"), []byte(`{"version":" v1.2.3 "}`), 0o600); err != nil {
		t.Fatal(err)
	}
	version, err := versionForRelease(dir)
	if err != nil || version != "v1.2.3" {
		t.Fatalf("unexpected version: %q %v", version, err)
	}
	for index, body := range []string{`{}`, `{"version":"line\nbreak"}`, `not-json`} {
		if err := os.WriteFile(filepath.Join(dir, "release.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := versionForRelease(dir); err == nil {
			t.Fatalf("invalid metadata case %d was accepted", index)
		}
	}
}

func TestCopyReleaseAndPluginInstallUseExpectedPermissions(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"music-room-gateway":         "gateway",
		"cosign":                     "cosign",
		"sigstore-trusted-root.json": `{"mediaType":"application/vnd.dev.sigstore.trustedroot+json;version=0.1"}`,
		"navidrome-music-room.ndp":   "plugin-v1",
		"release.json":               `{"version":"v1"}`,
	} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(body), 0o777); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyRelease(source, destination); err != nil {
		t.Fatal(err)
	}
	assertMode(t, filepath.Join(destination, "music-room-gateway"), 0o700)
	assertMode(t, filepath.Join(destination, "cosign"), 0o700)
	assertMode(t, filepath.Join(destination, "sigstore-trusted-root.json"), 0o600)
	assertMode(t, filepath.Join(destination, "navidrome-music-room.ndp"), 0o600)

	cfg := launcherConfig{dataDir: root, pluginDir: filepath.Join(root, "plugins")}
	pluginSource := filepath.Join(destination, "navidrome-music-room.ndp")
	if err := installPlugin(cfg, pluginSource); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pluginSource, []byte("plugin-v2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := installPlugin(cfg, pluginSource); err != nil {
		t.Fatal(err)
	}
	installed, err := os.ReadFile(filepath.Join(cfg.pluginDir, "navidrome-music-room.ndp"))
	if err != nil || string(installed) != "plugin-v2" {
		t.Fatalf("new plugin was not installed: %q %v", installed, err)
	}
	backups, err := filepath.Glob(filepath.Join(root, "releases", "plugin-backups", "*.ndp"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one plugin backup, got %v %v", backups, err)
	}
	backup, err := os.ReadFile(backups[0])
	if err != nil || string(backup) != "plugin-v1" {
		t.Fatalf("plugin backup is incorrect: %q %v", backup, err)
	}
}

func TestVerifyStagedReleaseRejectsChangesAfterSignatureVerification(t *testing.T) {
	staging := filepath.Join(t.TempDir(), "staging")
	createTestRelease(t, staging, "v2", "plugin-v2")
	gatewayDigest := mustSHA256(t, filepath.Join(staging, "music-room-gateway"))
	pluginDigest := mustSHA256(t, filepath.Join(staging, "navidrome-music-room.ndp"))
	metadataDigest := mustSHA256(t, filepath.Join(staging, "release.json"))
	cosignDigest := mustSHA256(t, filepath.Join(staging, "cosign"))
	trustedRootDigest := mustSHA256(t, filepath.Join(staging, "sigstore-trusted-root.json"))
	pending := updatemanager.Pending{
		GatewaySHA256: gatewayDigest, CosignSHA256: cosignDigest, TrustedRootSHA256: trustedRootDigest,
		PluginSHA256: pluginDigest, ReleaseMetadataSHA256: metadataDigest,
	}
	if err := verifyStagedRelease(staging, pending); err != nil {
		t.Fatalf("verified staging directory was rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staging, "navidrome-music-room.ndp"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyStagedRelease(staging, pending); err == nil {
		t.Fatal("modified plugin package was accepted after staging")
	}
}

func TestPathAndEnvironmentHelpers(t *testing.T) {
	parent := t.TempDir()
	if !inside(parent, filepath.Join(parent, "nested", "file")) || inside(parent, filepath.Join(filepath.Dir(parent), "sibling")) {
		t.Fatal("inside helper did not enforce the parent boundary")
	}
	environment := withEnvironment([]string{"A=1", "TARGET=old", "TARGET_EXTRA=keep"}, "TARGET", "new")
	joined := strings.Join(environment, "|")
	if strings.Contains(joined, "TARGET=old") || !strings.Contains(joined, "TARGET=new") || !strings.Contains(joined, "TARGET_EXTRA=keep") {
		t.Fatalf("environment replacement failed: %v", environment)
	}
}

func TestManualRollbackRestoresMatchingDatabaseAndCanRecoverForward(t *testing.T) {
	root := t.TempDir()
	cfg := launcherConfig{dataDir: root, pluginDir: filepath.Join(root, "plugins")}
	versions := filepath.Join(root, "releases", "versions")
	oldRelease := filepath.Join(versions, "v1")
	newRelease := filepath.Join(versions, "v2")
	createTestRelease(t, oldRelease, "v1", "plugin-v1")
	createTestRelease(t, newRelease, "v2", "plugin-v2")
	if err := switchLink(filepath.Join(root, "releases", "current"), newRelease); err != nil {
		t.Fatal(err)
	}
	if err := switchLink(filepath.Join(root, "releases", "previous"), oldRelease); err != nil {
		t.Fatal(err)
	}
	if err := installPlugin(cfg, filepath.Join(newRelease, "navidrome-music-room.ndp")); err != nil {
		t.Fatal(err)
	}

	storage, err := store.Open(t.Context(), filepath.Join(root, "rooms.sqlite3"), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.PutUpdateState(t.Context(), "schema-marker", "old-database"); err != nil {
		t.Fatal(err)
	}
	oldBackup, err := storage.Backup(t.Context(), "before-v2")
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.PutUpdateState(t.Context(), "schema-marker", "new-database"); err != nil {
		t.Fatal(err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	last := activation{NewRelease: newRelease, PreviousRelease: oldRelease, DatabaseBackup: oldBackup}
	if err := writeJSON(filepath.Join(root, "releases", "last-activation.json"), last); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inverse, err := switchToPrevious(cfg, logger)
	if err != nil {
		t.Fatal(err)
	}
	assertCurrentRelease(t, cfg, oldRelease)
	assertDatabaseMarker(t, cfg, "old-database")
	installed, _ := os.ReadFile(filepath.Join(cfg.pluginDir, "navidrome-music-room.ndp"))
	if string(installed) != "plugin-v1" {
		t.Fatalf("old plugin was not restored: %q", installed)
	}

	if err := rollbackActivation(cfg, inverse, logger); err != nil {
		t.Fatal(err)
	}
	assertCurrentRelease(t, cfg, newRelease)
	assertDatabaseMarker(t, cfg, "new-database")
}

func TestActivationPromotionIsAtomic(t *testing.T) {
	cfg := launcherConfig{dataDir: t.TempDir()}
	if err := os.MkdirAll(filepath.Join(cfg.dataDir, "releases"), 0o700); err != nil {
		t.Fatal(err)
	}
	item := activation{NewRelease: "/release/v2", PreviousRelease: "/release/v1", DatabaseBackup: "/backup/v1"}
	if err := stageActivation(cfg, item); err != nil {
		t.Fatal(err)
	}
	if err := promoteActivation(cfg); err != nil {
		t.Fatal(err)
	}
	stored, err := readActivation(filepath.Join(cfg.dataDir, "releases", "last-activation.json"))
	if err != nil || stored != item {
		t.Fatalf("activation was not promoted: %#v %v", stored, err)
	}
	if _, err := os.Stat(filepath.Join(cfg.dataDir, "releases", "pending-activation.json")); !os.IsNotExist(err) {
		t.Fatalf("pending activation remained after promotion: %v", err)
	}
}

func createTestRelease(t *testing.T, path, version, plugin string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"music-room-gateway":         "gateway-" + version,
		"cosign":                     "cosign-" + version,
		"sigstore-trusted-root.json": `{"mediaType":"application/vnd.dev.sigstore.trustedroot+json;version=0.1"}`,
		"navidrome-music-room.ndp":   plugin,
		"release.json":               `{"version":"` + version + `"}`,
	} {
		if err := os.WriteFile(filepath.Join(path, name), []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
	}
}

func assertCurrentRelease(t *testing.T, cfg launcherConfig, expected string) {
	t.Helper()
	current, err := currentRelease(cfg)
	if err != nil || !samePath(current, expected) {
		t.Fatalf("unexpected current release: %q expected=%q err=%v", current, expected, err)
	}
}

func assertDatabaseMarker(t *testing.T, cfg launcherConfig, expected string) {
	t.Helper()
	storage, err := store.Open(t.Context(), filepath.Join(cfg.dataDir, "rooms.sqlite3"), cfg.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	var marker string
	found, err := storage.GetUpdateState(t.Context(), "schema-marker", &marker)
	if err != nil || !found || marker != expected {
		t.Fatalf("unexpected database marker: %q found=%v err=%v", marker, found, err)
	}
}

func assertMode(t *testing.T, path string, expected os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != expected {
		t.Fatalf("unexpected mode for %s: %v %v", path, info, err)
	}
}

func mustSHA256(t *testing.T, path string) string {
	t.Helper()
	digest, err := sha256File(path)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
