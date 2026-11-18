package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mythezone/navidrome-music-room/gateway/internal/auth"
	"github.com/mythezone/navidrome-music-room/gateway/internal/config"
	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
	"github.com/mythezone/navidrome-music-room/gateway/internal/store"
	updatemanager "github.com/mythezone/navidrome-music-room/gateway/internal/update"
)

type apiFixture struct {
	server            *httptest.Server
	store             *store.Store
	pairing           string
	navidromeInternal string
	api               *Server
}

func newAPIFixture(t *testing.T) apiFixture {
	t.Helper()
	navidrome := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("u")
		admin := username == "admin"
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/getSong.view") {
			fmt.Fprint(w, `{"subsonic-response":{"status":"ok","song":{"id":"track-e2e","title":"E2E Track","artist":"MusicMate","album":"Integration","albumId":"album-e2e","coverArt":"cover-e2e","duration":180}}}`)
			return
		}
		fmt.Fprintf(w, `{"subsonic-response":{"status":"ok","user":{"username":%q,"adminRole":%v,"folder":[1,2]}}}`, username, admin)
	}))
	t.Cleanup(navidrome.Close)
	internal, _ := url.Parse(navidrome.URL)
	navidromePublic, _ := url.Parse("https://music.example.test")
	gatewayPublic, _ := url.Parse("https://rooms.example.test")
	dir := t.TempDir()
	storage, err := store.Open(t.Context(), filepath.Join(dir, "rooms.sqlite3"), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	pairing := "0123456789abcdef0123456789abcdef"
	cfg := config.Config{
		ListenAddress: ":0", DataDir: dir, DatabasePath: filepath.Join(dir, "rooms.sqlite3"),
		NavidromeInternal: internal, NavidromePublic: navidromePublic, GatewayPublic: gatewayPublic,
		PluginPairingToken: pairing, Version: "test", ReleaseRepository: "mythezone/navidrome-music-room",
		CosignBinary: "cosign", SigstoreTrustedRoot: filepath.Join(dir, "trusted-root.json"),
		UpdateIdentity: "https://github.com/mythezone/navidrome-music-room/.github/workflows/release.yml@refs/tags/.*",
		PluginLease:    90 * time.Second, ExistingGrace: time.Minute, SessionTTL: 15 * time.Minute,
		WebSocketTicketTTL: time.Minute, EmptyRoomPauseDelay: 15 * time.Second,
	}
	sessions := auth.NewSessionManager(storage, auth.NewNavidromeClient(internal), cfg.SessionTTL, cfg.PluginLease, cfg.ExistingGrace, navidromePublic.String(), gatewayPublic.String())
	server, err := NewServer(cfg, storage, sessions, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	fixture := apiFixture{server: httpServer, store: storage, pairing: pairing, navidromeInternal: internal.String(), api: server}
	fixture.syncPlugin(t)
	return fixture
}

func (f apiFixture) syncPlugin(t *testing.T) {
	t.Helper()
	body := map[string]any{
		"pluginVersion": "test", "generation": 1,
		"navidromeInternalURL": f.navidromeInternal,
		"navidromePublicURL":   "https://music.example.test", "gatewayPublicURL": "https://rooms.example.test",
		"sentAt": time.Now().UTC(),
		"users": []map[string]any{
			{"username": "admin", "displayName": "Admin", "admin": true},
			{"username": "member", "displayName": "Member", "admin": false},
			{"username": "member2", "displayName": "Member Two", "admin": false},
		},
	}
	response := f.request(t, http.MethodPost, "/internal/v1/plugin-sync", body, f.pairing)
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("plugin sync failed: %d %s", response.StatusCode, readBody(response))
	}
}

func (f apiFixture) exchange(t *testing.T, username string) string {
	t.Helper()
	response := f.request(t, http.MethodPost, "/api/v1/auth/exchange", map[string]any{
		"username": username, "salt": "0123456789abcdef", "token": "0123456789abcdef0123456789abcdef",
	}, "")
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("exchange failed: %d %s", response.StatusCode, readBody(response))
	}
	var payload domain.AuthExchange
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return payload.SessionToken
}

func (f apiFixture) request(t *testing.T, method, path string, body any, bearer string) *http.Response {
	return f.requestWithHeaders(t, method, path, body, bearer, nil)
}

func (f apiFixture) requestWithHeaders(t *testing.T, method, path string, body any, bearer string, headers map[string]string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, f.server.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func TestPublicDiscoveryContainsOnlyConnectionMetadata(t *testing.T) {
	fixture := newAPIFixture(t)
	response := fixture.request(t, http.MethodGet, "/api/v1/discovery", nil, "")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("discovery failed: %d %s", response.StatusCode, readBody(response))
	}
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["apiVersion"] != "v1" || payload["navidromeBaseURL"] != "https://music.example.test" {
		t.Fatalf("unexpected discovery response: %#v", payload)
	}
	if _, leaked := payload["pluginPairingToken"]; leaked {
		t.Fatal("discovery must not expose the plugin pairing token")
	}
}

