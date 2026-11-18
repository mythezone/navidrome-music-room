package api

import (
	"net/http"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

func (s *Server) statistics(w http.ResponseWriter, r *http.Request) {
	_, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "all"
	}
	result, err := s.store.Statistics(r.Context(), room.RoomID, period)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) rankings(w http.ResponseWriter, r *http.Request) {
	_, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "all"
	}
	result, err := s.store.Rankings(r.Context(), room.RoomID, period)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) profile(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	username := r.PathValue("username")
	if username == "" {
		username = session.User.Username
	}
	profile, err := s.store.MemberProfile(r.Context(), room.RoomID, username)
	if err != nil {
		writeError(w, err)
		return
	}
	s.setProfileOnline(room.RoomID, &profile)
	writeJSON(w, 200, profile)
}

func (s *Server) selectBadge(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	var input struct {
		BadgeID string `json:"badgeID"`
	}
	if err = decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	profile, err := s.store.SelectBadge(r.Context(), room.RoomID, session.User.Username, input.BadgeID)
	if err != nil {
		writeError(w, err)
		return
	}
	s.setProfileOnline(room.RoomID, &profile)
	writeJSON(w, 200, profile)
}

func (s *Server) setProfileOnline(roomID string, profile *domain.MemberProfile) {
	for username := range s.hub.OnlineUsers(roomID) {
		if domain.EqualUsername(username, profile.Username) {
			profile.Online = true
		}
	}
}
