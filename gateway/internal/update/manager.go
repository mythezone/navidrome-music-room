package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

var validRepository = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type StateStore interface {
	PutUpdateState(context.Context, string, any) error
	GetUpdateState(context.Context, string, any) (bool, error)
}

type Config struct {
	Repository     string
	CurrentVersion string
	DataDir        string
	CosignBinary   string
	TrustedRoot    string
	IdentityRegex  string
}

type Manager struct {
	mu     sync.Mutex
	config Config
	store  StateStore
	client *http.Client
}

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type Release struct {
	TagName     string  `json:"tag_name"`
	HTMLURL     string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
	Prerelease  bool    `json:"prerelease"`
	Draft       bool    `json:"draft"`
	Assets      []Asset `json:"assets"`
}

type Status struct {
	CurrentVersion    string    `json:"currentVersion"`
	Channel           string    `json:"channel"`
	LatestVersion     string    `json:"latestVersion,omitempty"`
	UpdateAvailable   bool      `json:"updateAvailable"`
	ReleaseURL        string    `json:"releaseURL,omitempty"`
	CheckedAt         time.Time `json:"checkedAt,omitempty"`
	StagedVersion     string    `json:"stagedVersion,omitempty"`
	RollbackVersion   string    `json:"rollbackVersion,omitempty"`
	RollbackAvailable bool      `json:"rollbackAvailable"`
	State             string    `json:"state"`
	LastError         string    `json:"lastError,omitempty"`
}

type Pending struct {
	Version               string    `json:"version"`
	StagingDir            string    `json:"stagingDir"`
	ArchiveSHA256         string    `json:"archiveSHA256"`
	SBOMSHA256            string    `json:"sbomSHA256"`
	ProvenanceSHA256      string    `json:"provenanceSHA256"`
	GatewaySHA256         string    `json:"gatewaySHA256"`
	CosignSHA256          string    `json:"cosignSHA256"`
	TrustedRootSHA256     string    `json:"trustedRootSHA256"`
	PluginSHA256          string    `json:"pluginSHA256"`
	ReleaseMetadataSHA256 string    `json:"releaseMetadataSHA256"`
	CreatedAt             time.Time `json:"createdAt"`
	GatewayBinary         string    `json:"gatewayBinary"`
	PluginPackage         string    `json:"pluginPackage"`
}

