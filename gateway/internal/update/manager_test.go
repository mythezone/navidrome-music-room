package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type memoryStateStore struct {
	values map[string][]byte
}

func (s *memoryStateStore) PutUpdateState(_ context.Context, key string, value any) error {
	if s.values == nil {
		s.values = map[string][]byte{}
	}
	payload, err := json.Marshal(value)
	if err == nil {
		s.values[key] = payload
	}
	return err
}

func (s *memoryStateStore) GetUpdateState(_ context.Context, key string, value any) (bool, error) {
	payload, ok := s.values[key]
	if !ok {
		return false, nil
	}
	return true, json.Unmarshal(payload, value)
}

func TestNormalizeAndSelectReleaseChannel(t *testing.T) {
	if normalizeChannel(" BETA ") != "beta" || normalizeChannel("nightly") != "stable" {
		t.Fatal("update channel normalization is not fail-closed")
	}
	releases := []Release{
		{TagName: "v9.0.0-draft", Draft: true},
		{TagName: "v2.0.0-beta.1", Prerelease: true},
		{TagName: "v1.9.0"},
	}
	beta, err := selectRelease(releases, "beta")
	if err != nil || beta.TagName != "v2.0.0-beta.1" {
		t.Fatalf("beta selection failed: %#v %v", beta, err)
	}
	stable, err := selectRelease(releases, "stable")
	if err != nil || stable.TagName != "v1.9.0" {
		t.Fatalf("stable selection failed: %#v %v", stable, err)
	}
	_, err = selectRelease([]Release{{TagName: "v2.0.0-beta.1", Prerelease: true}}, "stable")
	assertDomainError(t, err, "update_release_invalid")
}

func TestManagerRejectsInvalidRepositoryAndUntrustedAssets(t *testing.T) {
	if _, err := NewManager(Config{Repository: "../owner/repo"}, &memoryStateStore{}); err == nil {
		t.Fatal("unsafe repository was accepted")
	}
	if _, err := NewManager(Config{
		Repository: "owner/repo", CosignBinary: "cosign", IdentityRegex: ".*",
	}, &memoryStateStore{}); err == nil {
		t.Fatal("missing pinned trusted root was accepted")
	}
	manager, err := NewManager(Config{
		Repository: "owner/repo", CurrentVersion: "v1", CosignBinary: "cosign",
		TrustedRoot: "/trusted-root.json", IdentityRegex: "https://example.test/release/.*",
	}, &memoryStateStore{})
	if err != nil {
		t.Fatal(err)
	}
	err = manager.download(t.Context(), "https://example.com/release.tar.gz", filepath.Join(t.TempDir(), "release.tar.gz"), 1024)
	assertDomainError(t, err, "update_asset_url_invalid")
	for _, host := range []string{"github.com", "api.github.com", "objects.githubusercontent.com"} {
		if !trustedGitHubHost(host) {
			t.Fatalf("expected trusted GitHub host %q", host)
		}
	}
	if trustedGitHubHost("github.com.evil.example") {
		t.Fatal("lookalike GitHub host was trusted")
	}
}

func TestSigstoreVerificationUsesPinnedRootWithoutDeprecatedOfflineMode(t *testing.T) {
	dir := t.TempDir()
	arguments := filepath.Join(dir, "arguments.txt")
	fakeCosign := filepath.Join(dir, "cosign")
	if err := os.WriteFile(fakeCosign, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$COSIGN_ARGUMENTS\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COSIGN_ARGUMENTS", arguments)
	trustedRoot := filepath.Join(dir, "trusted-root.json")
	manager, err := NewManager(Config{
		Repository: "owner/repo", CurrentVersion: "v1", CosignBinary: fakeCosign,
		TrustedRoot: trustedRoot, IdentityRegex: "https://github\\.com/owner/repo/\\.github/workflows/release\\.yml@refs/tags/.*",
	}, &memoryStateStore{})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.verifySigstore(t.Context(), "/release/archive.tar.gz", "/release/archive.sigstore.json"); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(arguments)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSpace(string(payload)), "\n")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--trusted-root "+trustedRoot) || strings.Contains(joined, "--offline") {
		t.Fatalf("unexpected cosign arguments: %q", args)
	}
}