func TestAdminUIIsEmbeddedAndUsesStrictBrowserHeaders(t *testing.T) {
	fixture := newAPIFixture(t)
	response := fixture.request(t, http.MethodGet, "/admin/", nil, "")
	body := readBody(response)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "听歌房 · Navidrome") {
		t.Fatalf("embedded admin UI failed: status=%d body=%s", response.StatusCode, body)
	}
	if !strings.Contains(response.Header.Get("Content-Security-Policy"), "script-src 'self'") || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("admin UI security headers are incomplete: %v", response.Header)
	}

	marker := `src="./assets/`
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("admin UI asset was not referenced: %s", body)
	}
	start += len(`src=".`)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		t.Fatalf("admin UI asset reference is malformed: %s", body)
	}
	assetPath := "/admin" + body[start:start+end]
	response = fixture.request(t, http.MethodGet, assetPath, nil, "")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(response.Header.Get("Content-Type"), "javascript") {
		t.Fatalf("embedded admin UI asset failed: status=%d headers=%v", response.StatusCode, response.Header)
	}
}

func TestRoomUIIsEmbeddedAndStreamsAssetsWithStrictHeaders(t *testing.T) {
	fixture := newAPIFixture(t)
	roomID := "0123456789abcdef0123456789abcdef"
	response := fixture.request(t, http.MethodGet, "/join/"+roomID+"/", nil, "")
	body := readBody(response)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(body, "一起听歌 · Navidrome") {
		t.Fatalf("embedded room UI failed: status=%d body=%s", response.StatusCode, body)
	}
	csp := response.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self'") || !strings.Contains(csp, "media-src 'self' blob:") || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("room UI security headers are incomplete: %v", response.Header)
	}

	marker := `src="./assets/`
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("room UI asset was not referenced: %s", body)
	}
	start += len(`src=".`)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		t.Fatalf("room UI asset reference is malformed: %s", body)
	}
	assetPath := "/join/" + roomID + body[start:start+end]
	response = fixture.request(t, http.MethodGet, assetPath, nil, "")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(response.Header.Get("Content-Type"), "javascript") {
		t.Fatalf("embedded room UI asset failed: status=%d headers=%v", response.StatusCode, response.Header)
	}
	if !strings.Contains(response.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("room UI asset is not immutable: %v", response.Header)
	}
}

func TestRoomUIRedirectUsesRelativeLocationForProxyPrefix(t *testing.T) {
	fixture := newAPIFixture(t)
	roomID := "0123456789abcdef0123456789abcdef"
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get(fixture.server.URL + "/join/" + roomID)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusPermanentRedirect || response.Header.Get("Location") != roomID+"/" {
		t.Fatalf("unexpected room redirect: status=%d location=%q", response.StatusCode, response.Header.Get("Location"))
	}
}

func TestReadinessReportsGatewayAndPluginVersions(t *testing.T) {
	fixture := newAPIFixture(t)
	response := fixture.request(t, http.MethodGet, "/readyz", nil, "")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("readiness failed: %d %s", response.StatusCode, readBody(response))
	}
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["version"] != "test" || payload["pluginVersion"] != "test" || payload["plugin"] != "paired" {
		t.Fatalf("unexpected readiness response: %#v", payload)
	}
}

func TestPluginSyncRejectsNavidromeInternalURLMismatch(t *testing.T) {
	fixture := newAPIFixture(t)
	response := fixture.request(t, http.MethodPost, "/internal/v1/plugin-sync", map[string]any{
		"pluginVersion":        "test",
		"generation":           2,
		"navidromeInternalURL": "http://unexpected-navidrome:4533",
		"navidromePublicURL":   "https://music.example.test",
		"gatewayPublicURL":     "https://rooms.example.test",
		"users":                []map[string]any{{"username": "admin", "displayName": "Admin", "admin": true}},
		"sentAt":               time.Now().UTC(),
	}, fixture.pairing)
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict || !strings.Contains(readBody(response), "configuration_mismatch") {
		t.Fatalf("mismatched internal URL was accepted: %d", response.StatusCode)
	}
}

func TestDiagnosticExportIsAdminOnlyAndRedacted(t *testing.T) {
	fixture := newAPIFixture(t)
	memberToken := fixture.exchange(t, "member")
	response := fixture.request(t, http.MethodGet, "/api/v1/admin/diagnostics", nil, memberToken)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("member exported diagnostics: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	adminToken := fixture.exchange(t, "admin")
	response = fixture.request(t, http.MethodGet, "/api/v1/admin/diagnostics", nil, adminToken)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("diagnostic export failed: %d %s", response.StatusCode, readBody(response))
	}
	if response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("Content-Disposition") == "" {
		t.Fatalf("diagnostic response is not a private attachment: %v", response.Header)
	}
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{
		fixture.pairing, "https://music.example.test", "https://rooms.example.test",
		`"username"`, `"displayName"`,
	} {
		if bytes.Contains(payload, []byte(secret)) {
			t.Fatalf("diagnostic export leaked %q: %s", secret, payload)
		}
	}
	var bundle map[string]any
	if err := json.Unmarshal(payload, &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle["redacted"] != true || bundle["format"] != "navidrome-music-room-diagnostics/v1" {
		t.Fatalf("unexpected diagnostic envelope: %#v", bundle)
	}
}

func TestPluginAuthorizationRemovalRevokesExistingSessions(t *testing.T) {
	fixture := newAPIFixture(t)
	memberToken := fixture.exchange(t, "member")
	body := map[string]any{
		"pluginVersion": "test", "generation": 2,
		"navidromeInternalURL": fixture.navidromeInternal,
		"navidromePublicURL":   "https://music.example.test", "gatewayPublicURL": "https://rooms.example.test",
		"sentAt": time.Now().UTC(),
		"users":  []map[string]any{{"username": "admin", "displayName": "Admin", "admin": true}},
	}
	response := fixture.request(t, http.MethodPost, "/internal/v1/plugin-sync", body, fixture.pairing)
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("plugin resync failed: %d", response.StatusCode)
	}
	response = fixture.request(t, http.MethodGet, "/api/v1/rooms", nil, memberToken)
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("removed user session remained valid: %d %s", response.StatusCode, readBody(response))
	}
}

