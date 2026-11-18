package api

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type chatRequest struct {
	MessageType string                 `json:"messageType,omitempty"`
	Body        string                 `json:"body,omitempty"`
	StickerID   string                 `json:"stickerID,omitempty"`
	ReplyToID   string                 `json:"replyToID,omitempty"`
	Mentions    []string               `json:"mentions,omitempty"`
	References  []domain.ChatReference `json:"references,omitempty"`
}

var stickerPacks = builtInStickers()

func builtInStickers() []domain.StickerPack {
	languages := []string{"en", "zh-CN", "zh-TW", "ja", "es", "fr", "de"}
	definitions := []struct {
		pack, code, emoji string
		labels            []string
	}{
		{"vibes", "music", "🎵", []string{"Music", "音乐", "音樂", "音楽", "Música", "Musique", "Musik"}},
		{"vibes", "headphones", "🎧", []string{"Listening", "聆听", "聆聽", "聴いている", "Escuchando", "À l’écoute", "Zuhören"}},
		{"vibes", "dance", "💃", []string{"Dance", "跳舞", "跳舞", "ダンス", "Bailar", "Danser", "Tanzen"}},
		{"vibes", "fire", "🔥", []string{"Fire", "燃起来", "燃起來", "熱い", "Fuego", "Feu", "Feuer"}},
		{"vibes", "chill", "🌙", []string{"Chill", "放松", "放鬆", "くつろぐ", "Relajarse", "Détente", "Entspannen"}},
		{"vibes", "sparkles", "✨", []string{"Sparkles", "闪耀", "閃耀", "きらきら", "Destellos", "Étincelles", "Funkeln"}},
		{"vibes", "rock", "🤘", []string{"Rock", "摇滚", "搖滾", "ロック", "Rock", "Rock", "Rock"}},
		{"vibes", "piano", "🎹", []string{"Piano", "钢琴", "鋼琴", "ピアノ", "Piano", "Piano", "Klavier"}},
		{"reactions", "love", "❤️", []string{"Love", "喜欢", "喜歡", "大好き", "Amor", "Amour", "Liebe"}},
		{"reactions", "clap", "👏", []string{"Applause", "鼓掌", "鼓掌", "拍手", "Aplausos", "Applaudissements", "Applaus"}},
		{"reactions", "wow", "😮", []string{"Wow", "惊叹", "驚嘆", "びっくり", "Asombro", "Surprise", "Staunen"}},
		{"reactions", "laugh", "😂", []string{"Laugh", "大笑", "大笑", "笑う", "Risa", "Rire", "Lachen"}},
		{"reactions", "cry", "🥲", []string{"Moved", "感动", "感動", "感動", "Emoción", "Émotion", "Gerührt"}},
		{"reactions", "yes", "👍", []string{"Yes", "赞", "讚", "いいね", "Sí", "Oui", "Ja"}},
		{"reactions", "thanks", "🙏", []string{"Thanks", "谢谢", "謝謝", "ありがとう", "Gracias", "Merci", "Danke"}},
		{"reactions", "party", "🎉", []string{"Party", "庆祝", "慶祝", "お祝い", "Fiesta", "Fête", "Feiern"}},
	}
	packs := []domain.StickerPack{{PackID: "vibes", Stickers: []domain.Sticker{}}, {PackID: "reactions", Stickers: []domain.Sticker{}}}
	for _, item := range definitions {
		sticker := domain.Sticker{StickerID: item.pack + ":" + item.code, PackID: item.pack, Emoji: item.emoji, Code: item.code, Labels: map[string]string{}}
		for i, language := range languages {
			sticker.Labels[language] = item.labels[i]
		}
		index := 0
		if item.pack == "reactions" {
			index = 1
		}
		packs[index].Stickers = append(packs[index].Stickers, sticker)
	}
	return packs
}

func validSticker(id string) bool {
	for _, pack := range stickerPacks {
		for _, sticker := range pack.Stickers {
			if sticker.StickerID == id {
				return true
			}
		}
	}
	return false
}

func (s *Server) stickers(w http.ResponseWriter, r *http.Request) {
	if _, _, _, err := s.requireRoomAccess(r, true); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"packs": stickerPacks})
}

func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	_, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	before := r.URL.Query().Get("before")
	if before != "" && domain.ValidateID(before) != nil {
		writeError(w, domain.NewError(400, "message_id_invalid", "Message ID is invalid"))
		return
	}
	page, err := s.store.Chat(r.Context(), room.RoomID, before, boundedQueryInt(r, "limit", 50, 1, 200))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, page)
}

