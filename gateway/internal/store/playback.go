package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

type playbackRecord struct {
	State            domain.PlaybackState
	LastContributor  string
	CurrentHistoryID string
	PlaybackMode     string
}

func scanPlayback(row scanner, now time.Time) (playbackRecord, error) {
	var record playbackRecord
	var paused int
	var anchor sql.NullInt64
	var trackJSON sql.NullString
	err := row.Scan(
		&record.State.Revision, &record.State.Status, &paused, &record.State.PositionSeconds,
		&anchor, &trackJSON, &record.State.Contributor, &record.State.ContributorName,
		&record.CurrentHistoryID, &record.LastContributor, &record.PlaybackMode,
	)
	if err != nil {
		return playbackRecord{}, err
	}
	record.State.PausedForEmpty = paused != 0
	record.State.ServerTime = now.UTC()
	record.State.AnchorServerTime = nullableTime(anchor)
	if trackJSON.Valid && trackJSON.String != "" {
		var track domain.NavidromeTrackRef
		if err := json.Unmarshal([]byte(trackJSON.String), &track); err != nil {
			return playbackRecord{}, fmt.Errorf("decode playback track: %w", err)
		}
		record.State.CurrentTrack = &track
	}
	record.State.PositionSeconds = domain.EffectivePosition(record.State, now)
	return record, nil
}

func playbackQuery(q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, ctx context.Context, roomID string, now time.Time) (playbackRecord, error) {
	return scanPlayback(q.QueryRowContext(ctx, `
SELECT revision, playback_status, paused_for_empty, position_seconds, anchor_unix_ms,
       current_track_json, current_contributor, current_contributor_name,
       current_history_id, last_contributor, playback_mode
FROM rooms WHERE room_id = ?`, roomID), now)
}

func (s *Store) Playback(ctx context.Context, roomID string) (domain.PlaybackState, error) {
	now := time.Now().UTC()
	record, err := playbackQuery(s.db, ctx, roomID, now)
	if isNotFound(err) {
		return domain.PlaybackState{}, domain.NewError(404, "room_not_found", "Room was not found")
	}
	if err != nil {
		return domain.PlaybackState{}, err
	}
	queue, err := s.ListQueue(ctx, roomID)
	if err != nil {
		return domain.PlaybackState{}, err
	}
	if next := domain.SelectNext(queue, record.PlaybackMode, record.LastContributor); next != nil {
		record.State.NextTrack = &next.Track
	}
	return record.State, nil
}

