package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

const roomColumns = `
r.room_id, r.name, r.owner_username, r.owner_display_name, r.status,
r.queue_limit, r.playback_mode, r.music_folder_ids_json, r.preload_next_track,
r.created_unix_ms, r.updated_unix_ms`

type scanner interface{ Scan(...any) error }

func scanRoom(row scanner) (domain.Room, error) {
	var room domain.Room
	var foldersJSON string
	var preload int
	var created, updated int64
	err := row.Scan(
		&room.RoomID, &room.Name, &room.OwnerUsername, &room.OwnerDisplayName, &room.Status,
		&room.QueueLimit, &room.PlaybackMode, &foldersJSON, &preload, &created, &updated,
	)
	if err != nil {
		return domain.Room{}, err
	}
	if err := json.Unmarshal([]byte(foldersJSON), &room.MusicFolderIDs); err != nil {
		return domain.Room{}, fmt.Errorf("decode room music folders: %w", err)
	}
	room.PreloadNextTrack = preload != 0
	room.Capabilities = domain.FreeCapabilities()
	room.CreatedAt = fromUnixMillis(created)
	room.UpdatedAt = fromUnixMillis(updated)
	return room, nil
}

func (s *Store) CreateRoom(ctx context.Context, room domain.Room) (domain.Room, error) {
	foldersJSON, err := json.Marshal(room.MusicFolderIDs)
	if err != nil {
		return domain.Room{}, err
	}
	now := time.Now().UTC()
	room.CreatedAt = now
	room.UpdatedAt = now
	room.Status = domain.RoomOpen
	room.Capabilities = domain.FreeCapabilities()

	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Room{}, err
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO rooms(
    room_id, name, owner_username, owner_display_name, status, queue_limit, playback_mode,
    music_folder_ids_json, preload_next_track, created_unix_ms, updated_unix_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		room.RoomID, room.Name, room.OwnerUsername, room.OwnerDisplayName, room.Status,
		room.QueueLimit, room.PlaybackMode, string(foldersJSON), boolInt(room.PreloadNextTrack),
		unixMillis(now), unixMillis(now),
	); err != nil {
		_ = tx.Rollback()
		return domain.Room{}, err
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO members(room_id, username, display_name, role, active, joined_unix_ms, last_seen_unix_ms)
VALUES (?, ?, ?, 'owner', 1, ?, ?)`,
		room.RoomID, room.OwnerUsername, room.OwnerDisplayName, unixMillis(now), unixMillis(now),
	); err != nil {
		_ = tx.Rollback()
		return domain.Room{}, err
	}
	if err = insertAuditTx(ctx, tx, room.OwnerUsername, room.RoomID, "room.created", map[string]any{
		"musicFolderIDs": room.MusicFolderIDs,
	}); err != nil {
		_ = tx.Rollback()
		return domain.Room{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.Room{}, err
	}
	return room, nil
}

func (s *Store) GetRoom(ctx context.Context, roomID string) (domain.Room, error) {
	room, err := scanRoom(s.db.QueryRowContext(ctx, "SELECT "+roomColumns+" FROM rooms r WHERE r.room_id = ?", roomID))
	if isNotFound(err) {
		return domain.Room{}, domain.NewError(404, "room_not_found", "Room was not found")
	}
	return room, err
}

func (s *Store) ListRooms(ctx context.Context, username string, isAdmin bool) ([]domain.Room, error) {
	query := "SELECT " + roomColumns + " FROM rooms r"
	var args []any
	if !isAdmin {
		query += " JOIN members m ON m.room_id = r.room_id AND m.username = ? AND m.active = 1"
		args = append(args, username)
	}
	query += " ORDER BY r.updated_unix_ms DESC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Room
	for rows.Next() {
		room, err := scanRoom(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, room)
	}
	return result, rows.Err()
}

func (s *Store) UpdateRoom(ctx context.Context, room domain.Room, actor string) (domain.Room, error) {
	foldersJSON, err := json.Marshal(room.MusicFolderIDs)
	if err != nil {
		return domain.Room{}, err
	}
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
UPDATE rooms SET name = ?, queue_limit = ?, playback_mode = ?, music_folder_ids_json = ?,
    preload_next_track = ?, updated_unix_ms = ?
WHERE room_id = ?`, room.Name, room.QueueLimit, room.PlaybackMode, string(foldersJSON),
		boolInt(room.PreloadNextTrack), unixMillis(now), room.RoomID)
	if err != nil {
		return domain.Room{}, err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return domain.Room{}, domain.NewError(404, "room_not_found", "Room was not found")
	}
	room.UpdatedAt = now
	room.Capabilities = domain.FreeCapabilities()
	_ = s.Audit(ctx, actor, room.RoomID, "room.updated", nil)
	return room, nil
}

func (s *Store) SetRoomStatus(ctx context.Context, roomID, status, actor string) (domain.Room, error) {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, "UPDATE rooms SET status = ?, updated_unix_ms = ? WHERE room_id = ?", status, unixMillis(now), roomID)
	if err != nil {
		return domain.Room{}, err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return domain.Room{}, domain.NewError(404, "room_not_found", "Room was not found")
	}
	_ = s.Audit(ctx, actor, roomID, "room."+status, nil)
	return s.GetRoom(ctx, roomID)
}