func (s *Server) postChat(w http.ResponseWriter, r *http.Request) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	if room.Status != domain.RoomOpen {
		writeError(w, domain.NewError(409, "room_closed", "Room is closed"))
		return
	}
	var input chatRequest
	if err = decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.Body = strings.TrimSpace(input.Body)
	if input.MessageType == "" {
		input.MessageType = "chat"
	}
	if (input.MessageType != "chat" && input.MessageType != "review") || utf8.RuneCountInString(input.Body) > 4000 || len(input.Mentions) > 20 || len(input.References) > 10 {
		writeError(w, domain.NewError(400, "chat_invalid", "Message type or size is invalid"))
		return
	}
	if input.StickerID != "" && !validSticker(input.StickerID) {
		writeError(w, domain.NewError(400, "sticker_invalid", "Sticker was not found"))
		return
	}
	if input.Body == "" && input.StickerID == "" && len(input.References) == 0 {
		writeError(w, domain.NewError(400, "chat_empty", "A message needs text, a sticker, or a reference"))
		return
	}
	if input.ReplyToID != "" && domain.ValidateID(input.ReplyToID) != nil {
		writeError(w, domain.NewError(400, "message_id_invalid", "Reply message ID is invalid"))
		return
	}
	mentions := []string{}
	seen := map[string]bool{}
	for _, username := range input.Mentions {
		if len(username) > 128 {
			writeError(w, domain.NewError(400, "mention_invalid", "Mention is invalid"))
			return
		}
		member, err := s.store.ActiveMember(r.Context(), room.RoomID, strings.TrimSpace(username))
		if err != nil {
			writeError(w, domain.NewError(400, "mention_invalid", "Mention must identify an active room member"))
			return
		}
		key := strings.ToLower(member.Username)
		if !seen[key] {
			mentions = append(mentions, member.Username)
			seen[key] = true
		}
	}
	for _, ref := range input.References {
		if !validMediaKind(ref.Kind) || strings.TrimSpace(ref.ID) == "" || len(ref.ID) > 256 || strings.TrimSpace(ref.Title) == "" || utf8.RuneCountInString(ref.Title) > 300 || utf8.RuneCountInString(ref.Artist) > 300 {
			writeError(w, domain.NewError(400, "reference_invalid", "Reference must identify a track, album, or artist"))
			return
		}
	}
	if !s.chatLimiter.Allow(strings.ToLower(session.User.Username)) {
		w.Header().Set("Retry-After", "60")
		writeError(w, domain.NewError(429, "chat_rate_limited", "At most twenty messages per minute are allowed"))
		return
	}
	id, err := domain.NewID()
	if err != nil {
		writeError(w, err)
		return
	}
	msg, err := s.store.AddChat(r.Context(), domain.ChatMessage{
		MessageID: id, RoomID: room.RoomID, SenderUsername: session.User.Username, SenderDisplayName: session.User.DisplayName,
		MessageType: input.MessageType, Body: input.Body, StickerID: input.StickerID, ReplyToID: input.ReplyToID, Mentions: mentions, References: input.References,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "chat_message", Payload: msg})
	writeJSON(w, 201, msg)
}

func (s *Server) recallChat(w http.ResponseWriter, r *http.Request) { s.changeChat(w, r, true) }
func (s *Server) deleteChat(w http.ResponseWriter, r *http.Request) { s.changeChat(w, r, false) }
func (s *Server) changeChat(w http.ResponseWriter, r *http.Request, recall bool) {
	session, room, _, err := s.requireRoomAccess(r, true)
	if err != nil {
		writeError(w, err)
		return
	}
	id := r.PathValue("message_id")
	if domain.ValidateID(id) != nil {
		writeError(w, domain.NewError(400, "message_id_invalid", "Message ID is invalid"))
		return
	}
	msg, err := s.store.UpdateChat(r.Context(), room.RoomID, id, session.User.Username, requireManager(session, room) == nil, recall)
	if err != nil {
		writeError(w, err)
		return
	}
	s.hub.Broadcast(room.RoomID, domain.Event{Type: "chat_updated", Payload: msg})
	if recall {
		writeJSON(w, 200, msg)
	} else {
		w.WriteHeader(204)
	}
}

func validMediaKind(kind string) bool { return kind == "track" || kind == "album" || kind == "artist" }