func (s *Store) ApplyPlayback(ctx context.Context, roomID, command string, position *float64, expectedRevision int64, actor string) (domain.PlaybackState, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.PlaybackState{}, err
	}
	record, err := playbackQuery(tx, ctx, roomID, now)
	if isNotFound(err) {
		_ = tx.Rollback()
		return domain.PlaybackState{}, domain.NewError(404, "room_not_found", "Room was not found")
	}
	if err != nil {
		_ = tx.Rollback()
		return domain.PlaybackState{}, err
	}
	if record.State.Revision != expectedRevision {
		_ = tx.Rollback()
		latest, latestErr := s.Playback(ctx, roomID)
		if latestErr != nil {
			return domain.PlaybackState{}, latestErr
		}
		return domain.PlaybackState{}, domain.ErrorWithDetails(409, "revision_conflict", "Playback state has changed", map[string]any{"latest": latest})
	}

	switch command {
	case "play":
		if record.State.CurrentTrack == nil {
			if err := startNextTrackTx(ctx, tx, roomID, &record, now); err != nil {
				_ = tx.Rollback()
				return domain.PlaybackState{}, err
			}
		}
		if record.State.CurrentTrack == nil {
			_ = tx.Rollback()
			return domain.PlaybackState{}, domain.NewError(409, "queue_empty", "Queue is empty")
		}
		if err := startCurrentHistoryTx(ctx, tx, roomID, &record, now); err != nil {
			_ = tx.Rollback()
			return domain.PlaybackState{}, err
		}
		record.State.Status = domain.PlaybackPlaying
		record.State.PausedForEmpty = false
		record.State.AnchorServerTime = timePointer(now)
	case "pause":
		if record.State.CurrentTrack == nil {
			_ = tx.Rollback()
			return domain.PlaybackState{}, domain.NewError(409, "no_current_track", "No track is loaded")
		}
		record.State.Status = domain.PlaybackPaused
		record.State.AnchorServerTime = nil
	case "seek":
		if record.State.CurrentTrack == nil || position == nil {
			_ = tx.Rollback()
			return domain.PlaybackState{}, domain.NewError(400, "seek_invalid", "Seek requires a current track and position")
		}
		record.State.PositionSeconds = clampPosition(*position, record.State.CurrentTrack.DurationSeconds)
		if record.State.Status == domain.PlaybackPlaying {
			record.State.AnchorServerTime = timePointer(now)
		} else {
			record.State.AnchorServerTime = nil
		}
	case "next":
		if err := finishCurrentTrackTx(ctx, tx, roomID, &record, now); err != nil {
			_ = tx.Rollback()
			return domain.PlaybackState{}, err
		}
		if err := startNextTrackTx(ctx, tx, roomID, &record, now); err != nil {
			_ = tx.Rollback()
			return domain.PlaybackState{}, err
		}
		if record.State.CurrentTrack != nil {
			record.State.Status = domain.PlaybackPlaying
			record.State.AnchorServerTime = timePointer(now)
		}
	case "stop":
		if err := finishCurrentTrackTx(ctx, tx, roomID, &record, now); err != nil {
			_ = tx.Rollback()
			return domain.PlaybackState{}, err
		}
		record.State.Status = domain.PlaybackStopped
		record.State.CurrentTrack = nil
		record.State.PositionSeconds = 0
		record.State.AnchorServerTime = nil
		record.State.Contributor = ""
		record.State.ContributorName = ""
	default:
		_ = tx.Rollback()
		return domain.PlaybackState{}, domain.NewError(400, "playback_command_invalid", "Unsupported playback command")
	}
	record.State.Revision++
	record.State.ServerTime = now
	if err := persistPlaybackTx(ctx, tx, roomID, record, now); err != nil {
		_ = tx.Rollback()
		return domain.PlaybackState{}, err
	}
	if err := insertAuditTx(ctx, tx, actor, roomID, "playback."+command, map[string]any{"revision": record.State.Revision}); err != nil {
		_ = tx.Rollback()
		return domain.PlaybackState{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.PlaybackState{}, err
	}
	return s.Playback(ctx, roomID)
}

func persistPlaybackTx(ctx context.Context, tx *sql.Tx, roomID string, record playbackRecord, now time.Time) error {
	var anchor any
	if record.State.AnchorServerTime != nil {
		anchor = unixMillis(*record.State.AnchorServerTime)
	}
	var trackJSON any
	if record.State.CurrentTrack != nil {
		encoded, err := json.Marshal(record.State.CurrentTrack)
		if err != nil {
			return err
		}
		trackJSON = string(encoded)
	}
	_, err := tx.ExecContext(ctx, `
UPDATE rooms SET revision = ?, playback_status = ?, paused_for_empty = ?, position_seconds = ?,
    anchor_unix_ms = ?, current_track_json = ?, current_contributor = ?, current_contributor_name = ?,
    current_history_id = ?, last_contributor = ?, updated_unix_ms = ?
WHERE room_id = ?`, record.State.Revision, record.State.Status, boolInt(record.State.PausedForEmpty),
		record.State.PositionSeconds, anchor, trackJSON, record.State.Contributor, record.State.ContributorName,
		record.CurrentHistoryID, record.LastContributor, unixMillis(now), roomID)
	return err
}

func loadNextTrackTx(ctx context.Context, tx *sql.Tx, roomID string, record *playbackRecord) (bool, error) {
	queue, err := listQueue(ctx, tx, roomID)
	if err != nil {
		return false, err
	}
	next := domain.SelectNext(queue, record.PlaybackMode, record.LastContributor)
	if next == nil {
		record.State.Status = domain.PlaybackStopped
		record.State.CurrentTrack = nil
		record.State.PositionSeconds = 0
		record.State.AnchorServerTime = nil
		record.State.Contributor = ""
		record.State.ContributorName = ""
		record.CurrentHistoryID = ""
		return false, nil
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM queue WHERE room_id = ? AND queue_id = ?", roomID, next.QueueID); err != nil {
		return false, err
	}
	if err := normalizeQueueTx(ctx, tx, roomID); err != nil {
		return false, err
	}
	record.State.CurrentTrack = &next.Track
	record.State.PositionSeconds = 0
	record.State.PausedForEmpty = false
	record.State.Contributor = next.Contributor
	record.State.ContributorName = next.ContributorName
	record.CurrentHistoryID = ""
	record.LastContributor = next.Contributor
	return true, nil
}

