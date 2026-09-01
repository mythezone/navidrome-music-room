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
)

type apiFixture struct {
	server            *httptest.Server
	store             *store.Store
	pairing           string
	navidromeInternal string
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
		CosignBinary: "cosign", UpdateIdentity: "https://github.com/mythezone/navidrome-music-room/.github/workflows/release.yml@refs/tags/.*",
		PluginLease: 90 * time.Second, ExistingGrace: time.Minute, SessionTTL: 15 * time.Minute,
		WebSocketTicketTTL: time.Minute, EmptyRoomPauseDelay: 15 * time.Second,
	}
	sessions := auth.NewSessionManager(storage, auth.NewNavidromeClient(internal), cfg.SessionTTL, cfg.PluginLease, cfg.ExistingGrace, navidromePublic.String(), gatewayPublic.String())
	server, err := NewServer(cfg, storage, sessions, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	fixture := apiFixture{server: httpServer, store: storage, pairing: pairing, navidromeInternal: internal.String()}
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
		`"username"`, `"displayName"`, `"licenseID"`, `"subject"`,
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
	if response.StatusCode != 402 {
		t.Fatalf("chat should be license locked: %d %s", response.StatusCode, readBody(response))
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
		if payload.Playback.Revision != 0 {
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
		"command": "play", "expectedRevision": 0,
	}, adminToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("play failed: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	for _, connection := range connections {
		event := readWebSocketEvent(t, connection, "playback")
		if event.Revision != 1 {
			t.Fatalf("client did not receive playback revision 1: %#v", event)
		}
	}

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/playback", map[string]any{
		"command": "seek", "expectedRevision": 1, "positionSeconds": 42,
	}, adminToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("seek failed: %d %s", response.StatusCode, readBody(response))
	}
	response.Body.Close()
	for _, connection := range connections {
		event := readWebSocketEvent(t, connection, "playback")
		if event.Revision != 2 {
			t.Fatalf("client did not receive seek revision 2: %#v", event)
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
	if restored.Playback.Revision != 2 || restored.Playback.PositionSeconds < 42 {
		t.Fatalf("reconnect did not restore authoritative playback: %#v", restored.Playback)
	}

	response = fixture.request(t, http.MethodPost, "/api/v1/rooms/"+room.RoomID+"/playback", map[string]any{
		"command": "pause", "expectedRevision": 1,
	}, adminToken)
	defer response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("stale revision overwrote newer playback: %d %s", response.StatusCode, readBody(response))
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
