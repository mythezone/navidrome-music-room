package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

func (s *Store) CreateInvite(ctx context.Context, invite domain.Invite, tokenHash string) (domain.Invite, error) {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO invites(invite_id, room_id, token_hash, label, expires_unix_ms, max_uses, use_count, created_by, created_unix_ms)
VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		invite.InviteID, invite.RoomID, tokenHash, invite.Label, unixMillis(invite.ExpiresAt),
		invite.MaxUses, invite.CreatedBy, unixMillis(invite.CreatedAt))
	if err != nil {
		return domain.Invite{}, err
	}
	_ = s.Audit(ctx, invite.CreatedBy, invite.RoomID, "invite.created", map[string]any{
		"inviteID": invite.InviteID, "expiresAt": invite.ExpiresAt, "maxUses": invite.MaxUses,
	})
	return invite, nil
}

func scanInvite(row scanner) (domain.Invite, error) {
	var invite domain.Invite
	var expires, created int64
	var revoked sql.NullInt64
	err := row.Scan(
		&invite.InviteID, &invite.RoomID, &invite.Label, &expires, &invite.MaxUses,
		&invite.UseCount, &revoked, &invite.CreatedBy, &created,
	)
	if err != nil {
		return domain.Invite{}, err
	}
	invite.ExpiresAt = fromUnixMillis(expires)
	invite.CreatedAt = fromUnixMillis(created)
	invite.RevokedAt = nullableTime(revoked)
	return invite, nil
}

func (s *Store) ListInvites(ctx context.Context, roomID string) ([]domain.Invite, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT invite_id, room_id, label, expires_unix_ms, max_uses, use_count, revoked_unix_ms, created_by, created_unix_ms
FROM invites WHERE room_id = ? ORDER BY created_unix_ms DESC`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Invite
	for rows.Next() {
		invite, err := scanInvite(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, invite)
	}
	return result, rows.Err()
}

func (s *Store) RevokeInvite(ctx context.Context, roomID, inviteID, actor string) error {
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
UPDATE invites SET revoked_unix_ms = COALESCE(revoked_unix_ms, ?)
WHERE room_id = ? AND invite_id = ?`, unixMillis(now), roomID, inviteID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return domain.NewError(404, "invite_not_found", "Invitation was not found")
	}
	_ = s.Audit(ctx, actor, roomID, "invite.revoked", map[string]string{"inviteID": inviteID})
	return nil
}

func (s *Store) RedeemInvite(ctx context.Context, roomID, tokenHash string, user domain.User) (domain.Member, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Member{}, err
	}
	var inviteID string
	var expires int64
	var maxUses, useCount int
	var revoked sql.NullInt64
	err = tx.QueryRowContext(ctx, `
SELECT invite_id, expires_unix_ms, max_uses, use_count, revoked_unix_ms
FROM invites WHERE room_id = ? AND token_hash = ?`, roomID, tokenHash).Scan(
		&inviteID, &expires, &maxUses, &useCount, &revoked,
	)
	if isNotFound(err) {
		_ = tx.Rollback()
		return domain.Member{}, domain.NewError(404, "invite_invalid", "Invitation is invalid")
	}
	if err != nil {
		_ = tx.Rollback()
		return domain.Member{}, err
	}
	now := time.Now().UTC()
	if revoked.Valid {
		_ = tx.Rollback()
		return domain.Member{}, domain.NewError(410, "invite_revoked", "Invitation has been revoked")
	}
	if !fromUnixMillis(expires).After(now) {
		_ = tx.Rollback()
		return domain.Member{}, domain.NewError(410, "invite_expired", "Invitation has expired")
	}
	if useCount >= maxUses {
		_ = tx.Rollback()
		return domain.Member{}, domain.NewError(410, "invite_exhausted", "Invitation has reached its use limit")
	}

	var active int
	err = tx.QueryRowContext(ctx, `
SELECT active FROM members WHERE room_id = ? AND username = ? COLLATE NOCASE`, roomID, user.Username).Scan(&active)
	if err == nil && active != 0 {
		if _, err = tx.ExecContext(ctx, `
UPDATE members SET display_name = ?, last_seen_unix_ms = ? WHERE room_id = ? AND username = ? COLLATE NOCASE`,
			user.DisplayName, unixMillis(now), roomID, user.Username); err != nil {
			_ = tx.Rollback()
			return domain.Member{}, err
		}
		if err = tx.Commit(); err != nil {
			return domain.Member{}, err
		}
		return s.GetMember(ctx, roomID, user.Username)
	}
	if err != nil && !isNotFound(err) {
		_ = tx.Rollback()
		return domain.Member{}, err
	}
	if _, err = tx.ExecContext(ctx, `
UPDATE invites SET use_count = use_count + 1
WHERE invite_id = ? AND use_count < max_uses AND revoked_unix_ms IS NULL AND expires_unix_ms > ?`,
		inviteID, unixMillis(now)); err != nil {
		_ = tx.Rollback()
		return domain.Member{}, err
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO members(room_id, username, display_name, role, active, joined_unix_ms, last_seen_unix_ms)
VALUES (?, ?, ?, 'member', 1, ?, ?)
ON CONFLICT(room_id, username) DO UPDATE SET
    display_name = excluded.display_name, active = 1, last_seen_unix_ms = excluded.last_seen_unix_ms`,
		roomID, user.Username, user.DisplayName, unixMillis(now), unixMillis(now)); err != nil {
		_ = tx.Rollback()
		return domain.Member{}, err
	}
	if err = insertAuditTx(ctx, tx, user.Username, roomID, "invite.redeemed", map[string]string{"inviteID": inviteID}); err != nil {
		_ = tx.Rollback()
		return domain.Member{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.Member{}, err
	}
	return s.GetMember(ctx, roomID, user.Username)
}
