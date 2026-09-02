package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
)

const (
	heartbeatScheduleID = "navidrome-music-room-heartbeat-v1"
	heartbeatPayload    = "sync-users-and-heartbeat"
	generationKey       = "sync-generation"
)

// TinyGo only applies -ldflags=-X reliably to an uninitialized string global.
// Keep the development fallback in effectiveVersion instead of initializing
// this variable, otherwise release builds silently report "dev" in heartbeats.
var version string

type bridgePlugin struct{}

type syncUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Admin       bool   `json:"admin"`
}

type syncPayload struct {
	PluginVersion        string     `json:"pluginVersion"`
	Generation           int64      `json:"generation"`
	NavidromeInternalURL string     `json:"navidromeInternalURL"`
	NavidromePublicURL   string     `json:"navidromePublicURL"`
	GatewayPublicURL     string     `json:"gatewayPublicURL"`
	UpdateChannel        string     `json:"updateChannel"`
	Users                []syncUser `json:"users"`
	SentAt               time.Time  `json:"sentAt"`
}

func init() {
	plugin := &bridgePlugin{}
	lifecycle.Register(plugin)
	scheduler.Register(plugin)
}

var (
	_ lifecycle.InitProvider     = (*bridgePlugin)(nil)
	_ scheduler.CallbackProvider = (*bridgePlugin)(nil)
)

func (p *bridgePlugin) OnInit() error {
	if !configBool("enabled", true) {
		cancelStoredSchedule()
		pdk.Log(pdk.LogInfo, "Music Room bridge is disabled by configuration")
		return nil
	}
	cancelStoredSchedule()
	scheduleID, err := host.SchedulerScheduleRecurring("@every 30s", heartbeatPayload, heartbeatScheduleID)
	if err != nil {
		return fmt.Errorf("schedule heartbeat: %w", err)
	}
	if err := host.KVStoreSet("heartbeat-schedule-id", []byte(scheduleID)); err != nil {
		return fmt.Errorf("remember heartbeat schedule: %w", err)
	}
	if err := syncGateway(); err != nil {
		pdk.Log(pdk.LogWarn, "Initial Music Room gateway sync failed: "+err.Error())
		// Keep the plugin initialized so the recurring callback can recover when
		// the local gateway finishes starting or a transient network error clears.
		return nil
	}
	pdk.Log(pdk.LogInfo, "Music Room bridge paired successfully")
	return nil
}

func (p *bridgePlugin) OnCallback(input scheduler.SchedulerCallbackRequest) error {
	if input.Payload != heartbeatPayload || !configBool("enabled", true) {
		return nil
	}
	return syncGateway()
}

func syncGateway() error {
	gatewayURL, ok := configString("gateway_internal_url")
	if !ok {
		return fmt.Errorf("gateway_internal_url is required")
	}
	pairingToken, ok := configString("pairing_token")
	if !ok || len(pairingToken) < 32 {
		return fmt.Errorf("pairing_token must contain at least 32 characters")
	}
	navidromeInternalURL, ok := configString("navidrome_internal_url")
	if !ok {
		return fmt.Errorf("navidrome_internal_url is required")
	}
	navidromePublicURL, ok := configString("navidrome_public_url")
	if !ok {
		return fmt.Errorf("navidrome_public_url is required")
	}
	gatewayPublicURL, ok := configString("gateway_public_url")
	if !ok {
		return fmt.Errorf("gateway_public_url is required")
	}
	users, err := host.UsersGetUsers()
	if err != nil {
		return fmt.Errorf("read authorized Navidrome users: %w", err)
	}
	syncUsers := make([]syncUser, 0, len(users))
	for _, user := range users {
		displayName := strings.TrimSpace(user.Name)
		if displayName == "" {
			displayName = user.UserName
		}
		syncUsers = append(syncUsers, syncUser{
			Username: user.UserName, DisplayName: displayName, Admin: user.IsAdmin,
		})
	}
	generation := nextGeneration()
	for attempt := 0; attempt < 2; attempt++ {
		payload, err := json.Marshal(syncPayload{
			PluginVersion: effectiveVersion(), Generation: generation,
			NavidromeInternalURL: strings.TrimRight(navidromeInternalURL, "/"),
			NavidromePublicURL:   strings.TrimRight(navidromePublicURL, "/"),
			GatewayPublicURL:     strings.TrimRight(gatewayPublicURL, "/"),
			UpdateChannel:        configDefaultString("update_channel", "stable"),
			Users:                syncUsers, SentAt: time.Now().UTC(),
		})
		if err != nil {
			return fmt.Errorf("encode sync payload: %w", err)
		}
		response, err := host.HTTPSend(host.HTTPRequest{
			Method: "POST",
			URL:    strings.TrimRight(gatewayURL, "/") + "/internal/v1/plugin-sync",
			Headers: map[string]string{
				"Authorization": "Bearer " + pairingToken,
				"Content-Type":  "application/json",
				"Accept":        "application/json",
			},
			Body: payload, TimeoutMs: 8000, NoFollowRedirects: true,
		})
		if err != nil {
			return fmt.Errorf("send gateway sync: %w", err)
		}
		if response != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			if err := host.KVStoreSet(generationKey, []byte(strconv.FormatInt(generation, 10))); err != nil {
				return fmt.Errorf("persist sync generation: %w", err)
			}
			return nil
		}
		if current, ok := staleGeneration(response); ok && attempt == 0 && current >= generation && current < 1<<62 {
			generation = current + 1
			continue
		}
		status := int32(0)
		if response != nil {
			status = response.StatusCode
		}
		return fmt.Errorf("gateway rejected sync with HTTP %d", status)
	}
	return fmt.Errorf("gateway rejected generation recovery")
}

func effectiveVersion() string {
	value := strings.TrimSpace(version)
	if value == "" {
		return "dev"
	}
	return value
}

func staleGeneration(response *host.HTTPResponse) (int64, bool) {
	if response == nil || response.StatusCode != 409 {
		return 0, false
	}
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Details struct {
				CurrentGeneration int64 `json:"currentGeneration"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil || payload.Error.Code != "stale_plugin_generation" || payload.Error.Details.CurrentGeneration <= 0 {
		return 0, false
	}
	return payload.Error.Details.CurrentGeneration, true
}

func nextGeneration() int64 {
	value, exists, err := host.KVStoreGet(generationKey)
	if err != nil || !exists {
		return 1
	}
	parsed, err := strconv.ParseInt(string(value), 10, 64)
	if err != nil || parsed < 0 {
		return 1
	}
	return parsed + 1
}

func cancelStoredSchedule() {
	value, exists, err := host.KVStoreGet("heartbeat-schedule-id")
	if err == nil && exists && len(value) > 0 {
		_ = host.SchedulerCancelSchedule(string(value))
	}
	_ = host.KVStoreDelete("heartbeat-schedule-id")
}

func configString(key string) (string, bool) {
	value, ok := host.ConfigGet(key)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}

func configOptionalString(key string) string {
	value, _ := host.ConfigGet(key)
	return strings.TrimSpace(value)
}

func configDefaultString(key, fallback string) string {
	value := configOptionalString(key)
	if value == "" {
		return fallback
	}
	return value
}

func configBool(key string, fallback bool) bool {
	value, ok := host.ConfigGet(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func main() {}
