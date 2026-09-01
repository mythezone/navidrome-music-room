package entitlement

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOfflineLicenseVerification(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	claims := Claims{
		LicenseID: "license-1", Subject: "example.test", IssuedAt: now.Add(-time.Hour),
		NotBefore: now.Add(-time.Minute), ExpiresAt: now.Add(24 * time.Hour), Features: []string{"chat", "statistics"},
	}
	claimsPayload, _ := json.Marshal(claims)
	document := envelope{
		Format: Format, Payload: base64.RawURLEncoding.EncodeToString(claimsPayload),
		Signature: base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, claimsPayload)),
	}
	payload, _ := json.Marshal(document)
	path := filepath.Join(dir, "license.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	provider := NewProvider(dir, base64.RawURLEncoding.EncodeToString(publicKey))
	provider.now = func() time.Time { return now }
	status := provider.Verify("license.json")
	if status.State != "valid" || status.LicenseID != claims.LicenseID || len(status.GrantedFeatures) != 2 {
		t.Fatalf("unexpected valid status: %#v", status)
	}

	document.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, []byte("tampered")))
	payload, _ = json.Marshal(document)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if status := provider.Verify("license.json"); status.State != "invalid" || status.Reason != "signature_invalid" {
		t.Fatalf("tampered license was accepted: %#v", status)
	}
}

func TestLicensePathIsConfinedToDataDirectory(t *testing.T) {
	provider := NewProvider(t.TempDir(), "")
	status := provider.Verify("../../etc/passwd")
	if status.State != "invalid" || status.Reason != "invalid_path" {
		t.Fatalf("path traversal was not rejected: %#v", status)
	}
}

func TestMissingLicenseAndKeyStates(t *testing.T) {
	dir := t.TempDir()
	provider := NewProvider(dir, "")
	if status := provider.Verify(""); status.State != "not_installed" {
		t.Fatalf("missing license state: %#v", status)
	}
	if err := os.MkdirAll(filepath.Join(dir, "secrets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, defaultFile), []byte(`{"format":"musicmate-license/v1","payload":"e30","signature":"AA"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if status := provider.Verify(""); status.State != "verification_unavailable" {
		t.Fatalf("missing key state: %#v", status)
	}
}