func NewManager(config Config, store StateStore) (*Manager, error) {
	if !validRepository.MatchString(config.Repository) {
		return nil, fmt.Errorf("invalid GitHub release repository")
	}
	if strings.TrimSpace(config.CosignBinary) == "" || strings.TrimSpace(config.TrustedRoot) == "" {
		return nil, fmt.Errorf("cosign binary and Sigstore trusted root are required")
	}
	if _, err := regexp.Compile("^" + config.IdentityRegex + "$"); err != nil {
		return nil, fmt.Errorf("invalid update signing identity: %w", err)
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 5 || !trustedGitHubHost(request.URL.Hostname()) {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	return &Manager{config: config, store: store, client: client}, nil
}

func (m *Manager) Status(ctx context.Context, channel string) Status {
	channel = normalizeChannel(channel)
	status := Status{CurrentVersion: m.config.CurrentVersion, Channel: channel, State: "idle"}
	_, _ = m.store.GetUpdateState(ctx, "status", &status)
	status.CurrentVersion = m.config.CurrentVersion
	status.Channel = channel
	return m.withRollback(status)
}

func (m *Manager) Check(ctx context.Context, channel string) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	channel = normalizeChannel(channel)
	release, err := m.latest(ctx, channel)
	if err != nil {
		status := m.Status(ctx, channel)
		status.State = "check_failed"
		status.LastError = err.Error()
		_ = m.store.PutUpdateState(ctx, "status", status)
		return status, err
	}
	status := Status{
		CurrentVersion: m.config.CurrentVersion, Channel: channel, LatestVersion: release.TagName,
		UpdateAvailable: normalizeVersion(release.TagName) != normalizeVersion(m.config.CurrentVersion),
		ReleaseURL:      release.HTMLURL, CheckedAt: time.Now().UTC(), State: "checked",
	}
	status = m.withRollback(status)
	_ = m.store.PutUpdateState(ctx, "status", status)
	return status, nil
}

func (m *Manager) Stage(ctx context.Context, requestedVersion, channel string) (Pending, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	channel = normalizeChannel(channel)
	release, err := m.latest(ctx, channel)
	if err != nil {
		return Pending{}, err
	}
	if requestedVersion != "" && normalizeVersion(requestedVersion) != normalizeVersion(release.TagName) {
		return Pending{}, domain.NewError(409, "update_version_changed", "Requested release is no longer the latest release in the configured channel")
	}
	assetName := fmt.Sprintf("navidrome-music-room-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	archiveAsset, ok := findAsset(release.Assets, assetName)
	if !ok {
		return Pending{}, domain.NewError(409, "update_asset_missing", "Release does not contain an asset for this platform")
	}
	checksumsAsset, ok := findAsset(release.Assets, "checksums.txt")
	if !ok {
		return Pending{}, domain.NewError(409, "update_checksum_missing", "Release does not contain checksums.txt")
	}
	checksumsBundleAsset, ok := findAsset(release.Assets, "checksums.txt.sigstore.json")
	if !ok {
		return Pending{}, domain.NewError(409, "update_signature_missing", "Release does not contain a signed checksums manifest")
	}
	bundleAsset, ok := findAsset(release.Assets, assetName+".sigstore.json")
	if !ok {
		return Pending{}, domain.NewError(409, "update_signature_missing", "Release does not contain a Sigstore bundle")
	}
	sbomName := assetName + ".spdx.json"
	sbomAsset, ok := findAsset(release.Assets, sbomName)
	if !ok {
		return Pending{}, domain.NewError(409, "update_sbom_missing", "Release does not contain an SPDX SBOM for this platform")
	}
	sbomBundleAsset, ok := findAsset(release.Assets, sbomName+".sigstore.json")
	if !ok {
		return Pending{}, domain.NewError(409, "update_sbom_missing", "Release does not contain a signed SPDX SBOM")
	}
	provenanceName := assetName + ".provenance.json"
	provenanceAsset, ok := findAsset(release.Assets, provenanceName)
	if !ok {
		return Pending{}, domain.NewError(409, "update_provenance_missing", "Release does not contain offline provenance for this platform")
	}
	provenanceBundleAsset, ok := findAsset(release.Assets, provenanceName+".sigstore.json")
	if !ok {
		return Pending{}, domain.NewError(409, "update_provenance_missing", "Release does not contain signed offline provenance")
	}
	versionDir := filepath.Join(m.config.DataDir, "releases", safeVersion(release.TagName))
	stagingDir := filepath.Join(versionDir, "staging")
	if err := os.RemoveAll(stagingDir); err != nil {
		return Pending{}, err
	}
	if err := os.MkdirAll(stagingDir, 0o700); err != nil {
		return Pending{}, err
	}
	archivePath := filepath.Join(versionDir, assetName)
	checksumsPath := filepath.Join(versionDir, "checksums.txt")
	checksumsBundlePath := filepath.Join(versionDir, "checksums.txt.sigstore.json")
	bundlePath := filepath.Join(versionDir, assetName+".sigstore.json")
	sbomPath := filepath.Join(versionDir, sbomName)
	sbomBundlePath := filepath.Join(versionDir, sbomName+".sigstore.json")
	provenancePath := filepath.Join(versionDir, provenanceName)
	provenanceBundlePath := filepath.Join(versionDir, provenanceName+".sigstore.json")
	for _, item := range []struct {
		asset    Asset
		path     string
		maxBytes int64
	}{
		{archiveAsset, archivePath, 256 << 20},
		{checksumsAsset, checksumsPath, 1 << 20},
		{checksumsBundleAsset, checksumsBundlePath, 4 << 20},
		{bundleAsset, bundlePath, 4 << 20},
		{sbomAsset, sbomPath, 32 << 20},
		{sbomBundleAsset, sbomBundlePath, 4 << 20},
		{provenanceAsset, provenancePath, 1 << 20},
		{provenanceBundleAsset, provenanceBundlePath, 4 << 20},
	} {
		if err := m.download(ctx, item.asset.URL, item.path, item.maxBytes); err != nil {
			return Pending{}, err
		}
	}
	if err := m.verifySigstore(ctx, checksumsPath, checksumsBundlePath); err != nil {
		return Pending{}, err
	}
	expected, err := checksumFor(checksumsPath, assetName)
	if err != nil {
		return Pending{}, err
	}
	actual, err := fileSHA256(archivePath)
	if err != nil {
		return Pending{}, err
	}
	if !strings.EqualFold(expected, actual) {
		return Pending{}, domain.ErrorWithDetails(409, "update_checksum_invalid", "Release archive checksum does not match", map[string]string{"expected": expected, "actual": actual})
	}
	if err := m.verifySigstore(ctx, archivePath, bundlePath); err != nil {
		return Pending{}, err
	}
	if err := m.verifySigstore(ctx, sbomPath, sbomBundlePath); err != nil {
		return Pending{}, domain.NewError(409, "update_sbom_invalid", "SPDX SBOM signature verification failed")
	}
	if err := validateSBOM(sbomPath); err != nil {
		return Pending{}, err
	}
	if err := m.verifySigstore(ctx, provenancePath, provenanceBundlePath); err != nil {
		return Pending{}, domain.NewError(409, "update_provenance_invalid", "Provenance signature verification failed")
	}
	if err := validateProvenance(provenancePath, assetName, actual, m.config.Repository, release.TagName); err != nil {
		return Pending{}, err
	}
	sbomSHA256, err := fileSHA256(sbomPath)
	if err != nil {
		return Pending{}, err
	}
	provenanceSHA256, err := fileSHA256(provenancePath)
	if err != nil {
		return Pending{}, err
	}
	if err := extractArchive(archivePath, stagingDir); err != nil {
		return Pending{}, err
	}
	gatewayPath := filepath.Join(stagingDir, "music-room-gateway")
	pluginPath := filepath.Join(stagingDir, "navidrome-music-room.ndp")
	cosignPath := filepath.Join(stagingDir, "cosign")
	trustedRootPath := filepath.Join(stagingDir, "sigstore-trusted-root.json")
	for _, required := range []string{gatewayPath, cosignPath, trustedRootPath, pluginPath, filepath.Join(stagingDir, "release.json")} {
		if info, err := os.Stat(required); err != nil || !info.Mode().IsRegular() {
			return Pending{}, domain.NewError(409, "update_bundle_invalid", "Release archive is missing required files")
		}
	}
	for _, executable := range []string{gatewayPath, cosignPath} {
		if err := os.Chmod(executable, 0o700); err != nil {
			return Pending{}, err
		}
	}
	gatewaySHA256, err := fileSHA256(gatewayPath)
	if err != nil {
		return Pending{}, err
	}
	cosignSHA256, err := fileSHA256(cosignPath)
	if err != nil {
		return Pending{}, err
	}
	trustedRootSHA256, err := fileSHA256(trustedRootPath)
	if err != nil {
		return Pending{}, err
	}
	pluginSHA256, err := fileSHA256(pluginPath)
	if err != nil {
		return Pending{}, err
	}
	releaseMetadataSHA256, err := fileSHA256(filepath.Join(stagingDir, "release.json"))
	if err != nil {
		return Pending{}, err
	}
	pending := Pending{
		Version: release.TagName, StagingDir: stagingDir, ArchiveSHA256: actual,
		SBOMSHA256: sbomSHA256, ProvenanceSHA256: provenanceSHA256,
		GatewaySHA256: gatewaySHA256, CosignSHA256: cosignSHA256, TrustedRootSHA256: trustedRootSHA256,
		PluginSHA256: pluginSHA256, ReleaseMetadataSHA256: releaseMetadataSHA256,
		CreatedAt:     time.Now().UTC(),
		GatewayBinary: gatewayPath, PluginPackage: pluginPath,
	}
	if err := writeJSONAtomic(filepath.Join(m.config.DataDir, "releases", "pending-update.json"), pending, 0o600); err != nil {
		return Pending{}, err
	}
	status := Status{
		CurrentVersion: m.config.CurrentVersion, Channel: channel, LatestVersion: release.TagName,
		UpdateAvailable: true, ReleaseURL: release.HTMLURL, CheckedAt: time.Now().UTC(),
		StagedVersion: release.TagName, State: "staged",
	}
	status = m.withRollback(status)
	_ = m.store.PutUpdateState(ctx, "status", status)
	return pending, nil
}

func (m *Manager) RequestRollback(ctx context.Context, channel string) (string, error) {
	targetVersion, err := m.rollbackVersion()
	if err != nil {
		return "", err
	}
	request := map[string]any{"requestedAt": time.Now().UTC(), "version": targetVersion}
	if err := writeJSONAtomic(filepath.Join(m.config.DataDir, "releases", "rollback-request.json"), request, 0o600); err != nil {
		return "", err
	}
	status := m.Status(ctx, channel)
	status.State = "rollback_requested"
	status.StagedVersion = targetVersion
	status.RollbackVersion = targetVersion
	status.RollbackAvailable = true
	if err := m.store.PutUpdateState(ctx, "status", status); err != nil {
		_ = os.Remove(filepath.Join(m.config.DataDir, "releases", "rollback-request.json"))
		return "", err
	}
	return targetVersion, nil
}

func (m *Manager) withRollback(status Status) Status {
	version, err := m.rollbackVersion()
	status.RollbackAvailable = err == nil
	status.RollbackVersion = version
	return status
}

func (m *Manager) rollbackVersion() (string, error) {
	releasesDir := filepath.Join(m.config.DataDir, "releases")
	versionsDir := filepath.Join(releasesDir, "versions")
	current, err := filepath.EvalSymlinks(filepath.Join(releasesDir, "current"))
	if err != nil || !pathInside(versionsDir, current) {
		return "", domain.NewError(409, "rollback_unavailable", "The current managed release cannot be resolved")
	}
	previous, err := filepath.EvalSymlinks(filepath.Join(releasesDir, "previous"))
	if err != nil || !pathInside(versionsDir, previous) {
		return "", domain.NewError(409, "rollback_unavailable", "No previous managed release is available")
	}
	var activation struct {
		NewRelease      string `json:"newRelease"`
		PreviousRelease string `json:"previousRelease"`
		DatabaseBackup  string `json:"databaseBackup"`
	}
	payload, err := os.ReadFile(filepath.Join(releasesDir, "last-activation.json"))
	if err != nil || len(payload) > 16<<10 || json.Unmarshal(payload, &activation) != nil ||
		!samePath(activation.NewRelease, current) || !samePath(activation.PreviousRelease, previous) ||
		!pathInside(filepath.Join(m.config.DataDir, "backups"), activation.DatabaseBackup) {
		return "", domain.NewError(409, "rollback_unavailable", "No verified database rollback point is available for the previous release")
	}
	if info, err := os.Stat(activation.DatabaseBackup); err != nil || !info.Mode().IsRegular() {
		return "", domain.NewError(409, "rollback_unavailable", "The verified database rollback point is unavailable")
	}
	metadata, err := os.ReadFile(filepath.Join(previous, "release.json"))
	if err != nil || len(metadata) > 8192 {
		return "", domain.NewError(409, "rollback_unavailable", "Previous release metadata is unavailable")
	}
	var release struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(metadata, &release) != nil {
		return "", domain.NewError(409, "rollback_unavailable", "Previous release metadata is invalid")
	}
	release.Version = strings.TrimSpace(release.Version)
	if release.Version == "" || len(release.Version) > 128 || strings.ContainsAny(release.Version, "\r\n\x00") {
		return "", domain.NewError(409, "rollback_unavailable", "Previous release version is invalid")
	}
	return release.Version, nil
}

func (m *Manager) latest(ctx context.Context, channel string) (Release, error) {
	channel = normalizeChannel(channel)
	endpoint := "https://api.github.com/repos/" + m.config.Repository + "/releases/latest"
	if channel == "beta" {
		endpoint = "https://api.github.com/repos/" + m.config.Repository + "/releases?per_page=20"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Release{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "navidrome-music-room/"+m.config.CurrentVersion)
	response, err := m.client.Do(request)
	if err != nil {
		return Release{}, domain.NewError(502, "update_check_failed", "GitHub Releases is unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return Release{}, domain.ErrorWithDetails(502, "update_check_failed", "GitHub Releases returned an unexpected response", map[string]int{"status": response.StatusCode})
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4<<20))
	if channel == "beta" {
		var releases []Release
		if err := decoder.Decode(&releases); err != nil {
			return Release{}, domain.NewError(502, "update_check_failed", "GitHub Releases returned malformed JSON")
		}
		return selectRelease(releases, channel)
	}
	var release Release
	if err := decoder.Decode(&release); err != nil {
		return Release{}, domain.NewError(502, "update_check_failed", "GitHub Releases returned malformed JSON")
	}
	return selectRelease([]Release{release}, channel)
}

func selectRelease(releases []Release, channel string) (Release, error) {
	channel = normalizeChannel(channel)
	for _, release := range releases {
		if release.TagName == "" || release.Draft {
			continue
		}
		if channel == "stable" && release.Prerelease {
			continue
		}
		return release, nil
	}
	return Release{}, domain.NewError(409, "update_release_invalid", "No eligible GitHub release exists in the configured channel")
}

func normalizeChannel(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "beta") {
		return "beta"
	}
	return "stable"
}

func (m *Manager) download(ctx context.Context, rawURL, destination string, maxBytes int64) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || !trustedGitHubHost(parsed.Hostname()) {
		return domain.NewError(409, "update_asset_url_invalid", "Release asset URL is not trusted")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "navidrome-music-room/"+m.config.CurrentVersion)
	response, err := m.client.Do(request)
	if err != nil {
		return domain.NewError(502, "update_download_failed", "Release asset download failed")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || response.ContentLength > maxBytes {
		return domain.NewError(502, "update_download_failed", "Release asset response is invalid")
	}
	part := destination + ".part"
	file, err := os.OpenFile(part, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, maxBytes+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || written > maxBytes {
		_ = os.Remove(part)
		return domain.NewError(502, "update_download_failed", "Release asset download was incomplete or too large")
	}
	return os.Rename(part, destination)
}

func (m *Manager) verifySigstore(ctx context.Context, archivePath, bundlePath string) error {
	command := exec.CommandContext(ctx, m.config.CosignBinary,
		"verify-blob", "--bundle", bundlePath, "--trusted-root", m.config.TrustedRoot,
		"--certificate-identity-regexp", "^"+m.config.IdentityRegex+"$",
		"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com", archivePath,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		return domain.ErrorWithDetails(409, "update_signature_invalid", "Sigstore verification failed", map[string]string{"diagnostic": sanitizeDiagnostic(string(output))})
	}
	return nil
}

type spdxDocument struct {
	SPDXVersion       string            `json:"spdxVersion"`
	DataLicense       string            `json:"dataLicense"`
	DocumentNamespace string            `json:"documentNamespace"`
	Packages          []json.RawMessage `json:"packages"`
	Files             []json.RawMessage `json:"files"`
}

func validateSBOM(path string) error {
	payload, err := readLimitedFile(path, 32<<20)
	if err != nil {
		return domain.NewError(409, "update_sbom_invalid", "SPDX SBOM cannot be read")
	}
	var document spdxDocument
	if json.Unmarshal(payload, &document) != nil ||
		!strings.HasPrefix(document.SPDXVersion, "SPDX-") ||
		document.DataLicense != "CC0-1.0" || strings.TrimSpace(document.DocumentNamespace) == "" ||
		(len(document.Packages) == 0 && len(document.Files) == 0) {
		return domain.NewError(409, "update_sbom_invalid", "Release SBOM is not a valid SPDX inventory")
	}
	return nil
}

type provenanceStatement struct {
	Type          string `json:"_type"`
	PredicateType string `json:"predicateType"`
	Subject       []struct {
		Name   string            `json:"name"`
		Digest map[string]string `json:"digest"`
	} `json:"subject"`
	Predicate struct {
		BuildDefinition struct {
			BuildType          string `json:"buildType"`
			ExternalParameters struct {
				Workflow struct {
					Repository string `json:"repository"`
					Ref        string `json:"ref"`
					Path       string `json:"path"`
				} `json:"workflow"`
			} `json:"externalParameters"`
		} `json:"buildDefinition"`
		RunDetails struct {
			Builder struct {
				ID string `json:"id"`
			} `json:"builder"`
		} `json:"runDetails"`
	} `json:"predicate"`
}

func validateProvenance(path, assetName, archiveSHA256, repository, tag string) error {
	payload, err := readLimitedFile(path, 1<<20)
	if err != nil {
		return domain.NewError(409, "update_provenance_invalid", "Release provenance cannot be read")
	}
	var statement provenanceStatement
	if json.Unmarshal(payload, &statement) != nil ||
		statement.Type != "https://in-toto.io/Statement/v1" ||
		statement.PredicateType != "https://slsa.dev/provenance/v1" ||
		statement.Predicate.BuildDefinition.BuildType != "https://github.com/Attestations/GitHubActionsWorkflow@v1" ||
		statement.Predicate.BuildDefinition.ExternalParameters.Workflow.Repository != repository ||
		statement.Predicate.BuildDefinition.ExternalParameters.Workflow.Ref != "refs/tags/"+tag ||
		statement.Predicate.BuildDefinition.ExternalParameters.Workflow.Path != ".github/workflows/release.yml" ||
		statement.Predicate.RunDetails.Builder.ID != "https://github.com/actions/runner" {
		return domain.NewError(409, "update_provenance_invalid", "Release provenance does not identify the expected tagged release workflow")
	}
	for _, subject := range statement.Subject {
		if subject.Name == assetName && strings.EqualFold(subject.Digest["sha256"], archiveSHA256) {
			return nil
		}
	}
	return domain.NewError(409, "update_provenance_invalid", "Release provenance is not bound to the selected archive digest")
}

func readLimitedFile(path string, maximum int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maximum {
		return nil, errors.New("file size is invalid")
	}
	payload, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil || int64(len(payload)) > maximum {
		return nil, errors.New("file exceeds limit")
	}
	return payload, nil
}

func extractArchive(archivePath, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return domain.NewError(409, "update_bundle_invalid", "Release archive is not valid gzip")
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	var extracted int64
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return domain.NewError(409, "update_bundle_invalid", "Release archive is corrupt")
		}
		clean := filepath.Clean(header.Name)
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return domain.NewError(409, "update_bundle_invalid", "Release archive contains an unsafe path")
		}
		target := filepath.Join(destination, clean)
		if !strings.HasPrefix(target, filepath.Clean(destination)+string(filepath.Separator)) {
			return domain.NewError(409, "update_bundle_invalid", "Release archive escapes the staging directory")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
		case tar.TypeReg:
			extracted += header.Size
			if header.Size < 0 || extracted > 512<<20 {
				return domain.NewError(409, "update_bundle_invalid", "Release archive is too large")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return err
			}
			output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(output, tarReader, header.Size)
			closeErr := output.Close()
			if copyErr != nil || closeErr != nil {
				return domain.NewError(409, "update_bundle_invalid", "Release archive extraction failed")
			}
		default:
			return domain.NewError(409, "update_bundle_invalid", "Release archive contains unsupported links or devices")
		}
	}
	return nil
}

