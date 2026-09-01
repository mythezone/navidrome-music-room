package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

const maxIdempotentBodyBytes = 1 << 20

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

type cachedHTTPResponse struct {
	status    int
	header    http.Header
	body      []byte
	expiresAt time.Time
}

type idempotencyEntry struct {
	requestHash string
	done        chan struct{}
	response    *cachedHTTPResponse
}

type idempotencyCache struct {
	mu      sync.Mutex
	items   map[string]*idempotencyEntry
	ttl     time.Duration
	maxSize int
	now     func() time.Time
}

func newIdempotencyCache(ttl time.Duration, maxSize int) *idempotencyCache {
	return &idempotencyCache{
		items: make(map[string]*idempotencyEntry), ttl: ttl, maxSize: maxSize, now: time.Now,
	}
}

func (c *idempotencyCache) begin(scope, requestHash string) (*idempotencyEntry, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cleanupLocked()
	if entry, ok := c.items[scope]; ok {
		if entry.requestHash != requestHash {
			return nil, false, domain.NewError(409, "idempotency_key_reused", "Idempotency-Key was already used with a different request")
		}
		return entry, false, nil
	}
	if len(c.items) >= c.maxSize {
		return nil, false, domain.NewError(503, "idempotency_capacity_exceeded", "Idempotency cache is temporarily full")
	}
	entry := &idempotencyEntry{requestHash: requestHash, done: make(chan struct{})}
	c.items[scope] = entry
	return entry, true, nil
}

func (c *idempotencyCache) wait(ctx context.Context, entry *idempotencyEntry) (*cachedHTTPResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-entry.done:
		c.mu.Lock()
		defer c.mu.Unlock()
		return cloneCachedResponse(entry.response), nil
	}
}

func (c *idempotencyCache) commit(scope string, entry *idempotencyEntry, response *cachedHTTPResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if current, ok := c.items[scope]; !ok || current != entry {
		return
	}
	entry.response = cloneCachedResponse(response)
	close(entry.done)
}

func (c *idempotencyCache) abort(scope string, entry *idempotencyEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if current, ok := c.items[scope]; !ok || current != entry {
		return
	}
	delete(c.items, scope)
	close(entry.done)
}

func (c *idempotencyCache) cleanupLocked() {
	now := c.now()
	for key, entry := range c.items {
		if entry.response != nil && !entry.response.expiresAt.After(now) {
			delete(c.items, key)
		}
	}
}

func cloneCachedResponse(input *cachedHTTPResponse) *cachedHTTPResponse {
	if input == nil {
		return nil
	}
	return &cachedHTTPResponse{
		status: input.status, header: input.header.Clone(), body: bytes.Clone(input.body), expiresAt: input.expiresAt,
	}
}

type bufferedResponseWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header)}
}

func (w *bufferedResponseWriter) Header() http.Header { return w.header }

func (w *bufferedResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *bufferedResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(body)
}

func (s *Server) idempotencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if key == "" || !supportsIdempotency(r) {
			next.ServeHTTP(w, r)
			return
		}
		if !idempotencyKeyPattern.MatchString(key) {
			writeError(w, domain.NewError(400, "idempotency_key_invalid", "Idempotency-Key must contain 8 to 128 URL-safe characters"))
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, maxIdempotentBodyBytes+1))
		if err != nil {
			writeError(w, domain.NewError(400, "request_invalid", "Request body could not be read"))
			return
		}
		if len(body) > maxIdempotentBodyBytes {
			writeError(w, domain.NewError(413, "request_too_large", "Request body exceeds one MiB"))
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		scope := s.idempotencyScope(r, key)
		hash := idempotencyRequestHash(r, body)
		for {
			entry, owner, beginErr := s.idempotency.begin(scope, hash)
			if beginErr != nil {
				writeError(w, beginErr)
				return
			}
			if !owner {
				cached, waitErr := s.idempotency.wait(r.Context(), entry)
				if waitErr != nil {
					writeError(w, domain.NewError(499, "request_cancelled", "Request was cancelled while waiting for an idempotent operation"))
					return
				}
				if cached == nil {
					continue
				}
				cached.header.Set("Idempotency-Replayed", "true")
				writeBufferedResponse(w, cached.status, cached.header, cached.body)
				return
			}

			recorder := newBufferedResponseWriter()
			completed := false
			defer func() {
				if !completed {
					s.idempotency.abort(scope, entry)
				}
			}()
			next.ServeHTTP(recorder, r)
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			responseBody := recorder.body.Bytes()
			if status < 500 && len(responseBody) <= maxIdempotentBodyBytes {
				s.idempotency.commit(scope, entry, &cachedHTTPResponse{
					status: status, header: recorder.header, body: responseBody, expiresAt: time.Now().Add(s.idempotency.ttl),
				})
			} else {
				s.idempotency.abort(scope, entry)
			}
			completed = true
			writeBufferedResponse(w, status, recorder.header, responseBody)
			return
		}
	})
}

func supportsIdempotency(r *http.Request) bool {
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch && r.Method != http.MethodDelete {
		return false
	}
	return strings.HasPrefix(r.URL.Path, "/api/v1/rooms") ||
		r.URL.Path == "/api/v1/invites/redeem" || r.URL.Path == "/api/v1/admin/updates"
}

func (s *Server) idempotencyScope(r *http.Request, key string) string {
	credential := sha256.Sum256([]byte(r.Header.Get("Authorization")))
	principal := "session:" + hex.EncodeToString(credential[:])
	if session, _, err := s.session(r); err == nil {
		principal = "user:" + strings.ToLower(strings.TrimSpace(session.User.Username))
	}
	return principal + "\n" + r.Method + "\n" + r.URL.Path + "\n" + key
}

func idempotencyRequestHash(r *http.Request, body []byte) string {
	digest := sha256.New()
	_, _ = io.WriteString(digest, r.Method+"\n"+r.URL.EscapedPath()+"?"+r.URL.RawQuery+"\n"+r.Header.Get("Content-Type")+"\n")
	_, _ = digest.Write(body)
	return hex.EncodeToString(digest.Sum(nil))
}

func writeBufferedResponse(w http.ResponseWriter, status int, header http.Header, body []byte) {
	for key, values := range header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
