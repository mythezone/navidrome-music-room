package api

import (
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

func (s *Server) wishes(w http.ResponseWriter, r *http.Request) {
	_, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "all"
	}
	if status != "all" && status != "pending" && status != "fulfilled" {
		writeError(w, domain.NewError(400, "wish_status_invalid", "Status must be all, pending, or fulfilled"))
		return
	}
	wishes, err := s.store.Wishes(r.Context(), room.RoomID, status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"wishes": wishes})
}

func (s *Server) postWish(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	if room.Status != domain.RoomOpen {
		writeError(w, domain.NewError(409, "room_closed", "Room is closed"))
		return
	}
	var input struct {
		Kind   string `json:"kind"`
		Title  string `json:"title"`
		Artist string `json:"artist,omitempty"`
		Reason string `json:"reason,omitempty"`
	}
	if err = decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.Title, input.Artist, input.Reason = strings.TrimSpace(input.Title), strings.TrimSpace(input.Artist), strings.TrimSpace(input.Reason)
	if !validMediaKind(input.Kind) || input.Title == "" || utf8.RuneCountInString(input.Title) > 300 || utf8.RuneCountInString(input.Artist) > 300 || utf8.RuneCountInString(input.Reason) > 2000 {
		writeError(w, domain.NewError(400, "wish_invalid", "Wish must name a track, album, or artist within the text limits"))
		return
	}
	id, err := domain.NewID()
	if err != nil {
		writeError(w, err)
		return
	}
	wish, err := s.store.AddWish(r.Context(), domain.Wish{WishID: id, RoomID: room.RoomID, AuthorUsername: session.User.Username, AuthorDisplayName: session.User.DisplayName, Kind: input.Kind, Title: input.Title, Artist: input.Artist, Reason: input.Reason})
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "wish_updated", Payload: wish})
	writeJSON(w, 201, wish)
}

func (s *Server) fulfillWish(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	if err = requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	id := r.PathValue("wish_id")
	if domain.ValidateID(id) != nil {
		writeError(w, domain.NewError(400, "wish_id_invalid", "Wish ID is invalid"))
		return
	}
	var input struct {
		ResponseNote string                `json:"responseNote,omitempty"`
		Resources    []domain.WishResource `json:"resources,omitempty"`
	}
	if err = decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.ResponseNote = strings.TrimSpace(input.ResponseNote)
	if utf8.RuneCountInString(input.ResponseNote) > 2000 || len(input.Resources) > 10 {
		writeError(w, domain.NewError(400, "wish_response_invalid", "Wish response exceeds text or resource limits"))
		return
	}
	for _, resource := range input.Resources {
		parsed, err := url.Parse(resource.URL)
		if err != nil || parsed.Hostname() == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || len(resource.URL) > 2048 || strings.TrimSpace(resource.Title) == "" || utf8.RuneCountInString(resource.Title) > 300 {
			writeError(w, domain.NewError(400, "wish_resource_invalid", "Resources require a title and an absolute HTTP or HTTPS URL"))
			return
		}
	}
	wish, err := s.store.FulfillWish(r.Context(), room.RoomID, id, session.User, input.ResponseNote, input.Resources)
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "wish_updated", Payload: wish})
	writeJSON(w, 200, wish)
}

func (s *Server) deleteWish(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	id := r.PathValue("wish_id")
	if domain.ValidateID(id) != nil {
		writeError(w, domain.NewError(400, "wish_id_invalid", "Wish ID is invalid"))
		return
	}
	if err = s.store.DeleteWish(r.Context(), room.RoomID, id, session.User.Username, requireManager(session, room) == nil); err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "wish_deleted", Payload: map[string]string{"wishID": id}})
	w.WriteHeader(204)
}
