package realtime

import (
	"sync"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type Ticket struct {
	SessionToken string
	RoomID       string
	ExpiresAt    time.Time
}

type Tickets struct {
	mu    sync.Mutex
	items map[string]Ticket
	ttl   time.Duration
	now   func() time.Time
}

func NewTickets(ttl time.Duration) *Tickets {
	return &Tickets{items: map[string]Ticket{}, ttl: ttl, now: func() time.Time { return time.Now().UTC() }}
}

func (t *Tickets) Issue(sessionToken, roomID string) (string, time.Time, error) {
	value, err := domain.NewToken()
	if err != nil {
		return "", time.Time{}, err
	}
	now := t.now()
	expires := now.Add(t.ttl)
	t.mu.Lock()
	for key, ticket := range t.items {
		if !ticket.ExpiresAt.After(now) {
			delete(t.items, key)
		}
	}
	t.items[value] = Ticket{SessionToken: sessionToken, RoomID: roomID, ExpiresAt: expires}
	t.mu.Unlock()
	return value, expires, nil
}

func (t *Tickets) Consume(value, roomID string) (Ticket, error) {
	t.mu.Lock()
	ticket, ok := t.items[value]
	delete(t.items, value)
	t.mu.Unlock()
	if !ok || !ticket.ExpiresAt.After(t.now()) || ticket.RoomID != roomID {
		return Ticket{}, domain.NewError(401, "ws_ticket_invalid", "WebSocket ticket is invalid or expired")
	}
	return ticket, nil
}
