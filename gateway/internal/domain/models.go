package domain

import "time"

const (
	RoomOpen   = "open"
	RoomClosed = "closed"

	PlaybackStopped = "stopped"
	PlaybackPlaying = "playing"
	PlaybackPaused  = "paused"

	QueueFIFO       = "fifo"
	QueueFairRandom = "fair_random"
)

type User struct {
	Username       string `json:"username"`
	DisplayName    string `json:"displayName"`
	Admin          bool   `json:"adminRole"`
	MusicFolderIDs []int  `json:"musicFolderIDs"`
}

type AuthProof struct {
	Username string
	Salt     string
	Token    string
}

type Session struct {
	ID         string
	User       User
	Proof      AuthProof
	Generation int64
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

type AuthExchange struct {
	SessionToken       string    `json:"sessionToken"`
	ExpiresAt          time.Time `json:"expiresAt"`
	User               User      `json:"user"`
	NavidromeBaseURL   string    `json:"navidromeBaseURL"`
	GatewayBaseURL     string    `json:"gatewayBaseURL"`
	PluginLeaseExpires time.Time `json:"pluginLeaseExpiresAt"`
}

type RoomCapabilities struct {
	Chat          bool `json:"chat"`
	Stickers      bool `json:"stickers"`
	Statistics    bool `json:"statistics"`
	Rankings      bool `json:"rankings"`
	Achievements  bool `json:"achievements"`
	Uploads       bool `json:"uploads"`
	OnlineSources bool `json:"onlineSources"`
	Invites       bool `json:"invites"`
	Queue         bool `json:"queue"`
	History       bool `json:"history"`
}

func FreeCapabilities() RoomCapabilities {
	return RoomCapabilities{Invites: true, Queue: true, History: true}
}

type Room struct {
	RoomID           string           `json:"roomID"`
	Name             string           `json:"name"`
	OwnerUsername    string           `json:"ownerUsername"`
	OwnerDisplayName string           `json:"ownerDisplayName"`
	Status           string           `json:"status"`
	QueueLimit       int              `json:"queueLimit"`
	PlaybackMode     string           `json:"playbackMode"`
	MusicFolderIDs   []int            `json:"musicFolderIDs"`
	PreloadNextTrack bool             `json:"preloadNextTrack"`
	Capabilities     RoomCapabilities `json:"capabilities"`
	OnlineCount      int              `json:"onlineCount"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

type Member struct {
	RoomID      string    `json:"roomID,omitempty"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	Active      bool      `json:"active"`
	Online      bool      `json:"online"`
	JoinedAt    time.Time `json:"joinedAt"`
	LastSeenAt  time.Time `json:"lastSeenAt"`
}

type Invite struct {
	InviteID  string     `json:"inviteID"`
	RoomID    string     `json:"roomID"`
	Label     string     `json:"label"`
	ExpiresAt time.Time  `json:"expiresAt"`
	MaxUses   int        `json:"maxUses"`
	UseCount  int        `json:"useCount"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	CreatedBy string     `json:"createdBy"`
	CreatedAt time.Time  `json:"createdAt"`
	Invite    string     `json:"invite,omitempty"`
	ShareURL  string     `json:"shareURL,omitempty"`
	DeepLink  string     `json:"deepLink,omitempty"`
}

type NavidromeTrackRef struct {
	ID              string  `json:"id"`
	MusicFolderID   int     `json:"musicFolderID"`
	AlbumID         string  `json:"albumID,omitempty"`
	Title           string  `json:"title"`
	Artist          string  `json:"artist"`
	Album           string  `json:"album"`
	DurationSeconds float64 `json:"durationSeconds"`
	CoverArtID      string  `json:"coverArtID,omitempty"`
}

type QueueEntry struct {
	QueueID         string            `json:"queueID"`
	RoomID          string            `json:"roomID,omitempty"`
	Position        int               `json:"position"`
	Track           NavidromeTrackRef `json:"track"`
	Contributor     string            `json:"contributorUsername"`
	ContributorName string            `json:"contributorDisplayName"`
	CreatedAt       time.Time         `json:"createdAt"`
}

type PlaybackState struct {
	Revision         int64              `json:"revision"`
	ServerTime       time.Time          `json:"serverTime"`
	Status           string             `json:"status"`
	PausedForEmpty   bool               `json:"pausedForEmpty"`
	PositionSeconds  float64            `json:"positionSeconds"`
	AnchorServerTime *time.Time         `json:"anchorServerTime,omitempty"`
	CurrentTrack     *NavidromeTrackRef `json:"currentTrack,omitempty"`
	NextTrack        *NavidromeTrackRef `json:"nextTrack,omitempty"`
	Contributor      string             `json:"contributorUsername,omitempty"`
	ContributorName  string             `json:"contributorDisplayName,omitempty"`
}

type HistoryEntry struct {
	HistoryID       string            `json:"historyID"`
	RoomID          string            `json:"roomID,omitempty"`
	Track           NavidromeTrackRef `json:"track"`
	Contributor     string            `json:"contributorUsername"`
	ContributorName string            `json:"contributorDisplayName"`
	StartedAt       time.Time         `json:"startedAt"`
	FinishedAt      *time.Time        `json:"finishedAt,omitempty"`
	PlayedSeconds   float64           `json:"playedSeconds"`
}

type Snapshot struct {
	Room        Room           `json:"room"`
	Self        Member         `json:"self"`
	Members     []Member       `json:"members"`
	Queue       []QueueEntry   `json:"queue"`
	Playback    PlaybackState  `json:"playback"`
	History     []HistoryEntry `json:"history"`
	GeneratedAt time.Time      `json:"generatedAt"`
}

type PluginUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Admin       bool   `json:"admin"`
}

type PluginSync struct {
	PluginVersion        string       `json:"pluginVersion"`
	Generation           int64        `json:"generation"`
	NavidromeInternalURL string       `json:"navidromeInternalURL"`
	NavidromePublicURL   string       `json:"navidromePublicURL"`
	GatewayPublicURL     string       `json:"gatewayPublicURL"`
	LicenseFile          string       `json:"licenseFile,omitempty"`
	UpdateChannel        string       `json:"updateChannel,omitempty"`
	Users                []PluginUser `json:"users"`
	SentAt               time.Time    `json:"sentAt"`
}

type PluginState struct {
	PluginVersion      string       `json:"pluginVersion"`
	Generation         int64        `json:"generation"`
	NavidromePublicURL string       `json:"navidromePublicURL"`
	GatewayPublicURL   string       `json:"gatewayPublicURL"`
	LicenseFile        string       `json:"licenseFile,omitempty"`
	UpdateChannel      string       `json:"updateChannel,omitempty"`
	Users              []PluginUser `json:"users"`
	LastHeartbeat      time.Time    `json:"lastHeartbeat"`
}

func (p PluginState) User(username string) (PluginUser, bool) {
	for _, user := range p.Users {
		if EqualUsername(user.Username, username) {
			return user, true
		}
	}
	return PluginUser{}, false
}

type Event struct {
	Type       string    `json:"type"`
	RoomID     string    `json:"roomID"`
	Revision   int64     `json:"revision,omitempty"`
	ServerTime time.Time `json:"serverTime"`
	Payload    any       `json:"payload"`
}