func TestEmptyCollectionsSerializeAsArrays(t *testing.T) {
	fixture := newAPIFixture(t)
	memberToken := fixture.exchange(t, "member")
	response := fixture.request(t, http.MethodGet, "/api/v1/rooms", nil, memberToken)
	roomsBody := readBody(response)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(roomsBody, `"rooms":[]`) {
		t.Fatalf("empty room list must be a JSON array: status=%d body=%s", response.StatusCode, roomsBody)
	}

	adminToken := fixture.exchange(t, "admin")
	response = fixture.request(t, http.MethodPost, "/api/v1/rooms", map[string]any{
		"name": "Empty Collections", "musicFolderIDs": []int{1},
	}, adminToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create room failed: %d %s", response.StatusCode, readBody(response))
	}
	var room domain.Room
	if err := json.NewDecoder(response.Body).Decode(&room); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	response = fixture.request(t, http.MethodGet, "/api/v1/rooms/"+room.RoomID+"/snapshot", nil, adminToken)
	snapshotBody := readBody(response)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(snapshotBody, `"queue":[]`) || !strings.Contains(snapshotBody, `"history":[]`) {
		t.Fatalf("snapshot collections must be JSON arrays: status=%d body=%s", response.StatusCode, snapshotBody)
	}

	response = fixture.request(t, http.MethodGet, "/api/v1/rooms/"+room.RoomID+"/history", nil, adminToken)
	historyBody := readBody(response)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(historyBody, `"items":[]`) {
		t.Fatalf("empty history must be a JSON array: status=%d body=%s", response.StatusCode, historyBody)
	}

	response = fixture.request(t, http.MethodGet, "/api/v1/rooms/"+room.RoomID+"/invites", nil, adminToken)
	invitesBody := readBody(response)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(invitesBody, `"invites":[]`) {
		t.Fatalf("empty invite list must be a JSON array: status=%d body=%s", response.StatusCode, invitesBody)
	}
}

func TestCreateRoomIdempotencyReplaysAndRejectsDifferentPayload(t *testing.T) {
	fixture := newAPIFixture(t)
	token := fixture.exchange(t, "admin")
	headers := map[string]string{"Idempotency-Key": "create-room-test-0001"}

	first := fixture.requestWithHeaders(t, http.MethodPost, "/api/v1/rooms", map[string]any{"name": "Idempotent Room"}, token, headers)
	defer first.Body.Close()
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first create failed: %d %s", first.StatusCode, readBody(first))
	}
	var firstRoom domain.Room
	if err := json.NewDecoder(first.Body).Decode(&firstRoom); err != nil {
		t.Fatal(err)
	}

	refreshedToken := fixture.exchange(t, "admin")
	second := fixture.requestWithHeaders(t, http.MethodPost, "/api/v1/rooms", map[string]any{"name": "Idempotent Room"}, refreshedToken, headers)
	defer second.Body.Close()
	if second.StatusCode != http.StatusCreated || second.Header.Get("Idempotency-Replayed") != "true" {
		t.Fatalf("expected replayed create, got %d headers=%v body=%s", second.StatusCode, second.Header, readBody(second))
	}
	var secondRoom domain.Room
	if err := json.NewDecoder(second.Body).Decode(&secondRoom); err != nil {
		t.Fatal(err)
	}
	if firstRoom.RoomID != secondRoom.RoomID {
		t.Fatalf("idempotent replay created another room: %q != %q", firstRoom.RoomID, secondRoom.RoomID)
	}

	conflict := fixture.requestWithHeaders(t, http.MethodPost, "/api/v1/rooms", map[string]any{"name": "Different Room"}, token, headers)
	defer conflict.Body.Close()
	if conflict.StatusCode != http.StatusConflict {
		t.Fatalf("expected idempotency conflict, got %d %s", conflict.StatusCode, readBody(conflict))
	}
}