func startCurrentHistoryTx(ctx context.Context, tx *sql.Tx, roomID string, record *playbackRecord, now time.Time) error {
	if record.State.CurrentTrack == nil || record.CurrentHistoryID != "" {
		return nil
	}
	historyID, err := domain.NewID()
	if err != nil {
		return err
	}
	trackJSON, err := json.Marshal(record.State.CurrentTrack)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO playback_history(history_id, room_id, track_json, contributor_username, contributor_display_name, started_unix_ms)
VALUES (?, ?, ?, ?, ?, ?)`, historyID, roomID, string(trackJSON), record.State.Contributor, record.State.ContributorName, unixMillis(now)); err != nil {
		return err
	}
	record.CurrentHistoryID = historyID
	return nil
}

func startNextTrackTx(ctx context.Context, tx *sql.Tx, roomID string, record *playbackRecord, now time.Time) error {
	loaded, err := loadNextTrackTx(ctx, tx, roomID, record)
	if err != nil || !loaded {
		return err
	}
	return startCurrentHistoryTx(ctx, tx, roomID, record, now)
}

func primeNextTrackTx(ctx context.Context, tx *sql.Tx, roomID string, record *playbackRecord, now time.Time) (bool, error) {
	if record.State.CurrentTrack != nil {
		return false, nil
	}
	loaded, err := loadNextTrackTx(ctx, tx, roomID, record)
	if err != nil || !loaded {
		return loaded, err
	}
	record.State.Status = domain.PlaybackPaused
	record.State.PausedForEmpty = false
	record.State.PositionSeconds = 0
	record.State.AnchorServerTime = nil
	record.State.ServerTime = now.UTC()
	return true, nil
}

// PrimePlaybackIfIdle repairs rooms created by older gateway versions where a
// pending queue existed without a current track. It is also safe to call more
// than once: only the first successful caller changes the revision.
func (s *Store) PrimePlaybackIfIdle(ctx context.Context, roomID, actor string) (domain.PlaybackState, bool, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.PlaybackState{}, false, err
	}
	record, err := playbackQuery(tx, ctx, roomID, now)
	if err != nil {
		_ = tx.Rollback()
		return domain.PlaybackState{}, false, err
	}
	primed, err := primeNextTrackTx(ctx, tx, roomID, &record, now)
	if err != nil {
		_ = tx.Rollback()
		return domain.PlaybackState{}, false, err
	}
	if !primed {
		_ = tx.Rollback()
		return record.State, false, nil
	}
	record.State.Revision++
	if err := persistPlaybackTx(ctx, tx, roomID, record, now); err != nil {
		_ = tx.Rollback()
		return domain.PlaybackState{}, false, err
	}
	if err := insertAuditTx(ctx, tx, actor, roomID, "playback.primed", map[string]any{
		"revision": record.State.Revision, "trackID": record.State.CurrentTrack.ID,
	}); err != nil {
		_ = tx.Rollback()
		return domain.PlaybackState{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.PlaybackState{}, false, err
	}
	state, err := s.Playback(ctx, roomID)
	return state, true, err
}

func (s *Store) ListIdleQueuedRoomIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT room_id FROM rooms
WHERE current_track_json IS NULL
  AND EXISTS (SELECT 1 FROM queue WHERE queue.room_id = rooms.room_id)
ORDER BY room_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]string, 0)
	for rows.Next() {
		var roomID string
		if err := rows.Scan(&roomID); err != nil {
			return nil, err
		}
		result = append(result, roomID)
	}
	return result, rows.Err()
}

func finishCurrentTrackTx(ctx context.Context, tx *sql.Tx, roomID string, record *playbackRecord, now time.Time) error {
	if record.CurrentHistoryID != "" {
		position := record.State.PositionSeconds
		if record.State.CurrentTrack != nil {
			position = clampPosition(position, record.State.CurrentTrack.DurationSeconds)
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE playback_history SET finished_unix_ms = ?, played_seconds = ?
WHERE room_id = ? AND history_id = ? AND finished_unix_ms IS NULL`,
			unixMillis(now), position, roomID, record.CurrentHistoryID); err != nil {
			return err
		}
	}
	record.CurrentHistoryID = ""
	record.State.CurrentTrack = nil
	record.State.PositionSeconds = 0
	record.State.AnchorServerTime = nil
	record.State.Contributor = ""
	record.State.ContributorName = ""
	return nil
}

