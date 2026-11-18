package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

const wishColumns = `wish_id, room_id, author_username, author_display_name, kind, title, artist, reason,
 status, fulfilled_by, fulfilled_by_name, response_note, resources_json, created_unix_ms, updated_unix_ms`

func scanWish(row scanner) (domain.Wish, error) {
	var wish domain.Wish
	var resources string
	var created, updated int64
	err := row.Scan(&wish.WishID, &wish.RoomID, &wish.AuthorUsername, &wish.AuthorDisplayName, &wish.Kind, &wish.Title,
		&wish.Artist, &wish.Reason, &wish.Status, &wish.FulfilledBy, &wish.FulfilledByName, &wish.ResponseNote, &resources, &created, &updated)
	if err != nil {
		return wish, err
	}
	if err = json.Unmarshal([]byte(resources), &wish.Resources); err != nil {
		return wish, err
	}
	if wish.Resources == nil {
		wish.Resources = []domain.WishResource{}
	}
	wish.CreatedAt, wish.UpdatedAt = fromUnixMillis(created), fromUnixMillis(updated)
	return wish, nil
}

func (s *Store) Wishes(ctx context.Context, roomID, status string) ([]domain.Wish, error) {
	query, args := "SELECT "+wishColumns+" FROM room_wishes WHERE room_id = ?", []any{roomID}
	if status != "all" {
		query += " AND status = ?"
		args = append(args, status)
	}
	rows, err := s.db.QueryContext(ctx, query+" ORDER BY updated_unix_ms DESC, wish_id DESC", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.Wish{}
	for rows.Next() {
		wish, err := scanWish(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, wish)
	}
	return result, rows.Err()
}

func (s *Store) AddWish(ctx context.Context, wish domain.Wish) (domain.Wish, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return wish, err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM room_wishes WHERE room_id = ? AND author_username = ? COLLATE NOCASE AND status = 'pending'", wish.RoomID, wish.AuthorUsername).Scan(&count); err != nil {
		return wish, err
	}
	if count >= 50 {
		return wish, domain.NewError(409, "wish_limit_reached", "At most fifty pending wishes are allowed per member")
	}
	wish.Status, wish.Resources = "pending", []domain.WishResource{}
	wish.CreatedAt = time.Now().UTC()
	wish.UpdatedAt = wish.CreatedAt
	_, err = tx.ExecContext(ctx, `INSERT INTO room_wishes(wish_id, room_id, author_username, author_display_name, kind, title, artist, reason, created_unix_ms, updated_unix_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		wish.WishID, wish.RoomID, wish.AuthorUsername, wish.AuthorDisplayName, wish.Kind, wish.Title, wish.Artist, wish.Reason, unixMillis(wish.CreatedAt), unixMillis(wish.UpdatedAt))
	if err != nil {
		return wish, err
	}
	return wish, tx.Commit()
}

func (s *Store) FulfillWish(ctx context.Context, roomID, wishID string, actor domain.User, note string, resources []domain.WishResource) (domain.Wish, error) {
	if resources == nil {
		resources = []domain.WishResource{}
	}
	encoded, err := json.Marshal(resources)
	if err != nil {
		return domain.Wish{}, err
	}
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	result, err := s.db.ExecContext(ctx, `UPDATE room_wishes SET status = 'fulfilled', fulfilled_by = ?, fulfilled_by_name = ?, response_note = ?, resources_json = ?, updated_unix_ms = ? WHERE room_id = ? AND wish_id = ?`, actor.Username, actor.DisplayName, note, string(encoded), unixMillis(time.Now()), roomID, wishID)
	if err != nil {
		return domain.Wish{}, err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return domain.Wish{}, domain.NewError(404, "wish_not_found", "Wish was not found")
	}
	return scanWish(s.db.QueryRowContext(ctx, "SELECT "+wishColumns+" FROM room_wishes WHERE room_id = ? AND wish_id = ?", roomID, wishID))
}

func (s *Store) DeleteWish(ctx context.Context, roomID, wishID, actor string, manager bool) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	wish, err := scanWish(tx.QueryRowContext(ctx, "SELECT "+wishColumns+" FROM room_wishes WHERE room_id = ? AND wish_id = ?", roomID, wishID))
	if isNotFound(err) {
		return domain.NewError(404, "wish_not_found", "Wish was not found")
	}
	if err != nil {
		return err
	}
	if !manager && (!domain.EqualUsername(wish.AuthorUsername, actor) || wish.Status != "pending") {
		return domain.NewError(403, "wish_author_required", "Only the author of a pending wish or a room manager can delete it")
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM room_wishes WHERE room_id = ? AND wish_id = ?", roomID, wishID); err != nil {
		return err
	}
	return tx.Commit()
}