func TestChecksumLookupAndFileHash(t *testing.T) {
	dir := t.TempDir()
	asset := filepath.Join(dir, "bundle.tar.gz")
	if err := os.WriteFile(asset, []byte("signed release bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	hash, err := fileSHA256(asset)
	if err != nil {
		t.Fatal(err)
	}
	checksums := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(checksums, []byte(strings.ToUpper(hash)+"  *bundle.tar.gz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	found, err := checksumFor(checksums, "bundle.tar.gz")
	if err != nil || found != hash {
		t.Fatalf("checksum lookup failed: %q %v", found, err)
	}
	_, err = checksumFor(checksums, "other.tar.gz")
	assertDomainError(t, err, "update_checksum_missing")
}

func TestSignedMetadataValidatorsBindInventoryAndProvenance(t *testing.T) {
	dir := t.TempDir()
	sbomPath := filepath.Join(dir, "release.spdx.json")
	sbom := map[string]any{
		"spdxVersion": "SPDX-2.3", "dataLicense": "CC0-1.0",
		"documentNamespace": "https://example.invalid/spdx/test", "packages": []map[string]any{{"name": "gateway"}},
	}
	writeTestJSON(t, sbomPath, sbom)
	if err := validateSBOM(sbomPath); err != nil {
		t.Fatalf("valid SPDX inventory was rejected: %v", err)
	}
	writeTestJSON(t, sbomPath, map[string]any{
		"spdxVersion": "SPDX-2.3", "dataLicense": "CC0-1.0", "documentNamespace": "https://example.invalid/empty",
	})
	assertDomainError(t, validateSBOM(sbomPath), "update_sbom_invalid")

	provenancePath := filepath.Join(dir, "release.provenance.json")
	assetName := "navidrome-music-room-linux-amd64.tar.gz"
	digest := strings.Repeat("a", 64)
	provenance := map[string]any{
		"_type":         "https://in-toto.io/Statement/v1",
		"subject":       []map[string]any{{"name": assetName, "digest": map[string]string{"sha256": digest}}},
		"predicateType": "https://slsa.dev/provenance/v1",
		"predicate": map[string]any{
			"buildDefinition": map[string]any{
				"buildType": "https://github.com/Attestations/GitHubActionsWorkflow@v1",
				"externalParameters": map[string]any{"workflow": map[string]string{
					"repository": "owner/repo", "ref": "refs/tags/v1.2.3", "path": ".github/workflows/release.yml",
				}},
			},
			"runDetails": map[string]any{"builder": map[string]string{"id": "https://github.com/actions/runner"}},
		},
	}
	writeTestJSON(t, provenancePath, provenance)
	if err := validateProvenance(provenancePath, assetName, digest, "owner/repo", "v1.2.3"); err != nil {
		t.Fatalf("valid provenance was rejected: %v", err)
	}
	assertDomainError(t,
		validateProvenance(provenancePath, assetName, strings.Repeat("b", 64), "owner/repo", "v1.2.3"),
		"update_provenance_invalid",
	)
	assertDomainError(t,
		validateProvenance(provenancePath, assetName, digest, "owner/another", "v1.2.3"),
		"update_provenance_invalid",
	)
}

func TestExtractArchiveAllowsFilesAndRejectsTraversalAndLinks(t *testing.T) {
	dir := t.TempDir()
	validArchive := filepath.Join(dir, "valid.tar.gz")
	writeTestArchive(t, validArchive, []archiveEntry{
		{name: "release.json", body: []byte(`{"version":"v1"}`)},
		{name: "nested/music-room-gateway", body: []byte("binary")},
	})
	validDestination := filepath.Join(dir, "valid")
	if err := os.MkdirAll(validDestination, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := extractArchive(validArchive, validDestination); err != nil {
		t.Fatal(err)
	}
	if body, err := os.ReadFile(filepath.Join(validDestination, "nested", "music-room-gateway")); err != nil || string(body) != "binary" {
		t.Fatalf("valid archive did not extract: %q %v", body, err)
	}

	traversalArchive := filepath.Join(dir, "traversal.tar.gz")
	writeTestArchive(t, traversalArchive, []archiveEntry{{name: "../escaped", body: []byte("bad")}})
	err := extractArchive(traversalArchive, filepath.Join(dir, "traversal"))
	assertDomainError(t, err, "update_bundle_invalid")
	if _, err := os.Stat(filepath.Join(dir, "escaped")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("archive escaped staging directory: %v", err)
	}

	linkArchive := filepath.Join(dir, "link.tar.gz")
	writeTestArchive(t, linkArchive, []archiveEntry{{name: "link", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"}})
	err = extractArchive(linkArchive, filepath.Join(dir, "link"))
	assertDomainError(t, err, "update_bundle_invalid")
}

func TestWriteJSONAtomicUsesPrivatePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pending.json")
	if err := writeJSONAtomic(path, map[string]string{"state": "staged"}, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected pending file mode: %v %v", info, err)
	}
}

func TestRollbackRequestRequiresVersionAndDatabasePair(t *testing.T) {
	root := t.TempDir()
	releases := filepath.Join(root, "releases")
	versions := filepath.Join(releases, "versions")
	current := filepath.Join(versions, "v2")
	previous := filepath.Join(versions, "v1")
	for path, version := range map[string]string{current: "v2.0.0", previous: "v1.9.0"} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "release.json"), []byte(`{"version":"`+version+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(current, filepath.Join(releases, "current")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(previous, filepath.Join(releases, "previous")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "backups"), 0o700); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(root, "backups", "before-v2.sqlite3")
	if err := os.WriteFile(backup, []byte("sqlite backup"), 0o600); err != nil {
		t.Fatal(err)
	}
	activation := map[string]string{
		"newRelease": current, "previousRelease": previous, "databaseBackup": backup,
	}
	if err := writeJSONAtomic(filepath.Join(releases, "last-activation.json"), activation, 0o600); err != nil {
		t.Fatal(err)
	}
	state := &memoryStateStore{}
	manager, err := NewManager(Config{
		Repository: "owner/repo", CurrentVersion: "v2.0.0", DataDir: root,
		CosignBinary: "cosign", TrustedRoot: "/trusted-root.json", IdentityRegex: "https://example.test/release/.*",
	}, state)
	if err != nil {
		t.Fatal(err)
	}
	version, err := manager.RequestRollback(t.Context(), "stable")
	if err != nil || version != "v1.9.0" {
		t.Fatalf("rollback target was not resolved: %q %v", version, err)
	}
	var request map[string]any
	payload, err := os.ReadFile(filepath.Join(releases, "rollback-request.json"))
	if err != nil || json.Unmarshal(payload, &request) != nil || request["version"] != "v1.9.0" {
		t.Fatalf("rollback request is incomplete: %#v %v", request, err)
	}
	if err := os.Remove(backup); err != nil {
		t.Fatal(err)
	}
	_, err = manager.RequestRollback(t.Context(), "stable")
	assertDomainError(t, err, "rollback_unavailable")
}

type archiveEntry struct {
	name     string
	body     []byte
	typeflag byte
	linkname string
}

func writeTestArchive(t *testing.T, path string, entries []archiveEntry) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Typeflag: typeflag, Linkname: entry.linkname, Mode: 0o600}
		if typeflag == tar.TypeReg {
			header.Size = int64(len(entry.body))
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if len(entry.body) > 0 {
			if _, err := tarWriter.Write(entry.body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertDomainError(t *testing.T, err error, code string) {
	t.Helper()
	var roomError *domain.Error
	if !errors.As(err, &roomError) || roomError.Code != code {
		t.Fatalf("expected %s, got %T %v", code, err, err)
	}
}

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		latest, current string
		newer           bool
	}{
		{"1.2.0", "1.1.0-dev", true}, {"1.2.0", "1.2.0-beta.1", true},
		{"v1.2.0", "1.2.0", false}, {"1.2.0", "1.2.0", false}, {"1.1.0", "1.2.0", false},
		{"1.10.0", "1.9.9", true}, {"2.0.0", "1.99.99", true}, {"1.2.1", "1.2.0", true},
		{"1.2.0-beta.10", "1.2.0-beta.2", true}, {"1.2.0-beta.2", "1.2.0-beta.10", false},
		{"1.2.0-beta", "1.2.0-alpha", true}, {"1.2.0-rc.1", "1.2.0-beta.11", true},
		{"1.2.0-alpha.1", "1.2.0-alpha", true}, {"1.2.0-alpha", "1.2.0-alpha.1", false},
		{"1.2.0-alpha.beta", "1.2.0-alpha.1", true}, {"1.2.0-alpha.1", "1.2.0-alpha.beta", false},
		{"1.2.0-beta.1", "1.2.0", false}, {"1.2.0+build.2", "1.2.0+build.1", false},
		{"  v1.2.0  ", "  v1.1.0 ", true}, {"1.2.0", "dev", true}, {"v1.2.0", "bootstrap", true},
		{"", "dev", false}, {"  ", "bootstrap", false}, {"broken", "1.2.0", false},
		{"1.2.0", "broken", false}, {"1.2", "1.1.0", false}, {"01.2.0", "1.1.0", false},
		{"1.2.0-beta.01", "1.1.0", false}, {"1.2.0", "1.1.0-beta.01", false},
		{"1.2.0-", "1.1.0", false}, {"1.2.0-beta..1", "1.1.0", false},
		{"999999999999999999999.0.0", "100000000000000000000.0.0", true},
	}
	for _, tc := range cases {
		t.Run(tc.latest+"_vs_"+tc.current, func(t *testing.T) {
			if got := isNewerVersion(tc.latest, tc.current); got != tc.newer {
				t.Fatalf("isNewerVersion(%q,%q)=%v want %v", tc.latest, tc.current, got, tc.newer)
			}
		})
	}
}

type releaseTransport func(*http.Request) (*http.Response, error)

func (f releaseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCheckReleaseMetadataAndCachedStatus(t *testing.T) {
	storage := &memoryStateStore{}
	manager, err := NewManager(Config{Repository: "owner/repo", CurrentVersion: "1.1.0-dev", DataDir: t.TempDir(), CosignBinary: "cosign", TrustedRoot: "root", IdentityRegex: "identity"}, storage)
	if err != nil {
		t.Fatal(err)
	}
	manager.client.Transport = releaseTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			t.Fatalf("unexpected endpoint: %s", r.URL)
		}
		body, _ := json.Marshal(Release{TagName: "v1.2.0", HTMLURL: "https://github.com/owner/repo/releases/tag/v1.2.0", PublishedAt: "2026-11-18T00:00:00Z", Body: " \n" + strings.Repeat("新", 4100) + " \n"})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	})
	status, err := manager.Check(t.Context(), "stable")
	if err != nil || !status.UpdateAvailable || len([]rune(status.ReleaseNotes)) != 4000 || status.PublishedAt == "" || status.Prerelease {
		t.Fatalf("metadata: %+v %v", status, err)
	}
	manager.config.CurrentVersion = "1.2.0"
	if status = manager.Status(t.Context(), "stable"); status.UpdateAvailable {
		t.Fatal("cached status advertises installed version")
	}
	if status = manager.Status(t.Context(), "beta"); status.LatestVersion != "" || status.UpdateAvailable {
		t.Fatal("cached stable release reused for beta")
	}
	if got := releaseNotes(strings.Repeat("a", 3999) + " \ntrailing"); len(got) != 3999 {
		t.Fatal("truncated notes retain trailing whitespace")
	}
	if got := releaseNotes(" \nShort notes\n "); got != "Short notes" {
		t.Fatalf("untrimmed notes %q", got)
	}
}