func (s *Store) PauseForEmpty(ctx context.Context, roomID string) (domain.PlaybackState, bool, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.PlaybackState{}, false, err
	}
	record, err := playbackQuery(tx, ctx, roomID, now)
	if err != nil {
		_ = tx.Rollback()
		return domain.PlaybackState{}, false, err
	}
	if record.State.Status != domain.PlaybackPlaying || record.State.CurrentTrack == nil {
		_ = tx.Rollback()
		return record.State, false, nil
	}
	record.State.Status = domain.PlaybackPaused
	record.State.PausedForEmpty = true
	record.State.AnchorServerTime = nil
	record.State.Revision++
	if err := persistPlaybackTx(ctx, tx, roomID, record, now); err != nil {
		_ = tx.Rollback()
		return domain.PlaybackState{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.PlaybackState{}, false, err
	}
	state, err := s.Playback(ctx, roomID)
	return state, true, err
}

func (s *Store) AdvanceFinished(ctx context.Context, roomID string) (domain.PlaybackState, bool, error) {
	state, err := s.Playback(ctx, roomID)
	if err != nil {
		return domain.PlaybackState{}, false, err
	}
	if state.Status != domain.PlaybackPlaying || state.CurrentTrack == nil || state.CurrentTrack.DurationSeconds <= 0 {
		return state, false, nil
	}
	if state.PositionSeconds+0.05 < state.CurrentTrack.DurationSeconds {
		return state, false, nil
	}
	next, err := s.ApplyPlayback(ctx, roomID, "next", nil, state.Revision, "__clock__")
	if err != nil {
		if roomErr, ok := err.(*domain.Error); ok && roomErr.Code == "revision_conflict" {
			return state, false, nil
		}
		return domain.PlaybackState{}, false, err
	}
	return next, true, nil
}

func (s *Store) ListPlayingRoomIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT room_id FROM rooms WHERE playback_status = 'playing'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]string, 0)
	for rows.Next() {
		var roomID string
		if err := rows.Scan(&roomID); err != nil {
			return nil, err
		}
		result = append(result, roomID)
	}
	return result, rows.Err()
}

func scanHistory(row scanner) (domain.HistoryEntry, error) {
	var entry domain.HistoryEntry
	var trackJSON string
	var started int64
	var finished sql.NullInt64
	err := row.Scan(&entry.HistoryID, &entry.RoomID, &trackJSON, &entry.Contributor,
		&entry.ContributorName, &started, &finished, &entry.PlayedSeconds)
	if err != nil {
		return domain.HistoryEntry{}, err
	}
	if err := json.Unmarshal([]byte(trackJSON), &entry.Track); err != nil {
		return domain.HistoryEntry{}, fmt.Errorf("decode history track: %w", err)
	}
	entry.StartedAt = fromUnixMillis(started)
	entry.FinishedAt = nullableTime(finished)
	return entry, nil
}

func (s *Store) History(ctx context.Context, roomID string, limit, offset int) ([]domain.HistoryEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT history_id, room_id, track_json, contributor_username, contributor_display_name,
       started_unix_ms, finished_unix_ms, played_seconds
FROM playback_history WHERE room_id = ? ORDER BY started_unix_ms DESC LIMIT ? OFFSET ?`, roomID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]domain.HistoryEntry, 0)
	for rows.Next() {
		entry, err := scanHistory(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func clampPosition(value, duration float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	if duration > 0 && value > duration {
		return duration
	}
	return value
}

func timePointer(value time.Time) *time.Time {
	copy := value.UTC()
	return &copy
}