func TestAdminInvitationAndMemberPermissions(t *testing.T) {
	fixture := newAPIFixture(t)
	adminToken := fixture.exchange(t, "admin")
	memberToken := fixture.exchange(t, "member")

	response := fixture.request(t, http.MethodPost, "/api/v1/rooms", map[string]any{
		"name": "Friday Listening", "queueLimit": 20, "playbackMode": "fifo", "musicFolderIDs": []int{1},
	}, adminToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create room failed: %d %s", response.StatusCode, readBody(response))
	}
	var room domain.Room
	if err := json.NewDecoder(response.Body).Decode(&room); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms", map[string]any{"name": "Forbidden"}, memberToken)
	if response.StatusCode != 403 {
		t.Fatalf("member should not create room: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/join", map[string]any{}, memberToken)
	if response.StatusCode != 403 {
		t.Fatalf("member should need invitation before join: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/invites", map[string]any{}, adminToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create invite failed: %d %s", response.StatusCode, readBody(response))
	}
	var invite domain.Invite
	if err := json.NewDecoder(response.Body).Decode(&invite); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if invite.Invite == "" || invite.ShareURL == "" || invite.DeepLink == "" {
		t.Fatalf("invite links are incomplete: %#v", invite)
	}

	response = fixture.request(t, http.MethodPost, "/api/v1/invites/redeem", map[string]any{
		"roomID": room.RoomID, "invite": invite.Invite,
	}, memberToken)
	if response.StatusCode != 200 {
		t.Fatalf("redeem failed: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/playback", map[string]any{
		"command": "play", "expectedRevision": 0,
	}, memberToken)
	if response.StatusCode != 403 {
		t.Fatalf("member should not control global playback: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	response = fixture.request(t, http.MethodGet, "/api/v1/rooms/"+room.RoomID+"/chat", nil, memberToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("chat should return a page: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
}

func TestThreeClientsReceiveAuthoritativePlaybackAndReconnect(t *testing.T) {
	fixture := newAPIFixture(t)
	adminToken := fixture.exchange(t, "admin")
	memberToken := fixture.exchange(t, "member")
	memberTwoToken := fixture.exchange(t, "member2")

	response := fixture.request(t, http.MethodPost, "/api/v1/rooms", map[string]any{
		"name": "Three Clients", "musicFolderIDs": []int{1},
	}, adminToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create room failed: %d %s", response.StatusCode, readBody(response))
	}
	var room domain.Room
	if err := json.NewDecoder(response.Body).Decode(&room); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/invites", map[string]any{}, adminToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create invite failed: %d %s", response.StatusCode, readBody(response))
	}
	var invite domain.Invite
	if err := json.NewDecoder(response.Body).Decode(&invite); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	for _, token := range []string{memberToken, memberTwoToken} {
		response = fixture.request(t, http.MethodPost, "/api/v1/invites/redeem", map[string]any{
			"roomID": room.RoomID, "invite": invite.Invite,
		}, token)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("redeem failed: %d %s", response.StatusCode, readBody(response))
		}
		response.Body.Close()
	}

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/queue/tracks", map[string]any{
		"track": map[string]any{"id": "track-e2e", "musicFolderID": 1},
	}, memberToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("queue track failed: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()

	tokens := []string{adminToken, memberToken, memberTwoToken}
	connections := make([]*websocket.Conn, 0, len(tokens))
	for _, token := range tokens {
		connection := fixture.connectWebSocket(t, room.RoomID, token)
		connections = append(connections, connection)
		t.Cleanup(func() { _ = connection.Close() })
		snapshot := readWebSocketEvent(t, connection, "snapshot")
		var payload domain.Snapshot
		if err := json.Unmarshal(snapshot.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Playback.Revision != 1 || payload.Playback.Status != domain.PlaybackPaused || payload.Playback.CurrentTrack == nil {
			t.Fatalf("new client received stale initial revision: %#v", payload.Playback)
		}
	}
	for _, connection := range connections {
		presence := readWebSocketEvent(t, connection, "presence")
		var payload struct {
			OnlineCount int `json:"onlineCount"`
		}
		if err := json.Unmarshal(presence.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		for payload.OnlineCount != 3 {
			presence = readWebSocketEvent(t, connection, "presence")
			if err := json.Unmarshal(presence.Payload, &payload); err != nil {
				t.Fatal(err)
			}
		}
	}

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/playback", map[string]any{
		"command": "play", "expectedRevision": 1,
	}, adminToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("play failed: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	for _, connection := range connections {
		event := readWebSocketEvent(t, connection, "playback")
		if event.Revision != 2 {
			t.Fatalf("client did not receive playback revision 2: %#v", event)
		}
	}

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/playback", map[string]any{
		"command": "seek", "expectedRevision": 2, "positionSeconds": 42,
	}, adminToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("seek failed: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	for _, connection := range connections {
		event := readWebSocketEvent(t, connection, "playback")
		if event.Revision != 3 {
			t.Fatalf("client did not receive seek revision 3: %#v", event)
		}
	}

	_ = connections[2].Close()
	reconnected := fixture.connectWebSocket(t, room.RoomID, memberTwoToken)
	defer reconnected.Close()
	snapshot := readWebSocketEvent(t, reconnected, "snapshot")
	var restored domain.Snapshot
	if err := json.Unmarshal(snapshot.Payload, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Playback.Revision != 3 || restored.Playback.PositionSeconds < 42 {
		t.Fatalf("reconnect did not restore authoritative playback: %#v", restored.Playback)
	}

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/playback", map[string]any{
		"command": "pause", "expectedRevision": 2,
	}, adminToken)
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("stale revision overwrote newer playback: %d %s", response.StatusCode, readBody(response))
	}
}

