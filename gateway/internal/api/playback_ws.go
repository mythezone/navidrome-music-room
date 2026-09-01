package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type playbackRequest struct {
	Command          string   `json:"command"`
	ExpectedRevision int64    `json:"expectedRevision"`
	PositionSeconds  *float64 `json:"positionSeconds,omitempty"`
}

type addTrackRequest struct {
	Track domain.NavidromeTrackRef `json:"track"`
}

type queueOrderRequest struct {
	QueueIDs []string `json:"queueIDs"`
}

func (s *Server) snapshot(w http.ResponseWriter, r *http.Request) {
	session, room, member, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := s.buildSnapshot(r.Context(), session, room, *member)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	_, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	limit := boundedQueryInt(r, "limit", 50, 1, 200)
	offset := boundedQueryInt(r, "offset", 0, 0, 1000000)
	entries, err := s.store.History(r.Context(), room.RoomID, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": entries, "limit": limit, "offset": offset})
}

func (s *Server) playback(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	if room.Status != domain.RoomOpen {
		writeError(w, domain.NewError(409, "room_closed", "Room is closed"))
		return
	}
	var input playbackRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.Command = strings.ToLower(strings.TrimSpace(input.Command))
	state, err := s.store.ApplyPlayback(r.Context(), room.RoomID, input.Command, input.PositionSeconds, input.ExpectedRevision, session.User.Username)
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "playback", Revision: state.Revision, Payload: state})
	if input.Command == "next" || input.Command == "play" || input.Command == "stop" {
		if queue, err := s.store.ListQueue(r.Context(), room.RoomID); err == nil {
			s.hub.Broadcast(room.RoomID, domain.Event{Type: "queue", Payload: queue})
		}
		if history, err := s.store.History(r.Context(), room.RoomID, 50, 0); err == nil {
			s.hub.Broadcast(room.RoomID, domain.Event{Type: "history", Payload: history})
		}
	}
	writeJSON(w, 200, state)
}

func (s *Server) addQueueTrack(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	if room.Status != domain.RoomOpen {
		writeError(w, domain.NewError(409, "room_closed", "Room is closed"))
		return
	}
	var input addTrackRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	if input.Track.MusicFolderID <= 0 || !domain.ContainsFolder(room.MusicFolderIDs, input.Track.MusicFolderID) ||
		!domain.ContainsFolder(session.User.MusicFolderIDs, input.Track.MusicFolderID) {
		writeError(w, domain.NewError(403, "library_access_required", "Track is outside the room or user music folders"))
		return
	}
	track, err := s.sessions.ValidateTrack(r.Context(), session, input.Track)
	if err != nil {
		writeError(w, err)
		return
	}
	queueID, err := domain.NewID()
	if err != nil {
		writeError(w, err)
		return
	}
	entry, err := s.store.AddQueueEntry(r.Context(), room, domain.QueueEntry{
		QueueID: queueID, RoomID: room.RoomID, Track: track,
		Contributor: session.User.Username, ContributorName: session.User.DisplayName,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	queue, err := s.store.ListQueue(r.Context(), room.RoomID)
	if err == nil {
		s.hub.Broadcast(room.RoomID, domain.Event{Type: "queue", Payload: queue})
	}
	writeJSON(w, http.StatusCreated, map[string]any{"entry": entry, "queue": queue})
}

func (s *Server) removeQueueEntry(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	queueID := r.PathValue("queue_id")
	if domain.ValidateID(queueID) != nil {
		writeError(w, domain.NewError(400, "queue_id_invalid", "Queue ID is invalid"))
		return
	}
	entry, err := s.store.GetQueueEntry(r.Context(), room.RoomID, queueID)
	if err != nil {
		writeError(w, err)
		return
	}
	if !domain.EqualUsername(entry.Contributor, session.User.Username) {
		if err := requireManager(session, room); err != nil {
			writeError(w, domain.NewError(403, "queue_entry_owner_required", "Members can only remove their own pending queue entries"))
			return
		}
	}
	if err := s.store.RemoveQueueEntry(r.Context(), room.RoomID, queueID, session.User.Username); err != nil {
		writeError(w, err)
		return
	}
	queue, _ := s.store.ListQueue(r.Context(), room.RoomID)
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "queue", Payload: queue})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reorderQueue(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	var input queueOrderRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	if len(input.QueueIDs) > 1000 {
		writeError(w, domain.NewError(400, "queue_order_invalid", "Queue order is too large"))
		return
	}
	queue, err := s.store.ReorderQueue(r.Context(), room.RoomID, input.QueueIDs, session.User.Username)
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "queue", Payload: queue})
	writeJSON(w, 200, map[string]any{"queue": queue})
}