func checksumFor(path, assetName string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == assetName && len(fields[0]) == 64 {
			if _, err := hex.DecodeString(fields[0]); err == nil {
				return strings.ToLower(fields[0]), nil
			}
		}
	}
	return "", domain.NewError(409, "update_checksum_missing", "checksums.txt does not contain the selected asset")
}

func fileSHA256(path string) (string, error) {
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

func writeJSONAtomic(path string, value any, mode os.FileMode) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(payload, '\n'), mode); err != nil {
		return err
	}
	if err := os.Chmod(temporary, mode); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func findAsset(assets []Asset, name string) (Asset, bool) {
	for _, asset := range assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return Asset{}, false
}

func trustedGitHubHost(host string) bool {
	host = strings.ToLower(host)
	return host == "github.com" || host == "api.github.com" || strings.HasSuffix(host, ".githubusercontent.com")
}

func pathInside(parent, child string) bool {
	if strings.TrimSpace(child) == "" {
		return false
	}
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

func samePath(left, right string) bool {
	leftPath, leftErr := filepath.Abs(left)
	rightPath, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && filepath.Clean(leftPath) == filepath.Clean(rightPath)
}

func safeVersion(version string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, version)
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func sanitizeDiagnostic(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 500 {
		return value[:500]
	}
	return value
}
