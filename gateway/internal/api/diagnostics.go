package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

func (s *Server) exportDiagnostics(w http.ResponseWriter, r *http.Request) {
	session, _, err := s.session(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !session.User.Admin {
		writeError(w, domain.NewError(403, "navidrome_admin_required", "Only a Navidrome administrator can export diagnostics"))
		return
	}

	database, err := s.store.Diagnostics(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	now := time.Now().UTC()
	plugin := map[string]any{"status": "not_paired", "authorizedUserCount": 0, "administratorCount": 0}
	channel := "stable"
	if state, stateErr := s.store.PluginState(r.Context()); stateErr == nil {
		administratorCount := 0
		for _, user := range state.Users {
			if user.Admin {
				administratorCount++
			}
		}
		age := max(int64(0), now.Sub(state.LastHeartbeat).Milliseconds())
		status := "paired"
		if now.Sub(state.LastHeartbeat) > s.config.PluginLease {
			status = "stale"
		}
		plugin = map[string]any{
			"status": status, "version": state.PluginVersion, "generation": state.Generation,
			"heartbeatAgeMilliseconds": age, "authorizedUserCount": len(state.Users),
			"administratorCount": administratorCount, "updateChannel": state.UpdateChannel,
		}
		if state.UpdateChannel == "beta" {
			channel = "beta"
		}
	}

	updateStatus := s.updater.Status(r.Context(), channel)
	licenseState := "not_installed"
	if state, stateErr := s.store.PluginState(r.Context()); stateErr == nil {
		licenseState = s.entitlementProvider.Verify(state.LicenseFile).State
	}
	bundle := map[string]any{
		"format":      "navidrome-music-room-diagnostics/v1",
		"generatedAt": now,
		"redacted":    true,
		"gateway": map[string]any{
			"version": s.config.Version, "goVersion": runtime.Version(),
			"os": runtime.GOOS, "architecture": runtime.GOARCH,
			"managedByLauncher": s.config.ManagedByLauncher,
		},
		"configuration": map[string]any{
			"pluginLease": s.config.PluginLease.String(), "existingSessionGrace": s.config.ExistingGrace.String(),
			"sessionTTL": s.config.SessionTTL.String(), "webSocketTicketTTL": s.config.WebSocketTicketTTL.String(),
			"emptyRoomPauseDelay": s.config.EmptyRoomPauseDelay.String(), "allowedOriginCount": len(s.config.AllowedOrigins),
		},
		"plugin":   plugin,
		"database": database,
		"license":  map[string]any{"state": licenseState, "offlineVerification": true},
		"update": map[string]any{
			"channel": updateStatus.Channel, "currentVersion": updateStatus.CurrentVersion,
			"latestVersion": updateStatus.LatestVersion, "stagedVersion": updateStatus.StagedVersion,
			"rollbackVersion": updateStatus.RollbackVersion, "updateAvailable": updateStatus.UpdateAvailable,
			"rollbackAvailable": updateStatus.RollbackAvailable, "state": updateStatus.State,
		},
	}

	_ = s.store.Audit(r.Context(), session.User.Username, "", "diagnostics.exported", nil)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Disposition", `attachment; filename="music-room-diagnostics-`+now.Format("20060102T150405Z")+`.json"`)
	writeJSON(w, http.StatusOK, bundle)
}