func TestFirstQueueTrackBroadcastsPausedCurrentPlayback(t *testing.T) {
	fixture := newAPIFixture(t)
	adminToken := fixture.exchange(t, "admin")
	response := fixture.request(t, http.MethodPost, "/api/v1/rooms", map[string]any{
		"name": "Prime First Track", "musicFolderIDs": []int{1},
	}, adminToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create room failed: %d %s", response.StatusCode, readBody(response))
	}
	var room domain.Room
	if err := json.NewDecoder(response.Body).Decode(&room); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	connection := fixture.connectWebSocket(t, room.RoomID, adminToken)
	defer connection.Close()
	_ = readWebSocketEvent(t, connection, "snapshot")

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/queue/tracks", map[string]any{
		"track": map[string]any{"id": "track-e2e", "musicFolderID": 1},
	}, adminToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("queue track failed: %d %s", response.StatusCode, readBody(response))
	}
	var mutation struct {
		Queue    []domain.QueueEntry   `json:"queue"`
		Playback *domain.PlaybackState `json:"playback"`
	}
	if err := json.NewDecoder(response.Body).Decode(&mutation); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if mutation.Playback == nil || mutation.Playback.Revision != 1 || mutation.Playback.Status != domain.PlaybackPaused || mutation.Playback.CurrentTrack == nil || len(mutation.Queue) != 0 {
		t.Fatalf("first queue mutation did not expose primed playback: %#v", mutation)
	}

	event := readWebSocketEvent(t, connection, "playback")
	var playback domain.PlaybackState
	if err := json.Unmarshal(event.Payload, &playback); err != nil {
		t.Fatal(err)
	}
	if event.Revision != 1 || playback.Status != domain.PlaybackPaused || playback.CurrentTrack == nil || playback.CurrentTrack.ID != "track-e2e" {
		t.Fatalf("client did not receive the primed current track: event=%#v playback=%#v", event, playback)
	}
	queueEvent := readWebSocketEvent(t, connection, "queue")
	var queue []domain.QueueEntry
	if err := json.Unmarshal(queueEvent.Payload, &queue); err != nil || len(queue) != 0 {
		t.Fatalf("promoted track remained in the pending queue: %#v err=%v", queue, err)
	}
	if history, err := fixture.store.History(t.Context(), room.RoomID, 10, 0); err != nil || len(history) != 0 {
		t.Fatalf("priming created playback history too early: %#v err=%v", history, err)
	}
}

type webSocketEvent struct {
	Type     string          `json:"type"`
	Revision int64           `json:"revision"`
	Payload  json.RawMessage `json:"payload"`
}

func (f apiFixture) connectWebSocket(t *testing.T, roomID, bearer string) *websocket.Conn {
	t.Helper()
	response := f.request(t, http.MethodPost, "/api/v1/rooms/"+roomID+"/ws-ticket", map[string]any{}, bearer)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("ticket failed: %d %s", response.StatusCode, readBody(response))
	}
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	if err := json.NewDecoder(response.Body).Decode(&ticket); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	endpoint := "ws" + strings.TrimPrefix(f.server.URL, "http") + "/api/v1/rooms/" + roomID + "/ws?ticket=" + url.QueryEscape(ticket.Ticket)
	connection, dialResponse, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		if dialResponse != nil {
			defer dialResponse.Body.Close()
			t.Fatalf("websocket dial failed: %v: %s", err, readBody(dialResponse))
		}
		t.Fatal(err)
	}
	return connection
}

func readWebSocketEvent(t *testing.T, connection *websocket.Conn, wanted string) webSocketEvent {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if err := connection.SetReadDeadline(deadline); err != nil {
			t.Fatal(err)
		}
		var event webSocketEvent
		if err := connection.ReadJSON(&event); err != nil {
			t.Fatalf("read %s event: %v", wanted, err)
		}
		if event.Type == wanted {
			return event
		}
	}
}

func readBody(response *http.Response) string {
	body, _ := io.ReadAll(response.Body)
	return string(body)
}

func (f apiFixture) expectJSON(t *testing.T, method, path, token string, body any, status int, target any) string {
	t.Helper()
	response := f.request(t, method, path, body, token)
	defer response.Body.Close()
	raw := readBody(response)
	if response.StatusCode != status {
		t.Fatalf("%s %s: got %d want %d: %s", method, path, response.StatusCode, status, raw)
	}
	if target != nil {
		if err := json.Unmarshal([]byte(raw), target); err != nil {
			t.Fatal(err)
		}
	}
	return raw
}

func communityFixture(t *testing.T) (apiFixture, domain.Room, string, string) {
	t.Helper()
	f := newAPIFixture(t)
	admin, member := f.exchange(t, "admin"), f.exchange(t, "member")
	var room domain.Room
	f.expectJSON(t, "POST", "/api/v1/rooms", admin, map[string]any{"name": "Community", "musicFolderIDs": []int{1}}, 201, &room)
	if _, err := f.store.UpsertMember(t.Context(), room.RoomID, domain.User{Username: "member", DisplayName: "Member"}, "member", "admin", "member.joined"); err != nil {
		t.Fatal(err)
	}
	return f, room, admin, member
}

