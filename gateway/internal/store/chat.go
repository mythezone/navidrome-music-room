package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

const chatColumns = `message_id, room_id, sender_username, sender_display_name, message_type,
 body, sticker_id, reply_to_id, mentions_json, references_json, recalled_unix_ms, deleted_unix_ms, created_unix_ms`

func scanChat(row scanner) (domain.ChatMessage, error) {
	var msg domain.ChatMessage
	var mentions, references string
	var recalled, deleted sql.NullInt64
	var created int64
	err := row.Scan(&msg.MessageID, &msg.RoomID, &msg.SenderUsername, &msg.SenderDisplayName, &msg.MessageType,
		&msg.Body, &msg.StickerID, &msg.ReplyToID, &mentions, &references, &recalled, &deleted, &created)
	if err != nil {
		return msg, err
	}
	if err = json.Unmarshal([]byte(mentions), &msg.Mentions); err != nil {
		return msg, err
	}
	if err = json.Unmarshal([]byte(references), &msg.References); err != nil {
		return msg, err
	}
	if msg.Mentions == nil {
		msg.Mentions = []string{}
	}
	if msg.References == nil {
		msg.References = []domain.ChatReference{}
	}
	msg.CreatedAt = fromUnixMillis(created)
	msg.RecalledAt, msg.DeletedAt = nullableTime(recalled), nullableTime(deleted)
	if msg.RecalledAt != nil || msg.DeletedAt != nil {
		msg.Body, msg.StickerID = "", ""
		msg.Mentions, msg.References = []string{}, []domain.ChatReference{}
	}
	return msg, nil
}

func chatQuery(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, roomID, messageID string) (domain.ChatMessage, error) {
	msg, err := scanChat(q.QueryRowContext(ctx, "SELECT "+chatColumns+" FROM chat_messages WHERE room_id = ? AND message_id = ?", roomID, messageID))
	if isNotFound(err) {
		return msg, domain.NewError(404, "message_not_found", "Chat message was not found")
	}
	return msg, err
}

func (s *Store) chatReply(ctx context.Context, msg *domain.ChatMessage) error {
	if msg.ReplyToID == "" {
		return nil
	}
	reply, err := chatQuery(ctx, s.db, msg.RoomID, msg.ReplyToID)
	if err != nil {
		return err
	}
	snippet := []rune(reply.Body)
	if len(snippet) > 160 {
		snippet = snippet[:160]
	}
	msg.ReplyPreview = &domain.ChatReplyPreview{
		MessageID: reply.MessageID, SenderUsername: reply.SenderUsername, SenderDisplayName: reply.SenderDisplayName,
		Body: string(snippet), StickerID: reply.StickerID, RecalledAt: reply.RecalledAt, DeletedAt: reply.DeletedAt,
	}
	return nil
}

func (s *Store) Chat(ctx context.Context, roomID, before string, limit int) (domain.ChatPage, error) {
	page := domain.ChatPage{Messages: []domain.ChatMessage{}}
	query := "SELECT " + chatColumns + " FROM chat_messages WHERE room_id = ?"
	args := []any{roomID}
	if before != "" {
		cursor, err := chatQuery(ctx, s.db, roomID, before)
		if err != nil {
			return page, err
		}
		query += " AND (created_unix_ms, message_id) < (?, ?)"
		args = append(args, unixMillis(cursor.CreatedAt), cursor.MessageID)
	}
	query += " ORDER BY created_unix_ms DESC, message_id DESC LIMIT ?"
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		msg, err := scanChat(rows)
		if err != nil {
			return page, err
		}
		page.Messages = append(page.Messages, msg)
	}
	if err = rows.Err(); err != nil {
		return page, err
	}
	if err = rows.Close(); err != nil {
		return page, err
	}
	page.HasMore = len(page.Messages) > limit
	if page.HasMore {
		page.Messages = page.Messages[:limit]
	}
	slices.Reverse(page.Messages)
	if page.HasMore {
		page.NextBefore = page.Messages[0].MessageID
	}
	for i := range page.Messages {
		if err = s.chatReply(ctx, &page.Messages[i]); err != nil {
			return page, err
		}
	}
	return page, nil
}

func (s *Store) AddChat(ctx context.Context, msg domain.ChatMessage) (domain.ChatMessage, error) {
	if msg.Mentions == nil {
		msg.Mentions = []string{}
	}
	if msg.References == nil {
		msg.References = []domain.ChatReference{}
	}
	mentions, err := json.Marshal(msg.Mentions)
	if err != nil {
		return msg, err
	}
	references, err := json.Marshal(msg.References)
	if err != nil {
		return msg, err
	}
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return msg, err
	}
	defer tx.Rollback()
	if msg.ReplyToID != "" {
		if _, err = chatQuery(ctx, tx, msg.RoomID, msg.ReplyToID); err != nil {
			return msg, err
		}
	}
	msg.CreatedAt = time.Now().UTC()
	_, err = tx.ExecContext(ctx, `INSERT INTO chat_messages (`+chatColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, ?)`,
		msg.MessageID, msg.RoomID, msg.SenderUsername, msg.SenderDisplayName, msg.MessageType,
		msg.Body, msg.StickerID, msg.ReplyToID, string(mentions), string(references), unixMillis(msg.CreatedAt))
	if err != nil {
		return msg, err
	}
	if err = incrementStatsTx(ctx, tx, msg.RoomID, msg.SenderUsername, msg.SenderDisplayName, domain.MemberStats{ChatCount: 1}, msg.CreatedAt); err != nil {
		return msg, err
	}
	if err = tx.Commit(); err != nil {
		return msg, err
	}
	err = s.chatReply(ctx, &msg)
	return msg, err
}

func (s *Store) UpdateChat(ctx context.Context, roomID, messageID, actor string, manager, recall bool) (domain.ChatMessage, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ChatMessage{}, err
	}
	defer tx.Rollback()
	msg, err := chatQuery(ctx, tx, roomID, messageID)
	if err != nil {
		return msg, err
	}
	own := domain.EqualUsername(msg.SenderUsername, actor)
	now := time.Now().UTC()
	if recall {
		if !own {
			return msg, domain.NewError(403, "message_sender_required", "Only the sender can recall a message")
		}
		if msg.DeletedAt != nil || now.Sub(msg.CreatedAt) > 5*time.Minute {
			return msg, domain.NewError(409, "recall_expired", "Messages can only be recalled within five minutes")
		}
	} else if !own && !manager {
		return msg, domain.NewError(403, "message_sender_required", "Only the sender or a room manager can delete a message")
	}
	column := "deleted_unix_ms"
	if recall {
		column = "recalled_unix_ms"
	}
	_, err = tx.ExecContext(ctx, "UPDATE chat_messages SET "+column+" = COALESCE("+column+", ?), body = '', sticker_id = '', mentions_json = '[]', references_json = '[]' WHERE room_id = ? AND message_id = ?", unixMillis(now), roomID, messageID)
	if err != nil {
		return msg, err
	}
	msg, err = chatQuery(ctx, tx, roomID, messageID)
	if err != nil {
		return msg, err
	}
	if err = tx.Commit(); err != nil {
		return msg, err
	}
	err = s.chatReply(ctx, &msg)
	return msg, err
}
