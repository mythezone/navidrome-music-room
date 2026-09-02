package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mythezone/navidrome-music-room/gateway/internal/adminui"
	"github.com/mythezone/navidrome-music-room/gateway/internal/auth"
	"github.com/mythezone/navidrome-music-room/gateway/internal/config"
	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
	"github.com/mythezone/navidrome-music-room/gateway/internal/realtime"
	"github.com/mythezone/navidrome-music-room/gateway/internal/roomui"
	"github.com/mythezone/navidrome-music-room/gateway/internal/store"
	updatemanager "github.com/mythezone/navidrome-music-room/gateway/internal/update"
)

type Server struct {
	config          config.Config
	store           *store.Store
	sessions        *auth.SessionManager
	hub             *realtime.Hub
	tickets         *realtime.Tickets
	logger          *slog.Logger
	mux             *http.ServeMux
	upgrader        websocket.Upgrader
	authLimiter     *rateLimiter
	mutationLimiter *rateLimiter
	inviteLimiter   *rateLimiter
	idempotency     *idempotencyCache
	updater         *updatemanager.Manager
	restart         func()
}

func NewServer(cfg config.Config, storage *store.Store, sessions *auth.SessionManager, logger *slog.Logger) (*Server, error) {
	updater, err := updatemanager.NewManager(updatemanager.Config{
		Repository: cfg.ReleaseRepository, CurrentVersion: cfg.Version, DataDir: cfg.DataDir,
		CosignBinary: cfg.CosignBinary, TrustedRoot: cfg.SigstoreTrustedRoot, IdentityRegex: cfg.UpdateIdentity,
	}, storage)
	if err != nil {
		return nil, err
	}
	server := &Server{
		config: cfg, store: storage, sessions: sessions, tickets: realtime.NewTickets(cfg.WebSocketTicketTTL),
		logger: logger, mux: http.NewServeMux(), authLimiter: newRateLimiter(10, time.Minute),
		mutationLimiter: newRateLimiter(120, time.Minute), inviteLimiter: newRateLimiter(30, time.Minute),
		idempotency: newIdempotencyCache(15*time.Minute, 2048), updater: updater,
	}
	server.hub = realtime.NewHub(cfg.EmptyRoomPauseDelay, server.pauseForEmpty, logger)
	server.upgrader = websocket.Upgrader{
		HandshakeTimeout: 8 * time.Second,
		ReadBufferSize:   4096, WriteBufferSize: 16384,
		CheckOrigin: server.originAllowed,
	}
	server.routes()
	return server, nil
}

func (s *Server) SetRestartCallback(callback func()) { s.restart = callback }

func (s *Server) Handler() http.Handler {
	return s.recoverMiddleware(s.securityHeaders(s.corsMiddleware(s.loggingMiddleware(
		s.idempotencyMiddleware(s.mutationRateMiddleware(s.mux)),
	))))
}