func TestAdminRedirectAndCollaborativeEndpoints(t *testing.T) {
	f, room, admin, member := communityFixture(t)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get(f.server.URL + "/admin")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 308 || response.Header.Get("Location") != "admin/" || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("admin redirect: %+v", response)
	}
	if !room.AllowMemberSkip || !room.Capabilities.Chat || !room.Capabilities.Achievements {
		t.Fatalf("new defaults: %+v", room)
	}
	base := "/api/v1/rooms/" + room.RoomID
	for _, endpoint := range []string{"chat", "statistics", "rankings", "achievements", "wishes", "stickers"} {
		f.expectJSON(t, "GET", base+"/"+endpoint, member, nil, 200, nil)
	}
	var page domain.ChatPage
	f.expectJSON(t, "GET", base+"/chat", member, nil, 200, &page)
	if page.Messages == nil {
		t.Fatal("messages serialized as null")
	}
	var stats domain.RoomStatistics
	f.expectJSON(t, "GET", base+"/statistics", member, nil, 200, &stats)
	if stats.TopTracks == nil || stats.TopArtists == nil || stats.TopAlbums == nil || len(stats.HourlyDistribution) != 24 {
		t.Fatalf("empty stats: %+v", stats)
	}
	var ranks domain.RoomRankings
	f.expectJSON(t, "GET", base+"/rankings", member, nil, 200, &ranks)
	if ranks.Requested == nil || ranks.Listened == nil || ranks.Chat == nil {
		t.Fatalf("null ranks: %+v", ranks)
	}
	outsider := f.exchange(t, "member2")
	f.expectJSON(t, "GET", base+"/chat", outsider, nil, 403, nil)
	f.expectJSON(t, "GET", base+"/members/member2/profile", member, nil, 404, nil)
	f.expectJSON(t, "GET", base+"/statistics?period=bad", member, nil, 400, nil)
	f.expectJSON(t, "GET", base+"/rankings?period=bad", member, nil, 400, nil)
	f.expectJSON(t, "GET", base+"/wishes?status=bad", member, nil, 400, nil)
	var packs struct {
		Packs []domain.StickerPack `json:"packs"`
	}
	f.expectJSON(t, "GET", base+"/stickers", member, nil, 200, &packs)
	if len(packs.Packs) != 2 {
		t.Fatal("expected two packs")
	}
	for _, pack := range packs.Packs {
		if len(pack.Stickers) != 8 {
			t.Fatal("expected eight stickers")
		}
		for _, sticker := range pack.Stickers {
			if len(sticker.Labels) != 7 {
				t.Fatalf("missing labels: %+v", sticker)
			}
		}
	}
	var snapshot domain.Snapshot
	f.expectJSON(t, "GET", base+"/snapshot", member, nil, 200, &snapshot)
	if snapshot.Permissions.CanManage || snapshot.Permissions.CanControl || !snapshot.Permissions.CanSkip || snapshot.UpdateNotice != nil {
		t.Fatalf("member permissions: %+v", snapshot)
	}
	f.expectJSON(t, "GET", base+"/snapshot", admin, nil, 200, &snapshot)
	if !snapshot.Permissions.CanManage || !snapshot.Permissions.CanFulfillWish || !snapshot.Permissions.CanReorderQueue {
		t.Fatalf("manager permissions: %+v", snapshot.Permissions)
	}
}

func TestSkipPermissionsQueueEnhancementsAndRequeue(t *testing.T) {
	f, room, admin, member := communityFixture(t)
	base := "/api/v1/rooms/" + room.RoomID
	track := map[string]any{"track": map[string]any{"id": "track-e2e", "musicFolderID": 1}}
	var snapshot domain.Snapshot
	getState := func() domain.PlaybackState {
		t.Helper()
		f.expectJSON(t, "GET", base+"/snapshot", admin, nil, 200, &snapshot)
		return snapshot.Playback
	}
	command := func(token, command string, want int) string {
		t.Helper()
		state := getState()
		return f.expectJSON(t, "POST", base+"/playback", token, map[string]any{"command": command, "expectedRevision": state.Revision}, want, nil)
	}
	for _, cmd := range []string{"play", "pause", "seek", "previous", "stop"} {
		if !strings.Contains(command(member, cmd, 403), "room_manager_required") {
			t.Fatal("wrong manager error")
		}
	}
	f.expectJSON(t, "POST", base+"/queue/tracks", admin, track, 201, nil)
	f.expectJSON(t, "POST", base+"/queue/tracks", member, track, 201, nil)
	command(member, "next", 200)
	f.expectJSON(t, "PATCH", base, admin, map[string]any{"allowMemberSkip": false}, 200, nil)
	command(member, "next", 200) // The member contributed the current track.
	f.expectJSON(t, "POST", base+"/queue/tracks", admin, track, 201, nil)
	if !strings.Contains(command(member, "next", 403), "skip_permission_required") {
		t.Fatal("wrong skip error")
	}
	var memberSnapshot domain.Snapshot
	f.expectJSON(t, "GET", base+"/snapshot", member, nil, 200, &memberSnapshot)
	if memberSnapshot.Permissions.CanSkip {
		t.Fatal("snapshot grants disabled skip")
	}
	f.expectJSON(t, "POST", base+"/queue/tracks", member, map[string]any{"track": track["track"], "playNext": true}, 403, nil)
	for range 3 {
		f.expectJSON(t, "POST", base+"/queue/tracks", admin, track, 201, nil)
	}
	getState()
	fixed := snapshot.Queue[0].QueueID
	f.expectJSON(t, "POST", base+"/queue/shuffle", member, map[string]any{}, 403, nil)
	var queue struct {
		Queue []domain.QueueEntry `json:"queue"`
	}
	f.expectJSON(t, "POST", base+"/queue/shuffle", admin, map[string]any{}, 200, &queue)
	if len(queue.Queue) != 3 || queue.Queue[0].QueueID != fixed {
		t.Fatal("shuffle moved first queue entry")
	}
	top := queue.Queue[2].QueueID
	f.expectJSON(t, "POST", base+"/queue/"+top+"/top", member, map[string]any{}, 403, nil)
	f.expectJSON(t, "POST", base+"/queue/"+top+"/top", admin, map[string]any{}, 200, &queue)
	if queue.Queue[0].QueueID != top {
		t.Fatal("top did not move entry")
	}
	var mutation struct {
		Entry domain.QueueEntry   `json:"entry"`
		Queue []domain.QueueEntry `json:"queue"`
	}
	f.expectJSON(t, "POST", base+"/queue/tracks", admin, map[string]any{"track": track["track"], "playNext": true}, 201, &mutation)
	if mutation.Queue[0].QueueID != mutation.Entry.QueueID {
		t.Fatal("playNext not at front")
	}
	command(admin, "play", 200)
	command(admin, "next", 200)
	getState()
	historyID := snapshot.History[0].HistoryID
	f.expectJSON(t, "POST", base+"/history/"+historyID+"/requeue", member, map[string]any{}, 201, &mutation)
	if mutation.Entry.Contributor != "member" {
		t.Fatal("requeue retains old contributor")
	}
	f.expectJSON(t, "PATCH", base, admin, map[string]any{"musicFolderIDs": []int{2}}, 200, nil)
	f.expectJSON(t, "POST", base+"/history/"+historyID+"/requeue", member, map[string]any{}, 403, nil)
	f.expectJSON(t, "POST", base+"/close", admin, map[string]any{}, 200, nil)
	f.expectJSON(t, "POST", base+"/history/"+historyID+"/requeue", member, map[string]any{}, 409, nil)
}

