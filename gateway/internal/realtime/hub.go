package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type emptyRoomCallback func(context.Context, string)

type client struct {
	connection *websocket.Conn
	username   string
	writeMu    sync.Mutex
}

type Hub struct {
	mu          sync.RWMutex
	rooms       map[string]map[*client]struct{}
	emptyTimers map[string]*time.Timer
	emptyDelay  time.Duration
	onEmpty     emptyRoomCallback
	logger      *slog.Logger
}

func NewHub(delay time.Duration, onEmpty emptyRoomCallback, logger *slog.Logger) *Hub {
	return &Hub{
		rooms: map[string]map[*client]struct{}{}, emptyTimers: map[string]*time.Timer{},
		emptyDelay: delay, onEmpty: onEmpty, logger: logger,
	}
}

func (h *Hub) Serve(ctx context.Context, roomID, username string, connection *websocket.Conn, initial any) {
	item := &client{connection: connection, username: username}
	h.add(roomID, item)
	defer h.remove(roomID, item)
	_ = connection.SetReadDeadline(time.Now().Add(75 * time.Second))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(75 * time.Second))
	})
	if initial != nil {
		if err := item.writeJSON(initial); err != nil {
			return
		}
	}
	h.broadcastPresence(roomID)
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				item.writeMu.Lock()
				err := connection.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
				item.writeMu.Unlock()
				if err != nil {
					_ = connection.Close()
					return
				}
			}
		}
	}()
	defer close(stop)
	for {
		messageType, body, err := connection.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage || len(body) > 4096 {
			continue
		}
		var message struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(body, &message) == nil && message.Type == "ping" {
			_ = item.writeJSON(domain.Event{Type: "pong", RoomID: roomID, ServerTime: time.Now().UTC(), Payload: map[string]any{}})
		}
	}
}

func (h *Hub) Broadcast(roomID string, event domain.Event) {
	event.RoomID = roomID
	event.ServerTime = time.Now().UTC()
	h.mu.RLock()
	clients := make([]*client, 0, len(h.rooms[roomID]))
	for item := range h.rooms[roomID] {
		clients = append(clients, item)
	}
	h.mu.RUnlock()
	for _, item := range clients {
		if err := item.writeJSON(event); err != nil {
			_ = item.connection.Close()
		}
	}
}

func (h *Hub) OnlineUsers(roomID string) map[string]bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := map[string]bool{}
	for item := range h.rooms[roomID] {
		result[item.username] = true
	}
	return result
}

func (h *Hub) OnlineCount(roomID string) int {
	return len(h.OnlineUsers(roomID))
}

func (h *Hub) CloseRoom(roomID string) {
	h.mu.Lock()
	clients := h.rooms[roomID]
	delete(h.rooms, roomID)
	if timer := h.emptyTimers[roomID]; timer != nil {
		timer.Stop()
		delete(h.emptyTimers, roomID)
	}
	h.mu.Unlock()
	for item := range clients {
		item.writeMu.Lock()
		_ = item.connection.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "room closed"), time.Now().Add(2*time.Second))
		_ = item.connection.Close()
		item.writeMu.Unlock()
	}
}

func (h *Hub) KickUser(roomID, username string) {
	h.mu.RLock()
	var targets []*client
	for item := range h.rooms[roomID] {
		if domain.EqualUsername(item.username, username) {
			targets = append(targets, item)
		}
	}
	h.mu.RUnlock()
	for _, item := range targets {
		item.writeMu.Lock()
		_ = item.connection.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "membership removed"), time.Now().Add(2*time.Second))
		_ = item.connection.Close()
		item.writeMu.Unlock()
	}
}

func (h *Hub) KickUserEverywhere(username string) {
	h.mu.RLock()
	rooms := make([]string, 0, len(h.rooms))
	for roomID := range h.rooms {
		rooms = append(rooms, roomID)
	}
	h.mu.RUnlock()
	for _, roomID := range rooms {
		h.KickUser(roomID, username)
	}
}

func (h *Hub) add(roomID string, item *client) {
	h.mu.Lock()
	if timer := h.emptyTimers[roomID]; timer != nil {
		timer.Stop()
		delete(h.emptyTimers, roomID)
	}
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = map[*client]struct{}{}
	}
	h.rooms[roomID][item] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) remove(roomID string, item *client) {
	h.mu.Lock()
	if clients := h.rooms[roomID]; clients != nil {
		delete(clients, item)
		if len(clients) == 0 {
			delete(h.rooms, roomID)
			var timer *time.Timer
			timer = time.AfterFunc(h.emptyDelay, func() {
				h.mu.Lock()
				if h.emptyTimers[roomID] != timer || len(h.rooms[roomID]) != 0 {
					h.mu.Unlock()
					return
				}
				delete(h.emptyTimers, roomID)
				h.mu.Unlock()
				if h.onEmpty != nil {
					h.onEmpty(context.Background(), roomID)
				}
			})
			h.emptyTimers[roomID] = timer
		}
	}
	h.mu.Unlock()
	h.broadcastPresence(roomID)
}

func (h *Hub) broadcastPresence(roomID string) {
	users := h.OnlineUsers(roomID)
	usernames := make([]string, 0, len(users))
	for username := range users {
		usernames = append(usernames, username)
	}
	slices.Sort(usernames)
	h.Broadcast(roomID, domain.Event{
		Type: "presence", Payload: map[string]any{"onlineUsernames": usernames, "onlineCount": len(usernames)},
	})
}

func (c *client) writeJSON(value any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.connection.SetWriteDeadline(time.Now().Add(8 * time.Second))
	return c.connection.WriteJSON(value)
}
