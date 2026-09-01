package api

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type createInviteRequest struct {
	Label     string     `json:"label"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	MaxUses   int        `json:"maxUses"`
	SingleUse bool       `json:"singleUse"`
}

type redeemInviteRequest struct {
	RoomID string `json:"roomID"`
	Invite string `json:"invite"`
}

func (s *Server) listInvites(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, false)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	invites, err := s.store.ListInvites(r.Context(), room.RoomID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"invites": invites})
}

func (s *Server) createInvite(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, false)
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
	if err := s.sessions.RequireFreshLease(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	var input createInviteRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	now := time.Now().UTC()
	expires := now.Add(7 * 24 * time.Hour)
	if input.ExpiresAt != nil {
		expires = input.ExpiresAt.UTC()
	}
	if !expires.After(now.Add(time.Minute)) || expires.After(now.Add(365*24*time.Hour)) {
		writeError(w, domain.NewError(400, "invite_expiry_invalid", "Invitation expiry must be between one minute and one year"))
		return
	}
	maxUses := input.MaxUses
	if maxUses == 0 {
		maxUses = 20
	}
	if input.SingleUse {
		maxUses = 1
	}
	if maxUses < 1 || maxUses > 10000 {
		writeError(w, domain.NewError(400, "invite_use_limit_invalid", "Invitation use limit must be between 1 and 10000"))
		return
	}
	inviteID, err := domain.NewID()
	if err != nil {
		writeError(w, err)
		return
	}
	token, err := domain.NewToken()
	if err != nil {
		writeError(w, err)
		return
	}
	invite := domain.Invite{
		InviteID: inviteID, RoomID: room.RoomID, Label: strings.Join(strings.Fields(input.Label), " "),
		ExpiresAt: expires, MaxUses: maxUses, CreatedBy: session.User.Username, CreatedAt: now, Invite: token,
	}
	invite.ShareURL, invite.DeepLink = s.inviteLinks(room.RoomID, token)
	invite, err = s.store.CreateInvite(r.Context(), invite, domain.TokenHash(token))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, invite)
}

func (s *Server) revokeInvite(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, false)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	inviteID := r.PathValue("invite_id")
	if domain.ValidateID(inviteID) != nil {
		writeError(w, domain.NewError(400, "invite_id_invalid", "Invitation ID is invalid"))
		return
	}
	if err := s.store.RevokeInvite(r.Context(), room.RoomID, inviteID, session.User.Username); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) redeemInvite(w http.ResponseWriter, r *http.Request) {
	session, _, err := s.session(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.sessions.RequireFreshLease(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	var input redeemInviteRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	if domain.ValidateID(input.RoomID) != nil || len(input.Invite) < 32 || len(input.Invite) > 128 {
		writeError(w, domain.NewError(400, "invite_invalid", "Invitation payload is invalid"))
		return
	}
	room, err := s.store.GetRoom(r.Context(), input.RoomID)
	if err != nil {
		writeError(w, err)
		return
	}
	if room.Status != domain.RoomOpen {
		writeError(w, domain.NewError(409, "room_closed", "Room is closed"))
		return
	}
	if err := requireLibraryAccess(session.User, room); err != nil {
		writeError(w, err)
		return
	}
	member, err := s.store.RedeemInvite(r.Context(), room.RoomID, domain.TokenHash(input.Invite), session.User)
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "room_updated", Payload: room})
	writeJSON(w, 200, map[string]any{"member": member, "room": room})
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	_, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	members, err := s.membersWithPresence(r, room.RoomID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"members": members})
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, false)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	username := strings.TrimSpace(r.PathValue("username"))
	if username == "" {
		writeError(w, domain.NewError(400, "username_invalid", "Username is invalid"))
		return
	}
	if err := s.store.RemoveMember(r.Context(), room.RoomID, username, session.User.Username); err != nil {
		writeError(w, err)
		return
	}
	s.hub.KickUser(room.RoomID, username)
	members, _ := s.membersWithPresence(r, room.RoomID)
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "presence", Payload: map[string]any{"members": members}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) joinRoom(w http.ResponseWriter, r *http.Request) {
	session, _, err := s.session(r)
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
	if room.Status != domain.RoomOpen {
		writeError(w, domain.NewError(409, "room_closed", "Room is closed"))
		return
	}
	if err := requireLibraryAccess(session.User, room); err != nil {
		writeError(w, err)
		return
	}
	member, err := s.store.TouchMember(r.Context(), roomID, session.User)
	if err != nil && session.User.Admin {
		member, err = s.store.UpsertMember(r.Context(), roomID, session.User, "member", session.User.Username, "member.admin_joined")
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"member": member, "room": room})
}

func (s *Server) leaveRoom(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.store.LeaveRoom(r.Context(), room.RoomID, session.User.Username); err != nil {
		writeError(w, err)
		return
	}
	s.hub.KickUser(room.RoomID, session.User.Username)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) membersWithPresence(r *http.Request, roomID string) ([]domain.Member, error) {
	members, err := s.store.ListMembers(r.Context(), roomID)
	if err != nil {
		return nil, err
	}
	online := s.hub.OnlineUsers(roomID)
	for index := range members {
		for username := range online {
			if domain.EqualUsername(username, members[index].Username) {
				members[index].Online = true
				break
			}
		}
	}
	return members, nil
}

func (s *Server) inviteLinks(roomID, token string) (string, string) {
	share := *s.config.GatewayPublic
	share.Path = strings.TrimRight(share.Path, "/") + "/join/" + roomID
	share.RawQuery = ""
	share.Fragment = "invite=" + token
	deepQuery := url.Values{}
	deepQuery.Set("server", s.config.NavidromePublic.String())
	deepQuery.Set("gateway", s.config.GatewayPublic.String())
	deepQuery.Set("room", roomID)
	deepQuery.Set("invite", token)
	deep := url.URL{Scheme: "musicmate", Host: "join", RawQuery: deepQuery.Encode()}
	return share.String(), deep.String()
}