func TestChatWishBroadcastsAndRateLimit(t *testing.T) {
	f, room, admin, member := communityFixture(t)
	base := "/api/v1/rooms/" + room.RoomID
	connection := f.connectWebSocket(t, room.RoomID, member)
	defer connection.Close()
	readWebSocketEvent(t, connection, "snapshot")
	f.expectJSON(t, "POST", base+"/chat", member, map[string]any{"body": " "}, 400, nil)
	f.expectJSON(t, "POST", base+"/chat", member, map[string]any{"stickerID": "unknown"}, 400, nil)
	f.expectJSON(t, "POST", base+"/chat", member, map[string]any{"messageType": "system", "body": "hi"}, 400, nil)
	f.expectJSON(t, "POST", base+"/chat", member, map[string]any{"body": "hi", "references": []any{map[string]any{"kind": "bad", "id": "a", "title": "A"}}}, 400, nil)
	var msg domain.ChatMessage
	f.expectJSON(t, "POST", base+"/chat", member, map[string]any{"body": "hello", "mentions": []string{"ADMIN"}, "stickerID": "vibes:music"}, 201, &msg)
	event := readWebSocketEvent(t, connection, "chat_message")
	if !strings.Contains(string(event.Payload), msg.MessageID) || msg.Mentions[0] != "admin" {
		t.Fatalf("chat broadcast: %s", event.Payload)
	}
	var reply domain.ChatMessage
	f.expectJSON(t, "POST", base+"/chat", admin, map[string]any{"messageType": "review", "references": []domain.ChatReference{{Kind: "track", ID: "track-e2e", Title: "Track"}}, "replyToID": msg.MessageID}, 201, &reply)
	if reply.ReplyPreview == nil || reply.ReplyPreview.Body != "hello" {
		t.Fatal("missing reply preview")
	}
	f.expectJSON(t, "POST", base+"/chat/"+msg.MessageID+"/recall", admin, map[string]any{}, 403, nil)
	f.expectJSON(t, "POST", base+"/chat/"+msg.MessageID+"/recall", member, map[string]any{}, 200, &msg)
	event = readWebSocketEvent(t, connection, "chat_updated")
	if msg.Body != "" || msg.StickerID != "" || len(msg.Mentions) != 0 || !strings.Contains(string(event.Payload), "recalledAt") {
		t.Fatal("recall content not cleared")
	}
	f.expectJSON(t, "DELETE", base+"/chat/"+reply.MessageID, member, nil, 403, nil)
	f.expectJSON(t, "DELETE", base+"/chat/"+msg.MessageID, admin, nil, 204, nil)
	readWebSocketEvent(t, connection, "chat_updated")
	var wish domain.Wish
	f.expectJSON(t, "POST", base+"/wishes", member, map[string]any{"kind": "album", "title": "New album"}, 201, &wish)
	readWebSocketEvent(t, connection, "wish_updated")
	if wish.Resources == nil {
		t.Fatal("wish resources null")
	}
	f.expectJSON(t, "POST", base+"/wishes/"+wish.WishID+"/fulfill", member, map[string]any{}, 403, nil)
	f.expectJSON(t, "POST", base+"/wishes/"+wish.WishID+"/fulfill", admin, map[string]any{"resources": []domain.WishResource{{Title: "Unsafe", URL: "javascript:alert(1)"}}}, 400, nil)
	f.expectJSON(t, "POST", base+"/wishes/"+wish.WishID+"/fulfill", admin, map[string]any{"responseNote": "Ready"}, 200, &wish)
	readWebSocketEvent(t, connection, "wish_updated")
	f.expectJSON(t, "DELETE", base+"/wishes/"+wish.WishID, member, nil, 403, nil)
	f.expectJSON(t, "DELETE", base+"/wishes/"+wish.WishID, admin, nil, 204, nil)
	readWebSocketEvent(t, connection, "wish_deleted")
	for range 19 {
		f.expectJSON(t, "POST", base+"/chat", member, map[string]any{"body": "participating"}, 201, nil)
	}
	if raw := f.expectJSON(t, "POST", base+"/chat", member, map[string]any{"body": "over limit"}, 429, nil); !strings.Contains(raw, "chat_rate_limited") {
		t.Fatal(raw)
	}
	var profile domain.MemberProfile
	f.expectJSON(t, "GET", base+"/achievements", member, nil, 200, &profile)
	if profile.Stats.ChatCount != 20 {
		t.Fatalf("chat stats: %+v", profile)
	}
	f.expectJSON(t, "PUT", base+"/achievements/selected", member, map[string]any{"badgeID": "chat_50"}, 400, nil)
	f.expectJSON(t, "PUT", base+"/achievements/selected", member, map[string]any{"badgeID": "chat_10"}, 200, &profile)
	if profile.SelectedBadgeID != "chat_10" {
		t.Fatal("badge not persisted")
	}
}