func (s *Server) issueWebSocketTicket(w http.ResponseWriter, r *http.Request) {
	session, token, err := s.session(r)
	if err != nil {
		writeError(w, err)
		return
	}
	roomID := r.PathValue("room_id")
	room, err := s.store.GetRoom(r.Context(), roomID)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := s.store.ActiveMember(r.Context(), roomID, session.User.Username); err != nil {
		writeError(w, err)
		return
	}
	if err := requireLibraryAccess(session.User, room); err != nil {
		writeError(w, err)
		return
	}
	ticket, expiresAt, err := s.tickets.Issue(token, roomID)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{
		"ticket": ticket, "expiresAt": expiresAt, "webSocketURL": websocketURL(s.config.GatewayPublic, roomID, ticket),
	})
}

func (s *Server) webSocket(w http.ResponseWriter, r *http.Request) {
	roomID := r.PathValue("room_id")
	if domain.ValidateID(roomID) != nil {
		writeError(w, domain.NewError(400, "room_id_invalid", "Room ID is invalid"))
		return
	}
	ticket, err := s.tickets.Consume(r.URL.Query().Get("ticket"), roomID)
	if err != nil {
		writeError(w, err)
		return
	}
	session, err := s.sessions.Authenticate(r.Context(), ticket.SessionToken)
	if err != nil {
		writeError(w, err)
		return
	}
	room, err := s.store.GetRoom(r.Context(), roomID)
	if err != nil {
		writeError(w, err)
		return
	}
	member, err := s.store.ActiveMember(r.Context(), roomID, session.User.Username)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := requireLibraryAccess(session.User, room); err != nil {
		writeError(w, err)
		return
	}
	initial, err := s.buildSnapshot(r.Context(), session, room, member)
	if err != nil {
		writeError(w, err)
		return
	}
	connection, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.hub.Serve(r.Context(), roomID, session.User.Username, connection, domain.Event{
		Type: "snapshot", RoomID: roomID, ServerTime: time.Now().UTC(), Payload: initial,
	})
}

func (s *Server) buildSnapshot(ctx context.Context, session domain.Session, room domain.Room, self domain.Member) (domain.Snapshot, error) {
	members, err := s.store.ListMembers(ctx, room.RoomID)
	if err != nil {
		return domain.Snapshot{}, err
	}
	online := s.hub.OnlineUsers(room.RoomID)
	for index := range members {
		for username := range online {
			if domain.EqualUsername(members[index].Username, username) {
				members[index].Online = true
			}
		}
	}
	queue, err := s.store.ListQueue(ctx, room.RoomID)
	if err != nil {
		return domain.Snapshot{}, err
	}
	playback, err := s.store.Playback(ctx, room.RoomID)
	if err != nil {
		return domain.Snapshot{}, err
	}
	history, err := s.store.History(ctx, room.RoomID, 50, 0)
	if err != nil {
		return domain.Snapshot{}, err
	}
	room.OnlineCount = len(online)
	for username := range online {
		if domain.EqualUsername(username, self.Username) {
			self.Online = true
		}
	}
	_ = session
	return domain.Snapshot{
		Room: room, Self: self, Members: members, Queue: queue, Playback: playback,
		History: history, GeneratedAt: time.Now().UTC(),
	}, nil
}

func boundedQueryInt(r *http.Request, name string, fallback, minimum, maximum int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil {
		return fallback
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
