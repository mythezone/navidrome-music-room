package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

func scanQueueEntry(row scanner) (domain.QueueEntry, error) {
	var entry domain.QueueEntry
	var trackJSON string
	var created int64
	err := row.Scan(
		&entry.QueueID, &entry.RoomID, &entry.Position, &trackJSON,
		&entry.Contributor, &entry.ContributorName, &created,
	)
	if err != nil {
		return domain.QueueEntry{}, err
	}
	if err := json.Unmarshal([]byte(trackJSON), &entry.Track); err != nil {
		return domain.QueueEntry{}, fmt.Errorf("decode queue track: %w", err)
	}
	entry.CreatedAt = fromUnixMillis(created)
	return entry, nil
}

func (s *Store) ListQueue(ctx context.Context, roomID string) ([]domain.QueueEntry, error) {
	return listQueue(ctx, s.db, roomID)
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func listQueue(ctx context.Context, q queryer, roomID string) ([]domain.QueueEntry, error) {
	rows, err := q.QueryContext(ctx, `
SELECT queue_id, room_id, position, track_json, contributor_username, contributor_display_name, created_unix_ms
FROM queue WHERE room_id = ? ORDER BY position`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.QueueEntry
	for rows.Next() {
		entry, err := scanQueueEntry(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, rows.Err()
}

func (s *Store) GetQueueEntry(ctx context.Context, roomID, queueID string) (domain.QueueEntry, error) {
	entry, err := scanQueueEntry(s.db.QueryRowContext(ctx, `
SELECT queue_id, room_id, position, track_json, contributor_username, contributor_display_name, created_unix_ms
FROM queue WHERE room_id = ? AND queue_id = ?`, roomID, queueID))
	if isNotFound(err) {
		return domain.QueueEntry{}, domain.NewError(404, "queue_entry_not_found", "Queue entry was not found")
	}
	return entry, err
}

func (s *Store) AddQueueEntry(ctx context.Context, room domain.Room, entry domain.QueueEntry) (domain.QueueEntry, error) {
	trackJSON, err := json.Marshal(entry.Track)
	if err != nil {
		return domain.QueueEntry{}, err
	}
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.QueueEntry{}, err
	}
	var status string
	var queueLimit int
	if err = tx.QueryRowContext(ctx, "SELECT status, queue_limit FROM rooms WHERE room_id = ?", room.RoomID).Scan(&status, &queueLimit); err != nil {
		_ = tx.Rollback()
		if isNotFound(err) {
			return domain.QueueEntry{}, domain.NewError(404, "room_not_found", "Room was not found")
		}
		return domain.QueueEntry{}, err
	}
	if status != domain.RoomOpen {
		_ = tx.Rollback()
		return domain.QueueEntry{}, domain.NewError(409, "room_closed", "Room is closed")
	}
	var ownCount, totalCount, maxPosition int
	if err = tx.QueryRowContext(ctx, `
SELECT
    COUNT(CASE WHEN contributor_username = ? COLLATE NOCASE THEN 1 END),
    COUNT(*),
    COALESCE(MAX(position), 0)
FROM queue WHERE room_id = ?`, entry.Contributor, room.RoomID).Scan(&ownCount, &totalCount, &maxPosition); err != nil {
		_ = tx.Rollback()
		return domain.QueueEntry{}, err
	}
	if ownCount >= queueLimit {
		_ = tx.Rollback()
		return domain.QueueEntry{}, domain.ErrorWithDetails(409, "personal_queue_limit", "Personal queue limit reached", map[string]int{"limit": queueLimit})
	}
	if totalCount >= 1000 {
		_ = tx.Rollback()
		return domain.QueueEntry{}, domain.NewError(409, "room_queue_full", "Room queue is full")
	}
	entry.Position = maxPosition + 1
	entry.CreatedAt = time.Now().UTC()
	if _, err = tx.ExecContext(ctx, `
INSERT INTO queue(queue_id, room_id, position, track_json, contributor_username, contributor_display_name, created_unix_ms)
VALUES (?, ?, ?, ?, ?, ?, ?)`, entry.QueueID, room.RoomID, entry.Position, string(trackJSON),
		entry.Contributor, entry.ContributorName, unixMillis(entry.CreatedAt)); err != nil {
		_ = tx.Rollback()
		return domain.QueueEntry{}, err
	}
	if err = insertAuditTx(ctx, tx, entry.Contributor, room.RoomID, "queue.added", map[string]string{
		"queueID": entry.QueueID, "trackID": entry.Track.ID,
	}); err != nil {
		_ = tx.Rollback()
		return domain.QueueEntry{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.QueueEntry{}, err
	}
	return entry, nil
}

func (s *Store) RemoveQueueEntry(ctx context.Context, roomID, queueID, actor string) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM queue WHERE room_id = ? AND queue_id = ?", roomID, queueID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		_ = tx.Rollback()
		return domain.NewError(404, "queue_entry_not_found", "Queue entry was not found")
	}
	if err = normalizeQueueTx(ctx, tx, roomID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = insertAuditTx(ctx, tx, actor, roomID, "queue.removed", map[string]string{"queueID": queueID}); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) ReorderQueue(ctx context.Context, roomID string, queueIDs []string, actor string) ([]domain.QueueEntry, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	entries, err := listQueue(ctx, tx, roomID)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if len(entries) != len(queueIDs) {
		_ = tx.Rollback()
		return nil, domain.NewError(409, "queue_order_stale", "Queue contents changed before the reorder request")
	}
	existing := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		existing[entry.QueueID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(queueIDs))
	for _, id := range queueIDs {
		if _, ok := existing[id]; !ok {
			_ = tx.Rollback()
			return nil, domain.NewError(409, "queue_order_stale", "Queue contents changed before the reorder request")
		}
		if _, duplicate := seen[id]; duplicate {
			_ = tx.Rollback()
			return nil, domain.NewError(400, "queue_order_invalid", "Queue order contains duplicate identifiers")
		}
		seen[id] = struct{}{}
	}
	for index, id := range queueIDs {
		if _, err = tx.ExecContext(ctx, "UPDATE queue SET position = ? WHERE room_id = ? AND queue_id = ?", -(index + 1), roomID, id); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	for index, id := range queueIDs {
		if _, err = tx.ExecContext(ctx, "UPDATE queue SET position = ? WHERE room_id = ? AND queue_id = ?", index+1, roomID, id); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err = insertAuditTx(ctx, tx, actor, roomID, "queue.reordered", nil); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.ListQueue(ctx, roomID)
}

func normalizeQueueTx(ctx context.Context, tx *sql.Tx, roomID string) error {
	rows, err := tx.QueryContext(ctx, "SELECT queue_id FROM queue WHERE room_id = ? ORDER BY position", roomID)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for index, id := range ids {
		if _, err := tx.ExecContext(ctx, "UPDATE queue SET position = ? WHERE queue_id = ?", -(index + 1), id); err != nil {
			return err
		}
	}
	for index, id := range ids {
		if _, err := tx.ExecContext(ctx, "UPDATE queue SET position = ? WHERE queue_id = ?", index+1, id); err != nil {
			return err
		}
	}
	return nil
}
