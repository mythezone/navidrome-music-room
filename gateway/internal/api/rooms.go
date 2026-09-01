package api

import (
	"net/http"
	"slices"
	"strings"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type createRoomRequest struct {
	Name             string `json:"name"`
	QueueLimit       int    `json:"queueLimit"`
	PlaybackMode     string `json:"playbackMode"`
	MusicFolderIDs   []int  `json:"musicFolderIDs"`
	PreloadNextTrack *bool  `json:"preloadNextTrack,omitempty"`
}

type patchRoomRequest struct {
	Name             *string `json:"name,omitempty"`
	QueueLimit       *int    `json:"queueLimit,omitempty"`
	PlaybackMode     *string `json:"playbackMode,omitempty"`
	MusicFolderIDs   *[]int  `json:"musicFolderIDs,omitempty"`
	PreloadNextTrack *bool   `json:"preloadNextTrack,omitempty"`
}

func (s *Server) listRooms(w http.ResponseWriter, r *http.Request) {
	session, _, err := s.session(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rooms, err := s.store.ListRooms(r.Context(), session.User.Username, session.User.Admin)
	if err != nil {
		writeError(w, err)
		return
	}
	for index := range rooms {
		rooms[index].OnlineCount = s.hub.OnlineCount(rooms[index].RoomID)
	}
	writeJSON(w, 200, map[string]any{"rooms": rooms})
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	session, _, err := s.session(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.sessions.RequireFreshLease(r.Context()); err != nil {
		writeError(w, err)
		return
	}
	if !session.User.Admin {
		writeError(w, domain.NewError(403, "navidrome_admin_required", "Only a Navidrome administrator can create a room"))
		return
	}
	var input createRoomRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	name := strings.Join(strings.Fields(input.Name), " ")
	if name == "" || len([]rune(name)) > 80 {
		writeError(w, domain.NewError(400, "room_name_invalid", "Room name must contain 1 to 80 characters"))
		return
	}
	if input.QueueLimit == 0 {
		input.QueueLimit = 20
	}
	if input.QueueLimit < 1 || input.QueueLimit > 100 {
		writeError(w, domain.NewError(400, "queue_limit_invalid", "Queue limit must be between 1 and 100"))
		return
	}
	if input.PlaybackMode == "" {
		input.PlaybackMode = domain.QueueFIFO
	}
	if input.PlaybackMode != domain.QueueFIFO && input.PlaybackMode != domain.QueueFairRandom {
		writeError(w, domain.NewError(400, "playback_mode_invalid", "Playback mode must be fifo or fair_random"))
		return
	}
	folders, err := normalizeFolders(input.MusicFolderIDs, session.User.MusicFolderIDs)
	if err != nil {
		writeError(w, err)
		return
	}
	roomID, err := domain.NewID()
	if err != nil {
		writeError(w, err)
		return
	}
	preload := true
	if input.PreloadNextTrack != nil {
		preload = *input.PreloadNextTrack
	}
	room, err := s.store.CreateRoom(r.Context(), domain.Room{
		RoomID: roomID, Name: name, OwnerUsername: session.User.Username, OwnerDisplayName: session.User.DisplayName,
		QueueLimit: input.QueueLimit, PlaybackMode: input.PlaybackMode, MusicFolderIDs: folders, PreloadNextTrack: preload,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, room)
}

func (s *Server) getRoom(w http.ResponseWriter, r *http.Request) {
	_, room, _, err := s.requireRoomAccess(r, false)
	if err != nil {
		writeError(w, err)
		return
	}
	room.OnlineCount = s.hub.OnlineCount(room.RoomID)
	writeJSON(w, 200, room)
}

func (s *Server) patchRoom(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, false)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	var input patchRoomRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	if input.Name != nil {
		room.Name = strings.Join(strings.Fields(*input.Name), " ")
		if room.Name == "" || len([]rune(room.Name)) > 80 {
			writeError(w, domain.NewError(400, "room_name_invalid", "Room name must contain 1 to 80 characters"))
			return
		}
	}
	if input.QueueLimit != nil {
		if *input.QueueLimit < 1 || *input.QueueLimit > 100 {
			writeError(w, domain.NewError(400, "queue_limit_invalid", "Queue limit must be between 1 and 100"))
			return
		}
		room.QueueLimit = *input.QueueLimit
	}
	if input.PlaybackMode != nil {
		if *input.PlaybackMode != domain.QueueFIFO && *input.PlaybackMode != domain.QueueFairRandom {
			writeError(w, domain.NewError(400, "playback_mode_invalid", "Playback mode must be fifo or fair_random"))
			return
		}
		room.PlaybackMode = *input.PlaybackMode
	}
	if input.MusicFolderIDs != nil {
		folders, err := normalizeFolders(*input.MusicFolderIDs, session.User.MusicFolderIDs)
		if err != nil {
			writeError(w, err)
			return
		}
		room.MusicFolderIDs = folders
	}
	if input.PreloadNextTrack != nil {
		room.PreloadNextTrack = *input.PreloadNextTrack
	}
	room, err = s.store.UpdateRoom(r.Context(), room, session.User.Username)
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "room_updated", Payload: room})
	writeJSON(w, 200, room)
}

func (s *Server) deleteRoom(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, false)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	if err := s.store.DeleteRoom(r.Context(), room.RoomID, session.User.Username); err != nil {
		writeError(w, err)
		return
	}
	s.hub.CloseRoom(room.RoomID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) closeRoom(w http.ResponseWriter, r *http.Request) {
	s.changeRoomStatus(w, r, domain.RoomClosed)
}
func (s *Server) reopenRoom(w http.ResponseWriter, r *http.Request) {
	s.changeRoomStatus(w, r, domain.RoomOpen)
}

func (s *Server) changeRoomStatus(w http.ResponseWriter, r *http.Request, status string) {
	session, room, _, err := s.requireRoomAccess(r, false)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := requireManager(session, room); err != nil {
		writeError(w, err)
		return
	}
	if status == domain.RoomOpen {
		if err := s.sessions.RequireFreshLease(r.Context()); err != nil {
			writeError(w, err)
			return
		}
	}
	room, err = s.store.SetRoomStatus(r.Context(), room.RoomID, status, session.User.Username)
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "room_updated", Payload: room})
	writeJSON(w, 200, room)
}

func normalizeFolders(requested, allowed []int) ([]int, error) {
	if len(requested) == 0 {
		requested = slices.Clone(allowed)
	}
	seen := map[int]struct{}{}
	result := make([]int, 0, len(requested))
	for _, folder := range requested {
		if folder <= 0 || !domain.ContainsFolder(allowed, folder) {
			return nil, domain.ErrorWithDetails(403, "library_access_required", "Selected music folder is not accessible to this Navidrome user", map[string]any{
				"musicFolderID": folder, "userMusicFolderIDs": allowed,
			})
		}
		if _, duplicate := seen[folder]; duplicate {
			continue
		}
		seen[folder] = struct{}{}
		result = append(result, folder)
	}
	if len(result) == 0 {
		return nil, domain.NewError(403, "library_access_required", "At least one accessible Navidrome music folder is required")
	}
	slices.Sort(result)
	return result, nil
}
