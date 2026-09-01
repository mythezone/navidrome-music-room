package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
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
	writeJSON(w, 200, map[string]any{
		"status":            s.updater.Status(r.Context(), channel),
		"repository":        s.config.ReleaseRepository,
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
