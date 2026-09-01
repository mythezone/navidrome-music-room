package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type rateWindow struct {
	started time.Time
	count   int
}

func (s *Server) mutationRateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions ||
			r.URL.Path == "/api/v1/auth/exchange" || !strings.HasPrefix(r.URL.Path, "/api/v1/") {
			next.ServeHTTP(w, r)
			return
		}
		limiter := s.mutationLimiter
		code := "mutation_rate_limited"
		if r.URL.Path == "/api/v1/invites/redeem" {
			limiter = s.inviteLimiter
			code = "invite_rate_limited"
		}
		if !limiter.Allow(s.rateLimitPrincipal(r)) {
			w.Header().Set("Retry-After", "60")
			writeError(w, domain.NewError(429, code, "Too many write requests"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) rateLimitPrincipal(r *http.Request) string {
	if session, _, err := s.session(r); err == nil {
		return "user:" + strings.ToLower(strings.TrimSpace(session.User.Username))
	}
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); authorization != "" {
		digest := sha256.Sum256([]byte(authorization))
		return "session:" + hex.EncodeToString(digest[:])
	}
	return "ip:" + clientIP(r, s.config.TrustProxy)
}

type rateLimiter struct {
	mu       sync.Mutex
	items    map[string]rateWindow
	limit    int
	interval time.Duration
	now      func() time.Time
}

func newRateLimiter(limit int, interval time.Duration) *rateLimiter {
	return &rateLimiter{items: map[string]rateWindow{}, limit: limit, interval: interval, now: time.Now}
}

func (r *rateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	if _, exists := r.items[key]; !exists && len(r.items) >= 10000 {
		for itemKey, item := range r.items {
			if now.Sub(item.started) >= r.interval {
				delete(r.items, itemKey)
			}
		}
		if len(r.items) >= 10000 {
			return false
		}
	}
	window := r.items[key]
	if window.started.IsZero() || now.Sub(window.started) >= r.interval {
		window = rateWindow{started: now}
	}
	window.count++
	r.items[key] = window
	return window.count <= r.limit
}