func TestUpdateNoticeIsAdminOnlyAndNotifiedOnce(t *testing.T) {
	f, room, admin, member := communityFixture(t)
	manager, err := updatemanager.NewManager(updatemanager.Config{Repository: "mythezone/navidrome-music-room", CurrentVersion: "1.1.0", DataDir: f.api.config.DataDir, CosignBinary: "cosign", TrustedRoot: "root", IdentityRegex: "identity"}, f.store)
	if err != nil {
		t.Fatal(err)
	}
	f.api.updater = manager
	status := updatemanager.Status{CurrentVersion: "1.1.0", LatestVersion: "v1.2.0", Channel: "stable", UpdateAvailable: true, ReleaseURL: "https://github.com/mythezone/navidrome-music-room/releases/tag/v1.2.0", PublishedAt: "2026-11-18T00:00:00Z"}
	if err = f.store.PutUpdateState(t.Context(), "status", status); err != nil {
		t.Fatal(err)
	}
	base := "/api/v1/rooms/" + room.RoomID
	var snapshot domain.Snapshot
	f.expectJSON(t, "GET", base+"/snapshot", admin, nil, 200, &snapshot)
	if snapshot.UpdateNotice == nil || snapshot.UpdateNotice.LatestVersion != "v1.2.0" {
		t.Fatal("missing admin notice")
	}
	snapshot = domain.Snapshot{}
	f.expectJSON(t, "GET", base+"/snapshot", member, nil, 200, &snapshot)
	if snapshot.UpdateNotice != nil {
		t.Fatal("update notice leaked to member")
	}
	f.expectJSON(t, "GET", "/api/v1/admin/updates", member, nil, 403, nil)
	var updates struct {
		PluginVersion   string `json:"pluginVersion"`
		PluginPaired    bool   `json:"pluginPaired"`
		VersionMismatch bool   `json:"versionMismatch"`
	}
	f.expectJSON(t, "GET", "/api/v1/admin/updates", admin, nil, 200, &updates)
	if !updates.PluginPaired || updates.VersionMismatch || updates.PluginVersion != "test" {
		t.Fatalf("plugin status: %+v", updates)
	}
	var syncResponse map[string]any
	f.expectJSON(t, "POST", "/internal/v1/plugin-sync", f.pairing, domain.PluginSync{PluginVersion: "v1.2.0", Generation: 2, NavidromeInternalURL: f.navidromeInternal, UpdateChannel: "stable", Users: []domain.PluginUser{{Username: "admin", Admin: true}, {Username: "member"}}}, 200, &syncResponse)
	if syncResponse["updateAvailable"] != true || syncResponse["latestVersion"] != "v1.2.0" {
		t.Fatalf("sync status: %+v", syncResponse)
	}
	f.expectJSON(t, "GET", "/api/v1/admin/updates", admin, nil, 200, &updates)
	if !updates.VersionMismatch {
		t.Fatal("version mismatch not reported")
	}
	adminWS := f.connectWebSocket(t, room.RoomID, admin)
	defer adminWS.Close()
	memberWS := f.connectWebSocket(t, room.RoomID, member)
	defer memberWS.Close()
	readWebSocketEvent(t, adminWS, "snapshot")
	readWebSocketEvent(t, memberWS, "snapshot")
	received := make(chan map[string]string, 2)
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		received <- payload
		w.WriteHeader(204)
	}))
	defer webhook.Close()
	f.api.config.UpdateWebhookURL = webhook.URL
	if err = f.api.notifyUpdate(t.Context(), status); err != nil {
		t.Fatal(err)
	}
	if err = f.api.notifyUpdate(t.Context(), status); err != nil {
		t.Fatal(err)
	}
	readWebSocketEvent(t, adminWS, "update_available")
	f.api.hub.Broadcast(room.RoomID, domain.Event{Type: "test_marker", Payload: map[string]any{}})
	for {
		memberWS.SetReadDeadline(time.Now().Add(time.Second))
		var event webSocketEvent
		if err = memberWS.ReadJSON(&event); err != nil {
			t.Fatal(err)
		}
		if event.Type == "update_available" {
			t.Fatal("non-admin received update push")
		}
		if event.Type == "test_marker" {
			break
		}
	}
	if len(received) != 1 {
		t.Fatalf("expected one webhook, got %d", len(received))
	}
	payload := <-received
	if payload["event"] != "update_available" || payload["latestVersion"] != "v1.2.0" || payload["publishedAt"] != status.PublishedAt {
		t.Fatalf("webhook: %+v", payload)
	}
	f.expectJSON(t, "POST", "/internal/v1/plugin-sync", f.pairing, domain.PluginSync{PluginVersion: "v1.2.0", Generation: 3, NavidromeInternalURL: f.navidromeInternal, UpdateChannel: "beta", Users: []domain.PluginUser{{Username: "admin", Admin: true}, {Username: "member"}}}, 200, nil)
	select {
	case <-f.api.updateChecks:
	default:
		t.Fatal("channel change did not schedule check")
	}
}
