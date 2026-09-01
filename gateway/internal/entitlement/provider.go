package entitlement

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	Format          = "musicmate-license/v1"
	defaultFile     = "secrets/license.json"
	maximumFileSize = 64 << 10
)

var commercialFeatures = []string{"chat", "stickers", "vip", "statistics", "rankings", "achievements"}

type envelope struct {
	Format    string `json:"format"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

type Claims struct {
	LicenseID string    `json:"licenseID"`
	Subject   string    `json:"subject"`
	IssuedAt  time.Time `json:"issuedAt"`
	NotBefore time.Time `json:"notBefore"`
	ExpiresAt time.Time `json:"expiresAt"`
	Features  []string  `json:"features"`
}

type Status struct {
	State               string     `json:"state"`
	OfflineVerification bool       `json:"offlineVerification"`
	LicenseID           string     `json:"licenseID,omitempty"`
	Subject             string     `json:"subject,omitempty"`
	ExpiresAt           *time.Time `json:"expiresAt,omitempty"`
	GrantedFeatures     []string   `json:"grantedFeatures,omitempty"`
	Reason              string     `json:"reason,omitempty"`
}

type Provider struct {
	dataDir   string
	publicKey ed25519.PublicKey
	keyError  error
	now       func() time.Time
}

func NewProvider(dataDir, encodedPublicKey string) *Provider {
	key, err := parsePublicKey(strings.TrimSpace(encodedPublicKey))
	return &Provider{
		dataDir: dataDir, publicKey: key, keyError: err,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (p *Provider) Verify(configuredPath string) Status {
	path, err := p.resolve(configuredPath)
	if err != nil {
		return invalid("invalid_path")
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return Status{State: "not_installed", OfflineVerification: true}
	}
	if err != nil {
		return invalid("unreadable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maximumFileSize {
		return invalid("invalid_file")
	}
	payload, err := io.ReadAll(io.LimitReader(file, maximumFileSize+1))
	if err != nil || len(payload) > maximumFileSize {
		return invalid("invalid_file")
	}
	if p.keyError != nil || len(p.publicKey) != ed25519.PublicKeySize {
		return Status{State: "verification_unavailable", OfflineVerification: true, Reason: "public_key_not_configured"}
	}

	var document envelope
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&document) != nil || strings.TrimSpace(document.Format) != Format {
		return invalid("malformed")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return invalid("malformed")
	}
	claimsPayload, err := decodeBase64(document.Payload)
	if err != nil || len(claimsPayload) == 0 || len(claimsPayload) > 32<<10 {
		return invalid("malformed_payload")
	}
	signature, err := decodeBase64(document.Signature)
	if err != nil || !ed25519.Verify(p.publicKey, claimsPayload, signature) {
		return invalid("signature_invalid")
	}
	var claims Claims
	claimsDecoder := json.NewDecoder(bytes.NewReader(claimsPayload))
	claimsDecoder.DisallowUnknownFields()
	if claimsDecoder.Decode(&claims) != nil {
		return invalid("claims_invalid")
	}
	if err := validateClaims(claims); err != nil {
		return invalid("claims_invalid")
	}
	now := p.now()
	status := Status{
		State: "valid", OfflineVerification: true, LicenseID: claims.LicenseID,
		Subject: claims.Subject, ExpiresAt: timePointer(claims.ExpiresAt),
		GrantedFeatures: slices.Clone(claims.Features),
	}
	if now.Before(claims.NotBefore) {
		status.State = "not_yet_valid"
		status.Reason = "not_before"
	} else if !claims.ExpiresAt.After(now) {
		status.State = "expired"
		status.Reason = "expired"
	}
	return status
}

func (p *Provider) resolve(configured string) (string, error) {
	value := strings.TrimSpace(configured)
	if value == "" {
		value = defaultFile
	}
	if filepath.IsAbs(value) {
		return "", errors.New("license path must be relative to room-data")
	}
	root, err := filepath.Abs(p.dataDir)
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(root, filepath.Clean(value)))
	if err != nil || !inside(root, path) {
		return "", errors.New("license path escapes room-data")
	}
	if resolved, resolveErr := filepath.EvalSymlinks(path); resolveErr == nil {
		if !inside(root, resolved) {
			return "", errors.New("license symlink escapes room-data")
		}
		path = resolved
	} else if !errors.Is(resolveErr, os.ErrNotExist) {
		return "", resolveErr
	}
	return path, nil
}

func parsePublicKey(value string) (ed25519.PublicKey, error) {
	if value == "" {
		return nil, errors.New("license public key is not configured")
	}
	if block, _ := pem.Decode([]byte(value)); block != nil {
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		key, ok := parsed.(ed25519.PublicKey)
		if !ok {
			return nil, errors.New("license public key must use Ed25519")
		}
		return key, nil
	}
	decoded, err := decodeBase64(value)
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		return nil, errors.New("license public key must be a base64 Ed25519 public key")
	}
	return ed25519.PublicKey(decoded), nil
}

func decodeBase64(value string) ([]byte, error) {
	for _, encoding := range []*base64.Encoding{
		base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding,
	} {
		if decoded, err := encoding.DecodeString(strings.TrimSpace(value)); err == nil {
			return decoded, nil
		}
	}
	return nil, errors.New("invalid base64")
}

func validateClaims(claims Claims) error {
	if strings.TrimSpace(claims.LicenseID) == "" || len(claims.LicenseID) > 128 ||
		strings.TrimSpace(claims.Subject) == "" || len(claims.Subject) > 256 ||
		claims.IssuedAt.IsZero() || claims.NotBefore.IsZero() || claims.ExpiresAt.IsZero() ||
		claims.NotBefore.Before(claims.IssuedAt.Add(-24*time.Hour)) || !claims.ExpiresAt.After(claims.NotBefore) {
		return errors.New("invalid claim values")
	}
	seen := map[string]struct{}{}
	for _, feature := range claims.Features {
		if !slices.Contains(commercialFeatures, feature) {
			return fmt.Errorf("unknown feature %q", feature)
		}
		if _, duplicate := seen[feature]; duplicate {
			return fmt.Errorf("duplicate feature %q", feature)
		}
		seen[feature] = struct{}{}
	}
	slices.Sort(claims.Features)
	return nil
}

func invalid(reason string) Status {
	return Status{State: "invalid", OfflineVerification: true, Reason: reason}
}

func inside(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func timePointer(value time.Time) *time.Time {
	copy := value.UTC()
	return &copy
}
