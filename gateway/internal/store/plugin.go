package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

func (s *Store) SavePluginSync(ctx context.Context, input domain.PluginSync, receivedAt time.Time) (domain.PluginState, error) {
	usersJSON, err := json.Marshal(input.Users)
	if err != nil {
		return domain.PluginState{}, err
	}
	s.writeLock.Lock()
	defer s.writeLock.Unlock()

	var previous int64
	err = s.db.QueryRowContext(ctx, "SELECT generation FROM plugin_state WHERE singleton = 1").Scan(&previous)
	if err != nil && !isNotFound(err) {
		return domain.PluginState{}, err
	}
	if err == nil && input.Generation < previous {
		return domain.PluginState{}, domain.ErrorWithDetails(409, "stale_plugin_generation", "Plugin sync generation is older than the stored generation", map[string]int64{
			"currentGeneration":  previous,
			"receivedGeneration": input.Generation,
		})
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO plugin_state(singleton, plugin_version, generation, navidrome_public_url, gateway_public_url, license_file, update_channel, users_json, last_heartbeat_unix_ms)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(singleton) DO UPDATE SET
    plugin_version = excluded.plugin_version,
    generation = excluded.generation,
    navidrome_public_url = excluded.navidrome_public_url,
    gateway_public_url = excluded.gateway_public_url,
	license_file = excluded.license_file,
	update_channel = excluded.update_channel,
    users_json = excluded.users_json,
    last_heartbeat_unix_ms = excluded.last_heartbeat_unix_ms`,
		input.PluginVersion, input.Generation, input.NavidromePublicURL, input.GatewayPublicURL, input.LicenseFile, input.UpdateChannel,
		string(usersJSON), unixMillis(receivedAt),
	)
	if err != nil {
		return domain.PluginState{}, err
	}
	return domain.PluginState{
		PluginVersion: input.PluginVersion, Generation: input.Generation,
		NavidromePublicURL: input.NavidromePublicURL, GatewayPublicURL: input.GatewayPublicURL,
		LicenseFile:   input.LicenseFile,
		UpdateChannel: input.UpdateChannel,
		Users:         input.Users, LastHeartbeat: receivedAt.UTC(),
	}, nil
}

func (s *Store) PluginState(ctx context.Context) (domain.PluginState, error) {
	var state domain.PluginState
	var usersJSON string
	var heartbeat int64
	err := s.db.QueryRowContext(ctx, `
SELECT plugin_version, generation, navidrome_public_url, gateway_public_url, license_file, update_channel, users_json, last_heartbeat_unix_ms
FROM plugin_state WHERE singleton = 1`).Scan(
		&state.PluginVersion, &state.Generation, &state.NavidromePublicURL, &state.GatewayPublicURL,
		&state.LicenseFile, &state.UpdateChannel, &usersJSON, &heartbeat,
	)
	if isNotFound(err) {
		return domain.PluginState{}, domain.NewError(503, "plugin_not_paired", "Navidrome plugin has not paired with the gateway")
	}
	if err != nil {
		return domain.PluginState{}, err
	}
	if err := json.Unmarshal([]byte(usersJSON), &state.Users); err != nil {
		return domain.PluginState{}, fmt.Errorf("decode plugin users: %w", err)
	}
	state.LastHeartbeat = fromUnixMillis(heartbeat)
	return state, nil
}

func (s *Store) PutUpdateState(ctx context.Context, key string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO update_state(key, value_json, updated_unix_ms) VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_unix_ms = excluded.updated_unix_ms`,
		key, string(payload), unixMillis(time.Now().UTC()))
	return err
}

func (s *Store) GetUpdateState(ctx context.Context, key string, target any) (bool, error) {
	var payload string
	err := s.db.QueryRowContext(ctx, "SELECT value_json FROM update_state WHERE key = ?", key).Scan(&payload)
	if isNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal([]byte(payload), target)
}

func (s *Store) Audit(ctx context.Context, username, roomID, action string, metadata any) error {
	payload := "{}"
	if metadata != nil {
		encoded, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		payload = string(encoded)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO security_audit(occurred_unix_ms, username, room_id, action, metadata_json)
VALUES (?, ?, ?, ?, ?)`, unixMillis(time.Now().UTC()), username, roomID, action, payload)
	return err
}
