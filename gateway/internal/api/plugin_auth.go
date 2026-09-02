package api

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type authExchangeRequest struct {
	Username string `json:"username"`
	Salt     string `json:"salt"`
	Token    string `json:"token"`
}

func (s *Server) pluginSync(w http.ResponseWriter, r *http.Request) {
	if !s.pluginAuthorized(r) {
		writeError(w, domain.NewError(401, "plugin_pairing_failed", "Plugin pairing token is invalid"))
		return
	}
	var input domain.PluginSync
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	if input.Generation <= 0 || len(input.Users) > 10000 || strings.TrimSpace(input.PluginVersion) == "" {
		writeError(w, domain.NewError(400, "plugin_sync_invalid", "Plugin sync payload is invalid"))
		return
	}
	// Accept an older bridge payload during rolling upgrades while keeping its
	// retired field empty in existing databases.
	input.LicenseFile = ""
	input.UpdateChannel = strings.ToLower(strings.TrimSpace(input.UpdateChannel))
	if input.UpdateChannel == "" {
		input.UpdateChannel = "stable"
	}
	if input.UpdateChannel != "stable" && input.UpdateChannel != "beta" {
		writeError(w, domain.NewError(400, "plugin_sync_invalid", "Update channel must be stable or beta"))
		return
	}
	seen := map[string]struct{}{}
	users := make([]domain.PluginUser, 0, len(input.Users))
	for _, user := range input.Users {
		user.Username = strings.TrimSpace(user.Username)
		user.DisplayName = strings.Join(strings.Fields(user.DisplayName), " ")
		if user.Username == "" || len(user.Username) > 128 {
			writeError(w, domain.NewError(400, "plugin_sync_invalid", "Plugin user has an invalid username"))
			return
		}
		key := strings.ToLower(user.Username)
		if _, duplicate := seen[key]; duplicate {
			writeError(w, domain.NewError(400, "plugin_sync_invalid", "Plugin user list contains duplicate usernames"))
			return
		}
		seen[key] = struct{}{}
		if user.DisplayName == "" {
			user.DisplayName = user.Username
		}
		users = append(users, user)
	}
	input.Users = users
	if strings.TrimRight(input.NavidromeInternalURL, "/") != s.config.NavidromeInternal.String() {
		writeError(w, domain.NewError(409, "configuration_mismatch", "Plugin and gateway Navidrome internal URLs do not match"))
		return
	}
	if input.NavidromePublicURL != "" && strings.TrimRight(input.NavidromePublicURL, "/") != s.config.NavidromePublic.String() {
		writeError(w, domain.NewError(409, "configuration_mismatch", "Plugin and gateway Navidrome public URLs do not match"))
		return
	}
	if input.GatewayPublicURL != "" && strings.TrimRight(input.GatewayPublicURL, "/") != s.config.GatewayPublic.String() {
		writeError(w, domain.NewError(409, "configuration_mismatch", "Plugin and gateway public URLs do not match"))
		return
	}
	previous, _ := s.store.PluginState(r.Context())
	state, err := s.store.SavePluginSync(r.Context(), input, time.Now().UTC())
	if err != nil {
		writeError(w, err)
		return
	}
	for _, previousUser := range previous.Users {
		if _, stillAllowed := state.User(previousUser.Username); !stillAllowed {
			s.sessions.RevokeUser(previousUser.Username)
			s.hub.KickUserEverywhere(previousUser.Username)
		}
	}
	writeJSON(w, 200, map[string]any{
		"acceptedGeneration": state.Generation,
		"leaseExpiresAt":     state.LastHeartbeat.Add(s.config.PluginLease),
		"gatewayVersion":     s.config.Version,
	})
}

func (s *Server) authExchange(w http.ResponseWriter, r *http.Request) {
	key := clientIP(r, s.config.TrustProxy)
	if !s.authLimiter.Allow(key) {
		writeError(w, domain.NewError(429, "auth_rate_limited", "Too many authentication attempts"))
		return
	}
	var input authExchangeRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.sessions.Exchange(r.Context(), domain.AuthProof{
		Username: strings.TrimSpace(input.Username), Salt: strings.TrimSpace(input.Salt), Token: strings.TrimSpace(input.Token),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	_ = s.store.Audit(r.Context(), result.User.Username, "", "auth.exchanged", nil)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, result)
}

func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	session, token, err := s.session(r)
	if err != nil {
		writeError(w, err)
		return
	}
	s.sessions.Revoke(token)
	_ = s.store.Audit(r.Context(), session.User.Username, "", "auth.revoked", nil)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) featureAvailability(w http.ResponseWriter, r *http.Request) {
	if _, _, err := s.session(r); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{
		"project": map[string]any{"openSource": true, "spdxLicense": "GPL-3.0-only"},
		"features": map[string]any{
			"rooms": true, "invites": true, "presence": true, "synchronizedPlayback": true,
			"queue": true, "history": true, "chat": false, "stickers": false,
			"statistics": false, "rankings": false, "achievements": false, "uploads": false, "onlineSources": false,
		},
	})
}

func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
			return forwarded
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func websocketURL(base *url.URL, roomID, ticket string) string {
	result := *base
	if result.Scheme == "https" {
		result.Scheme = "wss"
	} else {
		result.Scheme = "ws"
	}
	result.Path = strings.TrimRight(result.Path, "/") + "/api/v1/rooms/" + roomID + "/ws"
	query := result.Query()
	query.Set("ticket", ticket)
	result.RawQuery = query.Encode()
	return result.String()
}