func (s *Server) routes() {
	s.mux.Handle("GET /admin", adminui.Handler())
	s.mux.Handle("GET /admin/{path...}", adminui.Handler())
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /readyz", s.ready)
	s.mux.HandleFunc("GET /join/{room_id}", s.redirectRoomUI)
	s.mux.Handle("GET /join/{room_id}/", roomui.Handler())
	s.mux.HandleFunc("GET /api/v1/discovery", s.discovery)
	s.mux.HandleFunc("POST /internal/v1/plugin-sync", s.pluginSync)
	s.mux.HandleFunc("POST /api/v1/auth/exchange", s.authExchange)
	s.mux.HandleFunc("DELETE /api/v1/auth/session", s.authLogout)
	s.mux.HandleFunc("GET /api/v1/rooms", s.listRooms)
	s.mux.HandleFunc("POST /api/v1/rooms", s.createRoom)
	s.mux.HandleFunc("GET /api/v1/rooms/{room_id}", s.getRoom)
	s.mux.HandleFunc("PATCH /api/v1/rooms/{room_id}", s.patchRoom)
	s.mux.HandleFunc("DELETE /api/v1/rooms/{room_id}", s.deleteRoom)
	s.mux.HandleFunc("POST /api/v1/rooms/{room_id}/close", s.closeRoom)
	s.mux.HandleFunc("POST /api/v1/rooms/{room_id}/reopen", s.reopenRoom)
	s.mux.HandleFunc("GET /api/v1/rooms/{room_id}/invites", s.listInvites)
	s.mux.HandleFunc("POST /api/v1/rooms/{room_id}/invites", s.createInvite)
	s.mux.HandleFunc("DELETE /api/v1/rooms/{room_id}/invites/{invite_id}", s.revokeInvite)
	s.mux.HandleFunc("POST /api/v1/invites/redeem", s.redeemInvite)
	s.mux.HandleFunc("GET /api/v1/rooms/{room_id}/members", s.listMembers)
	s.mux.HandleFunc("DELETE /api/v1/rooms/{room_id}/members/{username}", s.removeMember)
	s.mux.HandleFunc("POST /api/v1/rooms/{room_id}/join", s.joinRoom)
	s.mux.HandleFunc("POST /api/v1/rooms/{room_id}/leave", s.leaveRoom)
	s.mux.HandleFunc("GET /api/v1/rooms/{room_id}/snapshot", s.snapshot)
	s.mux.HandleFunc("GET /api/v1/rooms/{room_id}/history", s.history)
	s.mux.HandleFunc("POST /api/v1/rooms/{room_id}/playback", s.playback)
	s.mux.HandleFunc("POST /api/v1/rooms/{room_id}/queue/tracks", s.addQueueTrack)
	s.mux.HandleFunc("DELETE /api/v1/rooms/{room_id}/queue/{queue_id}", s.removeQueueEntry)
	s.mux.HandleFunc("PUT /api/v1/rooms/{room_id}/queue/order", s.reorderQueue)
	s.mux.HandleFunc("POST /api/v1/rooms/{room_id}/ws-ticket", s.issueWebSocketTicket)
	s.mux.HandleFunc("GET /api/v1/rooms/{room_id}/ws", s.webSocket)
	s.mux.HandleFunc("GET /api/v1/capabilities", s.featureAvailability)
	// Keep the v1.0 route during the protocol transition. New clients use /capabilities.
	s.mux.HandleFunc("GET /api/v1/entitlements", s.featureAvailability)
	s.mux.HandleFunc("GET /api/v1/admin/diagnostics", s.exportDiagnostics)
	s.mux.HandleFunc("GET /api/v1/admin/updates", s.updateStatus)
	s.mux.HandleFunc("POST /api/v1/admin/updates", s.updateAction)
	for _, path := range []string{
		"/api/v1/rooms/{room_id}/chat", "/api/v1/rooms/{room_id}/statistics",
		"/api/v1/rooms/{room_id}/rankings", "/api/v1/rooms/{room_id}/achievements",
		"/api/v1/rooms/{room_id}/stickers",
	} {
		s.mux.HandleFunc(path, s.unimplementedFeature)
	}
}

func (s *Server) RunClock(ctx context.Context) {
	roomIDs, err := s.store.ListIdleQueuedRoomIDs(ctx)
	if err != nil {
		s.logger.Error("idle queue recovery list failed", "error", err)
	} else {
		for _, roomID := range roomIDs {
			state, changed, primeErr := s.store.PrimePlaybackIfIdle(ctx, roomID, "__recovery__")
			if primeErr != nil {
				s.logger.Error("idle queue recovery failed", "room_id", roomID, "error", primeErr)
				continue
			}
			if changed {
				s.logger.Info("idle queue promoted to current playback", "room_id", roomID, "revision", state.Revision)
				s.hub.Broadcast(roomID, domain.Event{Type: "playback", Revision: state.Revision, Payload: state})
				if queue, queueErr := s.store.ListQueue(ctx, roomID); queueErr == nil {
					s.hub.Broadcast(roomID, domain.Event{Type: "queue", Payload: queue})
				}
			}
		}
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			roomIDs, err := s.store.ListPlayingRoomIDs(ctx)
			if err != nil {
				s.logger.Error("playback clock list failed", "error", err)
				continue
			}
			for _, roomID := range roomIDs {
				state, changed, err := s.store.AdvanceFinished(ctx, roomID)
				if err != nil {
					s.logger.Error("playback clock advance failed", "room_id", roomID, "error", err)
					continue
				}
				if changed {
					s.hub.Broadcast(roomID, domain.Event{Type: "playback", Revision: state.Revision, Payload: state})
					if queue, err := s.store.ListQueue(ctx, roomID); err == nil {
						s.hub.Broadcast(roomID, domain.Event{Type: "queue", Payload: queue})
					}
					if history, err := s.store.History(ctx, roomID, 50, 0); err == nil {
						s.hub.Broadcast(roomID, domain.Event{Type: "history", Payload: history})
					}
				}
			}
		}
	}
}

func (s *Server) pauseForEmpty(ctx context.Context, roomID string) {
	state, changed, err := s.store.PauseForEmpty(ctx, roomID)
	if err != nil {
		s.logger.Error("empty room pause failed", "room_id", roomID, "error", err)
		return
	}
	if changed {
		s.hub.Broadcast(roomID, domain.Event{Type: "playback", Revision: state.Revision, Payload: state})
	}
}

func (s *Server) session(r *http.Request) (domain.Session, string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return domain.Session{}, "", domain.NewError(401, "session_required", "A room session bearer token is required")
	}
	token := strings.TrimSpace(header[len("Bearer "):])
	session, err := s.sessions.Authenticate(r.Context(), token)
	return session, token, err
}