func (s *Store) DeleteRoom(ctx context.Context, roomID, actor string) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := insertAuditTx(ctx, tx, actor, roomID, "room.deleted", nil); err != nil {
		_ = tx.Rollback()
		return err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM rooms WHERE room_id = ?", roomID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		_ = tx.Rollback()
		return domain.NewError(404, "room_not_found", "Room was not found")
	}
	return tx.Commit()
}

func scanMember(row scanner) (domain.Member, error) {
	var member domain.Member
	var active int
	var joined, seen int64
	err := row.Scan(&member.RoomID, &member.Username, &member.DisplayName, &member.Role, &active, &joined, &seen)
	if err != nil {
		return domain.Member{}, err
	}
	member.Active = active != 0
	member.JoinedAt = fromUnixMillis(joined)
	member.LastSeenAt = fromUnixMillis(seen)
	return member, nil
}

func (s *Store) GetMember(ctx context.Context, roomID, username string) (domain.Member, error) {
	member, err := scanMember(s.db.QueryRowContext(ctx, `
SELECT room_id, username, display_name, role, active, joined_unix_ms, last_seen_unix_ms
FROM members WHERE room_id = ? AND username = ? COLLATE NOCASE`, roomID, username))
	if isNotFound(err) {
		return domain.Member{}, domain.NewError(403, "membership_required", "An invitation must be redeemed before joining this room")
	}
	return member, err
}

func (s *Store) ListMembers(ctx context.Context, roomID string) ([]domain.Member, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT room_id, username, display_name, role, active, joined_unix_ms, last_seen_unix_ms
FROM members WHERE room_id = ? ORDER BY role DESC, joined_unix_ms`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Member
	for rows.Next() {
		member, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, member)
	}
	return result, rows.Err()
}

func (s *Store) UpsertMember(ctx context.Context, roomID string, user domain.User, role, actor, action string) (domain.Member, error) {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO members(room_id, username, display_name, role, active, joined_unix_ms, last_seen_unix_ms)
VALUES (?, ?, ?, ?, 1, ?, ?)
ON CONFLICT(room_id, username) DO UPDATE SET
    display_name = excluded.display_name, active = 1, last_seen_unix_ms = excluded.last_seen_unix_ms`,
		roomID, user.Username, user.DisplayName, role, unixMillis(now), unixMillis(now))
	if err != nil {
		return domain.Member{}, err
	}
	_ = s.Audit(ctx, actor, roomID, action, map[string]string{"member": user.Username})
	return s.GetMember(ctx, roomID, user.Username)
}

func (s *Store) TouchMember(ctx context.Context, roomID string, user domain.User) (domain.Member, error) {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
UPDATE members SET display_name = ?, last_seen_unix_ms = ?
WHERE room_id = ? AND username = ? COLLATE NOCASE AND active = 1`,
		user.DisplayName, unixMillis(now), roomID, user.Username)
	if err != nil {
		return domain.Member{}, err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return domain.Member{}, domain.NewError(403, "membership_required", "An invitation must be redeemed before joining this room")
	}
	return s.GetMember(ctx, roomID, user.Username)
}

func (s *Store) RemoveMember(ctx context.Context, roomID, username, actor string) error {
	member, err := s.GetMember(ctx, roomID, username)
	if err != nil {
		return err
	}
	if member.Role == "owner" {
		return domain.NewError(409, "owner_cannot_be_removed", "The room owner cannot be removed")
	}
	_, err = s.db.ExecContext(ctx, `UPDATE members SET active = 0 WHERE room_id = ? AND username = ? COLLATE NOCASE`, roomID, username)
	if err == nil {
		_ = s.Audit(ctx, actor, roomID, "member.removed", map[string]string{"member": username})
	}
	return err
}

func (s *Store) LeaveRoom(ctx context.Context, roomID, username string) error {
	member, err := s.GetMember(ctx, roomID, username)
	if err != nil {
		return err
	}
	if member.Role == "owner" {
		return domain.NewError(409, "owner_cannot_leave", "The room owner must close or delete the room")
	}
	_, err = s.db.ExecContext(ctx, `UPDATE members SET active = 0 WHERE room_id = ? AND username = ? COLLATE NOCASE`, roomID, username)
	if err == nil {
		_ = s.Audit(ctx, username, roomID, "member.left", nil)
	}
	return err
}

func (s *Store) ActiveMember(ctx context.Context, roomID, username string) (domain.Member, error) {
	member, err := s.GetMember(ctx, roomID, username)
	if err != nil {
		return domain.Member{}, err
	}
	if !member.Active {
		return domain.Member{}, domain.NewError(403, "membership_required", "Room membership is inactive")
	}
	return member, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func insertAuditTx(ctx context.Context, tx *sql.Tx, username, roomID, action string, metadata any) error {
	payload := "{}"
	if metadata != nil {
		encoded, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		payload = string(encoded)
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO security_audit(occurred_unix_ms, username, room_id, action, metadata_json)
VALUES (?, ?, ?, ?, ?)`, unixMillis(time.Now().UTC()), username, roomID, action, payload)
	return err
}

func NormalizeName(value string) string { return strings.Join(strings.Fields(value), " ") }
