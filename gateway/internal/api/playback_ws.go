package api

import (
	"context"
	"math/rand/v2"
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
	PlayNext bool                     `json:"playNext,omitempty"`
	Track    domain.NavidromeTrackRef `json:"track"`
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

	if input.Command == "next" {
		current, err := s.store.Playback(r.Context(), room.RoomID)
		if err != nil {
			writeError(w, err)
			return
		}
		if requireManager(session, room) != nil && !room.AllowMemberSkip && !(current.CurrentTrack != nil && domain.EqualUsername(current.Contributor, session.User.Username)) {
			writeError(w, domain.NewError(403, "skip_permission_required", "Skipping requires permission or contribution of the current track"))
			return
		}
		// Bind contributor permission to the same revision the transaction will change.
		if current.Revision != input.ExpectedRevision {
			writeError(w, domain.ErrorWithDetails(409, "revision_conflict", "Playback state has changed", map[string]any{"latest": current}))
			return
		}
	} else if err := requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	state, err := s.store.ApplyPlayback(r.Context(), room.RoomID, input.Command, input.PositionSeconds, input.ExpectedRevision, session.User.Username, s.hub.OnlineUsers(room.RoomID))
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "playback", Revision: state.Revision, Payload: state})
	if input.Command == "next" || input.Command == "previous" || input.Command == "play" || input.Command == "stop" {
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

	if input.PlayNext {
		if err := requireManager(session, room); err != nil {
			writeError(w, err)
			return
		}
	}
	s.enqueueTrack(w, r, session, room, input.Track, input.PlayNext)
}

func (s *Server) enqueueTrack(w http.ResponseWriter, r *http.Request, session domain.Session, room domain.Room, ref domain.NavidromeTrackRef, playNext bool) {
	if ref.MusicFolderID <= 0 || !domain.ContainsFolder(room.MusicFolderIDs, ref.MusicFolderID) ||
		!domain.ContainsFolder(session.User.MusicFolderIDs, ref.MusicFolderID) {
		writeError(w, domain.NewError(403, "library_access_required", "Track is outside the room or user music folders"))
		return
	}
	track, err := s.sessions.ValidateTrack(r.Context(), session, ref)
	if err != nil {
		writeError(w, err)
		return
	}
	queueID, err := domain.NewID()
	if err != nil {
		writeError(w, err)
		return
	}
	position := 0
	if playNext {
		position = 1
	}
	entry, primedPlayback, err := s.store.AddQueueEntry(r.Context(), room, domain.QueueEntry{
		QueueID: queueID, RoomID: room.RoomID, Track: track, Position: position,
		Contributor: session.User.Username, ContributorName: session.User.DisplayName,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if primedPlayback != nil {
		s.hub.Broadcast(room.RoomID, domain.Event{
			Type: "playback", Revision: primedPlayback.Revision, Payload: *primedPlayback,
		})
	}
	queue, err := s.store.ListQueue(r.Context(), room.RoomID)
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "queue", Payload: queue})
	response := map[string]any{"entry": entry, "queue": queue}
	if primedPlayback != nil {
		response["playback"] = *primedPlayback
	}
	writeJSON(w, http.StatusCreated, response)
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

	canManage := session.User.Admin || domain.EqualUsername(session.User.Username, room.OwnerUsername)
	canSkip := canManage || room.AllowMemberSkip || (playback.CurrentTrack != nil && domain.EqualUsername(playback.Contributor, session.User.Username))
	var notice *domain.UpdateNotice
	if session.User.Admin {
		channel := "stable"
		if state, err := s.store.PluginState(ctx); err == nil && state.UpdateChannel == "beta" {
			channel = "beta"
		}
		status := s.updater.Status(ctx, channel)
		if status.UpdateAvailable {
			value := updateNotice(status)
			notice = &value
		}
	}
	return domain.Snapshot{
		Permissions:  domain.RoomPermissions{CanManage: canManage, CanControl: canManage, CanSkip: canSkip, CanReorderQueue: canManage, CanFulfillWish: canManage},
		UpdateNotice: notice,
		Room:         room, Self: self, Members: members, Queue: queue, Playback: playback,
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

func (s *Server) shuffleQueue(w http.ResponseWriter, r *http.Request) { s.changeQueueOrder(w, r, true) }
func (s *Server) topQueue(w http.ResponseWriter, r *http.Request)     { s.changeQueueOrder(w, r, false) }
func (s *Server) changeQueueOrder(w http.ResponseWriter, r *http.Request, shuffle bool) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	if err = requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	queue, err := s.store.ListQueue(r.Context(), room.RoomID)
	if err != nil {
		writeError(w, err)
		return
	}
	if shuffle {
		if len(queue) >= 3 {
			rand.Shuffle(len(queue)-1, func(i, j int) { queue[i+1], queue[j+1] = queue[j+1], queue[i+1] })
		}
	} else {
		id := r.PathValue("queue_id")
		if domain.ValidateID(id) != nil {
			writeError(w, domain.NewError(400, "queue_id_invalid", "Queue ID is invalid"))
			return
		}
		index := -1
		for i, entry := range queue {
			if entry.QueueID == id {
				index = i
				break
			}
		}
		if index < 0 {
			writeError(w, domain.NewError(404, "queue_entry_not_found", "Queue entry was not found"))
			return
		}
		entry := queue[index]
		copy(queue[1:index+1], queue[:index])
		queue[0] = entry
	}
	ids := make([]string, 0, len(queue))
	for _, entry := range queue {
		ids = append(ids, entry.QueueID)
	}
	queue, err = s.store.ReorderQueue(r.Context(), room.RoomID, ids, session.User.Username)
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "queue", Payload: queue})
	writeJSON(w, 200, map[string]any{"queue": queue})
}

func (s *Server) requeueHistory(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	if room.Status != domain.RoomOpen {
		writeError(w, domain.NewError(409, "room_closed", "Room is closed"))
		return
	}
	id := r.PathValue("history_id")
	if domain.ValidateID(id) != nil {
		writeError(w, domain.NewError(400, "history_id_invalid", "History ID is invalid"))
		return
	}
	entry, err := s.store.GetHistoryEntry(r.Context(), room.RoomID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	s.enqueueTrack(w, r, session, room, entry.Track, false)
}
