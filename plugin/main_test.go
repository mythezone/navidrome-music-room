package main

import (
	"testing"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
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
