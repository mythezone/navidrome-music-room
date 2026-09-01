package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type fakePluginStore struct {
	mu    sync.Mutex
	state domain.PluginState
}

func (f *fakePluginStore) PluginState(context.Context) (domain.PluginState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state, nil
}

func TestExchangeAndImmediateAllowlistRevocation(t *testing.T) {
	navidrome := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/getUser.view" || r.URL.Query().Get("u") != "alice" {
			t.Fatalf("unexpected Navidrome request: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"subsonic-response":{"status":"ok","user":{"username":"alice","adminRole":true,"folder":[1,2]}}}`)
	}))
	defer navidrome.Close()
	baseURL, _ := url.Parse(navidrome.URL)
	plugin := &fakePluginStore{state: domain.PluginState{
		Generation: 4, LastHeartbeat: time.Now().UTC(),
		Users: []domain.PluginUser{{Username: "alice", DisplayName: "Alice", Admin: true}},
	}}
	manager := NewSessionManager(plugin, NewNavidromeClient(baseURL), 15*time.Minute, 90*time.Second, time.Minute, "https://music.test", "https://rooms.test")
	exchange, err := manager.Exchange(t.Context(), domain.AuthProof{
		Username: "alice", Salt: "0123456789abcdef", Token: "0123456789abcdef0123456789abcdef",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !exchange.User.Admin || len(exchange.User.MusicFolderIDs) != 2 {
		t.Fatalf("unexpected exchange: %#v", exchange)
	}
	if _, err := manager.Authenticate(t.Context(), exchange.SessionToken); err != nil {
		t.Fatal(err)
	}
	plugin.mu.Lock()
	plugin.state.Users = nil
	plugin.mu.Unlock()
	if _, err := manager.Authenticate(t.Context(), exchange.SessionToken); err == nil {
		t.Fatal("expected authorization removal to invalidate the session immediately")
	}
}

func TestExistingSessionGetsLeaseGrace(t *testing.T) {
	navidrome := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"subsonic-response":{"status":"ok","user":{"username":"alice","adminRole":false,"folder":[1]}}}`)
	}))
	defer navidrome.Close()
	baseURL, _ := url.Parse(navidrome.URL)
	plugin := &fakePluginStore{state: domain.PluginState{
		Generation: 1, LastHeartbeat: time.Now().UTC(), Users: []domain.PluginUser{{Username: "alice"}},
	}}
	manager := NewSessionManager(plugin, NewNavidromeClient(baseURL), time.Hour, 90*time.Second, 60*time.Second, "https://music.test", "https://rooms.test")
	exchange, err := manager.Exchange(t.Context(), domain.AuthProof{Username: "alice", Salt: "01234567", Token: "0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	plugin.mu.Lock()
	plugin.state.LastHeartbeat = time.Now().Add(-120 * time.Second)
	plugin.mu.Unlock()
	if _, err := manager.Authenticate(t.Context(), exchange.SessionToken); err != nil {
		t.Fatalf("existing session should be inside grace period: %v", err)
	}
	if err := manager.RequireFreshLease(t.Context()); err == nil {
		t.Fatal("new activity should reject a stale 90-second lease")
	}
}