func (s *Server) requireRoomAccess(r *http.Request, requireMember bool) (domain.Session, domain.Room, *domain.Member, error) {
	session, _, err := s.session(r)
	if err != nil {
		return domain.Session{}, domain.Room{}, nil, err
	}
	roomID := r.PathValue("room_id")
	if err := domain.ValidateID(roomID); err != nil {
		return domain.Session{}, domain.Room{}, nil, domain.NewError(400, "room_id_invalid", "Room ID is invalid")
	}
	room, err := s.store.GetRoom(r.Context(), roomID)
	if err != nil {
		return domain.Session{}, domain.Room{}, nil, err
	}
	member, memberErr := s.store.ActiveMember(r.Context(), roomID, session.User.Username)
	if memberErr == nil {
		return session, room, &member, nil
	}
	if session.User.Admin && !requireMember {
		return session, room, nil, nil
	}
	return domain.Session{}, domain.Room{}, nil, memberErr
}

func requireManager(session domain.Session, room domain.Room) error {
	if session.User.Admin || domain.EqualUsername(session.User.Username, room.OwnerUsername) {
		return nil
	}
	return domain.NewError(403, "room_manager_required", "Room owner or Navidrome administrator permission is required")
}

func requireLibraryAccess(user domain.User, room domain.Room) error {
	if domain.ContainsAllFolders(user.MusicFolderIDs, room.MusicFolderIDs) {
		return nil
	}
	return domain.ErrorWithDetails(403, "library_access_required", "This Navidrome user cannot access every music folder used by the room", map[string]any{
		"requiredMusicFolderIDs": room.MusicFolderIDs,
		"userMusicFolderIDs":     user.MusicFolderIDs,
	})
}

func (s *Server) pluginAuthorized(r *http.Request) bool {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return false
	}
	provided := strings.TrimSpace(header[len("Bearer "):])
	expected := s.config.PluginPairingToken
	if len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return domain.NewError(400, "request_invalid", "Request body is not valid JSON")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return domain.NewError(400, "request_invalid", "Request body must contain one JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	var roomError *domain.Error
	if !errors.As(err, &roomError) {
		roomError = domain.NewError(500, "internal_error", "An internal server error occurred")
	}
	payload := map[string]any{"code": roomError.Code, "message": roomError.Message}
	if roomError.Details != nil {
		payload["details"] = roomError.Details
	}
	writeJSON(w, roomError.Status, map[string]any{"error": payload})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("request panic", "path", r.URL.Path, "panic", recovered, "stack", string(debug.Stack()))
				writeError(w, domain.NewError(500, "internal_error", "An internal server error occurred"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(started).Milliseconds())
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if !s.originAllowed(r) {
				writeError(w, domain.NewError(403, "origin_forbidden", "Request origin is not allowed"))
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	if parsed, err := url.Parse(origin); err == nil {
		for _, sameOriginService := range []*url.URL{s.config.NavidromePublic, s.config.GatewayPublic} {
			if sameOriginService != nil && strings.EqualFold(parsed.Scheme, sameOriginService.Scheme) && strings.EqualFold(parsed.Host, sameOriginService.Host) {
				return true
			}
		}
	}
	for _, allowed := range s.config.AllowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "version": s.config.Version, "time": time.Now().UTC()})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeError(w, domain.NewError(503, "database_unavailable", "Room database is unavailable"))
		return
	}
	state, err := s.store.PluginState(r.Context())
	pluginStatus := "paired"
	if err != nil || time.Since(state.LastHeartbeat) > s.config.PluginLease {
		pluginStatus = "stale"
	}
	writeJSON(w, 200, map[string]any{
		"status": "ready", "plugin": pluginStatus, "pluginVersion": state.PluginVersion, "version": s.config.Version,
	})
}

func (s *Server) discovery(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, map[string]any{
		"apiVersion":         "v1",
		"version":            s.config.Version,
		"gatewayBaseURL":     s.config.GatewayPublic.String(),
		"navidromeBaseURL":   s.config.NavidromePublic.String(),
		"authenticationMode": "opensubsonic-proof-exchange",
	})
}

func featureFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "unknown"
	}
	return parts[len(parts)-1]
}

func (s *Server) unimplementedFeature(w http.ResponseWriter, r *http.Request) {
	if _, _, err := s.session(r); err != nil {
		writeError(w, err)
		return
	}
	writeError(w, domain.FeatureNotImplemented(featureFromPath(r.URL.Path)))
}

func (s *Server) redirectRoomUI(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("room_id")
	if domain.ValidateID(roomID) != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	// A relative Location retains reverse-proxy prefixes such as /music-room and
	// browsers carry the invitation fragment across the redirect.
	w.Header().Set("Location", roomID+"/")
	w.WriteHeader(http.StatusPermanentRedirect)
}
