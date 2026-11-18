package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"strings"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
	updatemanager "github.com/mythezone/navidrome-music-room/gateway/internal/update"
)

type updateRequest struct {
	Action  string `json:"action"`
	Version string `json:"version,omitempty"`
}

func (s *Server) updateStatus(w http.ResponseWriter, r *http.Request) {
	session, _, err := s.session(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !session.User.Admin {
		writeError(w, domain.NewError(403, "navidrome_admin_required", "Only a Navidrome administrator can manage updates"))
		return
	}
	channel := s.updateChannel(r)
	plugin, pluginErr := s.store.PluginState(r.Context())
	paired := pluginErr == nil && time.Since(plugin.LastHeartbeat) <= s.config.PluginLease
	writeJSON(w, 200, map[string]any{
		"status":        s.updater.Status(r.Context(), channel),
		"repository":    s.config.ReleaseRepository,
		"pluginVersion": plugin.PluginVersion, "pluginPaired": paired,
		"versionMismatch":   paired && strings.TrimPrefix(strings.TrimSpace(plugin.PluginVersion), "v") != strings.TrimPrefix(strings.TrimSpace(s.config.Version), "v"),
		"managedByLauncher": s.config.ManagedByLauncher,
		"actions":           []string{"check", "stage", "install", "rollback"},
	})
}

func (s *Server) updateAction(w http.ResponseWriter, r *http.Request) {
	session, _, err := s.session(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !session.User.Admin {
		writeError(w, domain.NewError(403, "navidrome_admin_required", "Only a Navidrome administrator can manage updates"))
		return
	}
	var input updateRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.Action = strings.ToLower(strings.TrimSpace(input.Action))
	channel := s.updateChannel(r)
	switch input.Action {
	case "check":
		status, err := s.updater.Check(r.Context(), channel)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"status": status})
	case "stage", "install":
		if input.Action == "install" && (!s.config.ManagedByLauncher || s.restart == nil) {
			writeError(w, domain.NewError(409, "launcher_required", "Automatic activation requires the stable Music Room launcher"))
			return
		}
		pending, err := s.updater.Stage(r.Context(), strings.TrimSpace(input.Version), channel)
		if err != nil {
			writeError(w, err)
			return
		}
		_ = s.store.Audit(r.Context(), session.User.Username, "", "update.staged", map[string]string{"version": pending.Version})
		writeJSON(w, 202, map[string]any{
			"state": "staged", "version": pending.Version,
			"activationScheduled": input.Action == "install",
			"nextStep":            "After activation, MusicMate must rescan and re-enable the Navidrome plugin, then confirm its heartbeat.",
		})
		if input.Action == "install" {
			time.AfterFunc(750*time.Millisecond, s.restart)
		}
	case "rollback":
		if !s.config.ManagedByLauncher || s.restart == nil {
			writeError(w, domain.NewError(409, "launcher_required", "Automatic rollback requires the stable Music Room launcher"))
			return
		}
		targetVersion, err := s.updater.RequestRollback(r.Context(), channel)
		if err != nil {
			writeError(w, err)
			return
		}
		_ = s.store.Audit(r.Context(), session.User.Username, "", "update.rollback_requested", map[string]string{"version": targetVersion})
		writeJSON(w, 202, map[string]any{"state": "rollback_requested", "version": targetVersion})
		time.AfterFunc(750*time.Millisecond, s.restart)
	default:
		writeError(w, domain.NewError(400, "update_action_invalid", "Update action must be check, stage, install, or rollback"))
	}
}

func (s *Server) updateChannel(r *http.Request) string {
	state, err := s.store.PluginState(r.Context())
	if err == nil && state.UpdateChannel == "beta" {
		return "beta"
	}
	return "stable"
}

func updateNotice(status updatemanager.Status) domain.UpdateNotice {
	return domain.UpdateNotice{CurrentVersion: status.CurrentVersion, LatestVersion: status.LatestVersion, Channel: status.Channel, ReleaseURL: status.ReleaseURL}
}

func (s *Server) RunUpdateChecker(ctx context.Context) {
	interval := s.config.UpdateCheckInterval
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	timer := time.NewTimer(25 * time.Second)
	defer timer.Stop()
	var scheduled <-chan time.Time
	if s.config.AutoUpdateCheck {
		scheduled = timer.C
	} else {
		timer.Stop()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-scheduled:
		case <-s.updateChecks:
		}
		channel := "stable"
		if state, err := s.store.PluginState(ctx); err == nil && state.UpdateChannel == "beta" {
			channel = "beta"
		}
		status, err := s.updater.Check(ctx, channel)
		if err == nil {
			err = s.notifyUpdate(ctx, status)
		}
		if err != nil && ctx.Err() == nil {
			s.logger.Warn("update check failed", "error", err)
		}
		if s.config.AutoUpdateCheck {
			timer.Reset(interval)
		}
	}
}

func (s *Server) notifyUpdate(ctx context.Context, status updatemanager.Status) error {
	if !status.UpdateAvailable || status.LatestVersion == "" {
		return nil
	}
	var notified string
	if _, err := s.store.GetUpdateState(ctx, "notified_version", &notified); err != nil {
		return err
	}
	if status.LatestVersion == notified {
		return nil
	}
	state, err := s.store.PluginState(ctx)
	if err != nil {
		if roomErr, ok := err.(*domain.Error); !ok || roomErr.Code != "plugin_not_paired" {
			return err
		}
	}
	if err = s.store.PutUpdateState(ctx, "notified_version", status.LatestVersion); err != nil {
		return err
	}
	s.logger.Info("new release available", "current_version", status.CurrentVersion, "latest_version", status.LatestVersion, "channel", status.Channel, "release_url", status.ReleaseURL)
	s.hub.BroadcastUsers(func(username string) bool { user, ok := state.User(username); return ok && user.Admin }, domain.Event{Type: "update_available", Payload: updateNotice(status)})
	if s.config.UpdateWebhookURL == "" {
		return nil
	}
	payload, err := json.Marshal(map[string]string{
		"event": "update_available", "repository": s.config.ReleaseRepository, "currentVersion": status.CurrentVersion,
		"latestVersion": status.LatestVersion, "channel": status.Channel, "releaseURL": status.ReleaseURL, "publishedAt": status.PublishedAt,
	})
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.UpdateWebhookURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("update webhook request failed")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("update webhook returned HTTP %d", response.StatusCode)
	}
	return nil
}
