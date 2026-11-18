package main

import (
	"fmt"
	"testing"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/stretchr/testify/mock"
)

func TestStaleGeneration(t *testing.T) {
	response := &host.HTTPResponse{
		StatusCode: 409,
		Body:       []byte(`{"error":{"code":"stale_plugin_generation","details":{"currentGeneration":42}}}`),
	}
	generation, ok := staleGeneration(response)
	if !ok || generation != 42 {
		t.Fatalf("expected generation 42, got %d ok=%v", generation, ok)
	}
}

func TestEffectiveVersionSupportsTinyGoInjectionAndDevelopmentFallback(t *testing.T) {
	previous := version
	t.Cleanup(func() { version = previous })

	version = ""
	if got := effectiveVersion(); got != "dev" {
		t.Fatalf("expected development fallback, got %q", got)
	}
	version = " v1.2.3 "
	if got := effectiveVersion(); got != "v1.2.3" {
		t.Fatalf("expected injected version, got %q", got)
	}
}

func TestStaleGenerationRejectsUnrelatedResponses(t *testing.T) {
	tests := []*host.HTTPResponse{
		nil,
		{StatusCode: 500, Body: []byte(`{}`)},
		{StatusCode: 409, Body: []byte(`not-json`)},
		{StatusCode: 409, Body: []byte(`{"error":{"code":"configuration_mismatch","details":{"currentGeneration":42}}}`)},
		{StatusCode: 409, Body: []byte(`{"error":{"code":"stale_plugin_generation","details":{"currentGeneration":0}}}`)},
	}
	for index, response := range tests {
		if generation, ok := staleGeneration(response); ok || generation != 0 {
			t.Fatalf("case %d unexpectedly accepted: generation=%d ok=%v", index, generation, ok)
		}
	}
}

func TestUpdateNotificationIsPersistedAndDeduplicated(t *testing.T) {
	host.KVStoreMock.Mock = mock.Mock{}
	pdk.ResetMock()
	t.Cleanup(func() { host.KVStoreMock.Mock = mock.Mock{}; pdk.ResetMock() })
	body := []byte(`{"gatewayVersion":"1.1.0","updateAvailable":true,"latestVersion":"v1.2.0","releaseURL":"https://github.com/owner/repo/releases/tag/v1.2.0"}`)
	host.KVStoreMock.On("Get", "notified-update-version").Return([]byte(nil), false, nil).Once()
	host.KVStoreMock.On("Set", "notified-update-version", []byte("v1.2.0")).Return(nil).Once()
	pdk.PDKMock.On("Log", pdk.LogInfo, "Navidrome Music Room update available: v1.2.0 (https://github.com/owner/repo/releases/tag/v1.2.0). Open the plugin Website console to upgrade.").Once()
	if err := notifyUpdate(body); err != nil {
		t.Fatal(err)
	}
	host.KVStoreMock.On("Get", "notified-update-version").Return([]byte("v1.2.0"), true, nil).Once()
	if err := notifyUpdate(body); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{}`, `{"updateAvailable":false,"latestVersion":"v2.0.0"}`, `{"updateAvailable":true,"latestVersion":""}`} {
		if err := notifyUpdate([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := notifyUpdate([]byte(`not json`)); err == nil {
		t.Fatal("malformed sync response accepted")
	}
	host.KVStoreMock.AssertExpectations(t)
	pdk.PDKMock.AssertExpectations(t)
}

func TestUpdateNotificationPropagatesPersistenceErrors(t *testing.T) {
	host.KVStoreMock.Mock = mock.Mock{}
	pdk.ResetMock()
	t.Cleanup(func() { host.KVStoreMock.Mock = mock.Mock{}; pdk.ResetMock() })
	host.KVStoreMock.On("Get", "notified-update-version").Return([]byte(nil), false, fmt.Errorf("storage unavailable")).Once()
	if err := notifyUpdate([]byte(`{"updateAvailable":true,"latestVersion":"v1.2.0"}`)); err == nil {
		t.Fatal("KV error swallowed")
	}
	host.KVStoreMock.AssertExpectations(t)
}
