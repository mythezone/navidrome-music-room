package domain

import (
	"testing"
	"time"
)

func TestEffectivePositionUsesServerAnchor(t *testing.T) {
	anchor := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	state := PlaybackState{
		Status: PlaybackPlaying, PositionSeconds: 10, AnchorServerTime: &anchor,
		CurrentTrack: &NavidromeTrackRef{DurationSeconds: 60},
	}
	if got := EffectivePosition(state, anchor.Add(7*time.Second)); got != 17 {
		t.Fatalf("expected position 17, got %v", got)
	}
}

func TestSelectNextAvoidsLastContributor(t *testing.T) {
	entries := []QueueEntry{
		{QueueID: "aaaaaaaaaaaaaaaa", Position: 1, Contributor: "alice"},
		{QueueID: "bbbbbbbbbbbbbbbb", Position: 2, Contributor: "bob"},
		{QueueID: "cccccccccccccccc", Position: 3, Contributor: "alice"},
	}
	next := SelectNext(entries, QueueFIFO, "alice")
	if next == nil || next.Contributor != "bob" {
		t.Fatalf("expected bob to prevent queue monopolization, got %#v", next)
	}
}

func TestTokenHashDoesNotStoreToken(t *testing.T) {
	token, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == TokenHash(token) || len(TokenHash(token)) != 64 {
		t.Fatal("expected a one-way SHA-256 token digest")
	}
}
