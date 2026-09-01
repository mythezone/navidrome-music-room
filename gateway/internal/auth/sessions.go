package auth

import (
	"context"
	"sync"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type PluginStateStore interface {
	PluginState(context.Context) (domain.PluginState, error)
}

type SessionManager struct {
	mu              sync.RWMutex
	sessions        map[string]domain.Session
	pluginStore     PluginStateStore
	navidrome       *NavidromeClient
	sessionTTL      time.Duration
	leaseTTL        time.Duration
	existingGrace   time.Duration
	navidromePublic string
	gatewayPublic   string
	now             func() time.Time
}

func NewSessionManager(
	pluginStore PluginStateStore,
	navidrome *NavidromeClient,
	sessionTTL, leaseTTL, existingGrace time.Duration,
	navidromePublic, gatewayPublic string,
) *SessionManager {
	return &SessionManager{
		sessions: map[string]domain.Session{}, pluginStore: pluginStore, navidrome: navidrome,
		sessionTTL: sessionTTL, leaseTTL: leaseTTL, existingGrace: existingGrace,
		navidromePublic: navidromePublic, gatewayPublic: gatewayPublic, now: func() time.Time { return time.Now().UTC() },
	}
}

func (m *SessionManager) Exchange(ctx context.Context, proof domain.AuthProof) (domain.AuthExchange, error) {
	state, err := m.freshPluginState(ctx, false)
	if err != nil {
		return domain.AuthExchange{}, err
	}
	user, err := m.navidrome.Verify(ctx, proof)
	if err != nil {
		return domain.AuthExchange{}, err
	}
	pluginUser, allowed := state.User(user.Username)
	if !allowed {
		return domain.AuthExchange{}, domain.NewError(403, "plugin_user_not_allowed", "This Navidrome user is not authorized for the plugin")
	}
	if pluginUser.DisplayName != "" {
		user.DisplayName = pluginUser.DisplayName
	}
	user.Admin = pluginUser.Admin && user.Admin
	token, err := domain.NewToken()
	if err != nil {
		return domain.AuthExchange{}, err
	}
	now := m.now()
	session := domain.Session{
		ID: token, User: user, Proof: proof, Generation: state.Generation,
		CreatedAt: now, ExpiresAt: now.Add(m.sessionTTL),
	}
	m.mu.Lock()
	m.sessions[token] = session
	for key, existing := range m.sessions {
		if !existing.ExpiresAt.After(now) {
			delete(m.sessions, key)
		}
	}
	m.mu.Unlock()
	return domain.AuthExchange{
		SessionToken: token, ExpiresAt: session.ExpiresAt, User: user,
		NavidromeBaseURL: m.navidromePublic, GatewayBaseURL: m.gatewayPublic,
		PluginLeaseExpires: state.LastHeartbeat.Add(m.leaseTTL),
	}, nil
}

func (m *SessionManager) Authenticate(ctx context.Context, token string) (domain.Session, error) {
	if token == "" {
		return domain.Session{}, domain.NewError(401, "session_required", "A room session bearer token is required")
	}
	m.mu.RLock()
	session, ok := m.sessions[token]
	m.mu.RUnlock()
	now := m.now()
	if !ok || !session.ExpiresAt.After(now) {
		if ok {
			m.mu.Lock()
			delete(m.sessions, token)
			m.mu.Unlock()
		}
		return domain.Session{}, domain.NewError(401, "session_expired", "Room session has expired")
	}
	state, err := m.freshPluginState(ctx, true)
	if err != nil {
		return domain.Session{}, err
	}
	pluginUser, allowed := state.User(session.User.Username)
	if !allowed {
		m.mu.Lock()
		delete(m.sessions, token)
		m.mu.Unlock()
		return domain.Session{}, domain.NewError(403, "plugin_user_not_allowed", "This Navidrome user is no longer authorized for the plugin")
	}
	session.User.Admin = pluginUser.Admin && session.User.Admin
	if pluginUser.DisplayName != "" {
		session.User.DisplayName = pluginUser.DisplayName
	}
	m.mu.Lock()
	m.sessions[token] = session
	m.mu.Unlock()
	return session, nil
}

func (m *SessionManager) RequireFreshLease(ctx context.Context) error {
	_, err := m.freshPluginState(ctx, false)
	return err
}

func (m *SessionManager) ValidateTrack(ctx context.Context, session domain.Session, track domain.NavidromeTrackRef) (domain.NavidromeTrackRef, error) {
	return m.navidrome.ValidateTrack(ctx, session.Proof, track)
}

func (m *SessionManager) Revoke(token string) {
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
}

func (m *SessionManager) RevokeUser(username string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for token, session := range m.sessions {
		if domain.EqualUsername(session.User.Username, username) {
			delete(m.sessions, token)
		}
	}
}

func (m *SessionManager) freshPluginState(ctx context.Context, existing bool) (domain.PluginState, error) {
	state, err := m.pluginStore.PluginState(ctx)
	if err != nil {
		return domain.PluginState{}, err
	}
	maxAge := m.leaseTTL
	if existing {
		maxAge += m.existingGrace
	}
	if m.now().Sub(state.LastHeartbeat) > maxAge {
		return domain.PluginState{}, domain.ErrorWithDetails(503, "plugin_lease_expired", "Navidrome plugin heartbeat is stale", map[string]any{
			"lastHeartbeat":     state.LastHeartbeat,
			"maximumAgeSeconds": int(maxAge.Seconds()),
		})
	}
	return state, nil
}
