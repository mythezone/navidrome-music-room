package domain

import "time"

type ChatReference struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist,omitempty"`
}

type ChatReplyPreview struct {
	MessageID         string     `json:"messageID"`
	SenderUsername    string     `json:"senderUsername"`
	SenderDisplayName string     `json:"senderDisplayName"`
	Body              string     `json:"body"`
	StickerID         string     `json:"stickerID"`
	RecalledAt        *time.Time `json:"recalledAt,omitempty"`
	DeletedAt         *time.Time `json:"deletedAt,omitempty"`
}

type ChatMessage struct {
	MessageID         string            `json:"messageID"`
	RoomID            string            `json:"roomID"`
	SenderUsername    string            `json:"senderUsername"`
	SenderDisplayName string            `json:"senderDisplayName"`
	MessageType       string            `json:"messageType"`
	Body              string            `json:"body"`
	StickerID         string            `json:"stickerID"`
	ReplyToID         string            `json:"replyToID"`
	Mentions          []string          `json:"mentions"`
	References        []ChatReference   `json:"references"`
	ReplyPreview      *ChatReplyPreview `json:"replyPreview,omitempty"`
	RecalledAt        *time.Time        `json:"recalledAt,omitempty"`
	DeletedAt         *time.Time        `json:"deletedAt,omitempty"`
	CreatedAt         time.Time         `json:"createdAt"`
}

type ChatPage struct {
	Messages   []ChatMessage `json:"messages"`
	HasMore    bool          `json:"hasMore"`
	NextBefore string        `json:"nextBefore"`
}

type Sticker struct {
	StickerID string            `json:"stickerID"`
	PackID    string            `json:"packID"`
	Emoji     string            `json:"emoji"`
	Code      string            `json:"code"`
	Labels    map[string]string `json:"labels"`
}

type StickerPack struct {
	PackID   string    `json:"packID"`
	Stickers []Sticker `json:"stickers"`
}

type WishResource struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type Wish struct {
	WishID            string         `json:"wishID"`
	RoomID            string         `json:"roomID"`
	AuthorUsername    string         `json:"authorUsername"`
	AuthorDisplayName string         `json:"authorDisplayName"`
	Kind              string         `json:"kind"`
	Title             string         `json:"title"`
	Artist            string         `json:"artist"`
	Reason            string         `json:"reason"`
	Status            string         `json:"status"`
	FulfilledBy       string         `json:"fulfilledBy"`
	FulfilledByName   string         `json:"fulfilledByName"`
	ResponseNote      string         `json:"responseNote"`
	Resources         []WishResource `json:"resources"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

type MediaStat struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Artist  string  `json:"artist"`
	Count   int     `json:"count"`
	Seconds float64 `json:"seconds"`
}

type HourStat struct {
	Hour    int     `json:"hour"`
	Count   int     `json:"count"`
	Seconds float64 `json:"seconds"`
}

type RoomStatistics struct {
	Period             string      `json:"period"`
	TotalPlayCount     int         `json:"totalPlayCount"`
	TotalPlayedSeconds float64     `json:"totalPlayedSeconds"`
	TotalChatCount     int         `json:"totalChatCount"`
	ActiveMemberCount  int         `json:"activeMemberCount"`
	TopArtists         []MediaStat `json:"topArtists"`
	TopAlbums          []MediaStat `json:"topAlbums"`
	TopTracks          []MediaStat `json:"topTracks"`
	HourlyDistribution []HourStat  `json:"hourlyDistribution"`
}

type MemberRank struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Count       int    `json:"count"`
}

type RoomRankings struct {
	Period     string       `json:"period"`
	Requested  []MemberRank `json:"requested"`
	Listened   []MemberRank `json:"listened"`
	Chat       []MemberRank `json:"chat"`
	TopArtists []MediaStat  `json:"topArtists"`
	TopAlbums  []MediaStat  `json:"topAlbums"`
	TopTracks  []MediaStat  `json:"topTracks"`
}

type MemberStats struct {
	StaySeconds    float64 `json:"staySeconds"`
	RequestedCount int     `json:"requestedCount"`
	ListenedCount  int     `json:"listenedCount"`
	ChatCount      int     `json:"chatCount"`
}

type MemberLevel struct {
	Level                   int     `json:"level"`
	Code                    string  `json:"code"`
	EffectiveMinutes        float64 `json:"effectiveMinutes"`
	CurrentThresholdMinutes float64 `json:"currentThresholdMinutes"`
	NextThresholdMinutes    float64 `json:"nextThresholdMinutes"`
	Progress                float64 `json:"progress"`
}

type Badge struct {
	BadgeID    string     `json:"badgeID"`
	Code       string     `json:"code"`
	Kind       string     `json:"kind"`
	Subject    string     `json:"subject"`
	Tier       int        `json:"tier"`
	Count      int        `json:"count"`
	Target     int        `json:"target"`
	Unlocked   bool       `json:"unlocked"`
	UnlockedAt *time.Time `json:"unlockedAt"`
}

type MemberProfile struct {
	Username        string      `json:"username"`
	DisplayName     string      `json:"displayName"`
	Role            string      `json:"role"`
	Online          bool        `json:"online"`
	JoinedAt        time.Time   `json:"joinedAt"`
	LastSeenAt      time.Time   `json:"lastSeenAt"`
	Stats           MemberStats `json:"stats"`
	Level           MemberLevel `json:"level"`
	SelectedBadgeID string      `json:"selectedBadgeID"`
	Badges          []Badge     `json:"badges"`
}
