package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

func incrementStatsTx(ctx context.Context, tx *sql.Tx, roomID, username, displayName string, delta domain.MemberStats, now time.Time) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO member_stats
 (room_id, username, display_name, stay_seconds, requested_count, listened_count, chat_count, updated_unix_ms)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
 ON CONFLICT(room_id, username) DO UPDATE SET
 display_name = CASE WHEN excluded.display_name = '' THEN member_stats.display_name ELSE excluded.display_name END,
 stay_seconds = member_stats.stay_seconds + excluded.stay_seconds,
 requested_count = member_stats.requested_count + excluded.requested_count,
 listened_count = member_stats.listened_count + excluded.listened_count,
 chat_count = member_stats.chat_count + excluded.chat_count, updated_unix_ms = excluded.updated_unix_ms`,
		roomID, username, displayName, delta.StaySeconds, delta.RequestedCount, delta.ListenedCount, delta.ChatCount, unixMillis(now))
	return err
}

func (s *Store) IncrementStay(ctx context.Context, rooms map[string][]string, seconds float64) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	for roomID, users := range rooms {
		seen := map[string]bool{}
		for _, username := range users {
			key := strings.ToLower(username)
			if seen[key] {
				continue
			}
			seen[key] = true
			var name string
			err = tx.QueryRowContext(ctx, `SELECT m.display_name FROM members m JOIN rooms r USING(room_id)
    WHERE m.room_id = ? AND m.username = ? COLLATE NOCASE AND m.active = 1 AND r.status = 'open'`, roomID, username).Scan(&name)
			if isNotFound(err) {
				continue
			}
			if err != nil {
				return err
			}
			if err = incrementStatsTx(ctx, tx, roomID, username, name, domain.MemberStats{StaySeconds: seconds}, now); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, "UPDATE members SET last_seen_unix_ms = ? WHERE room_id = ? AND username = ? COLLATE NOCASE", unixMillis(now), roomID, username); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// Periods use UTC calendar boundaries, shared by statistics and rankings.
func periodStart(period string, now time.Time) (int64, error) {
	now = now.UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	switch period {
	case "all":
		return 0, nil
	case "year":
		return unixMillis(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)), nil
	case "month":
		return unixMillis(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)), nil
	case "week":
		return unixMillis(day.AddDate(0, 0, -(int(day.Weekday())+6)%7)), nil
	case "day":
		return unixMillis(day), nil
	default:
		return 0, domain.NewError(400, "period_invalid", "Period must be all, year, month, week, or day")
	}
}

const listenedHistoryFilter = `finished_unix_ms IS NOT NULL AND
 (played_seconds >= 30 OR (json_extract(track_json, '$.durationSeconds') > 0 AND played_seconds >= json_extract(track_json, '$.durationSeconds') * 0.5))
 AND (contributor_username = ? COLLATE NOCASE OR EXISTS (
 SELECT 1 FROM security_audit a WHERE a.room_id = h.room_id AND a.username = ? COLLATE NOCASE
 AND a.action = 'playback.listened' AND json_extract(a.metadata_json, '$.historyID') = h.history_id))`

func mediaColumns(kind string) (string, string, string) {
	id, name, artist := "json_extract(track_json, '$.id')", "json_extract(track_json, '$.title')", "COALESCE(json_extract(track_json, '$.artist'), '')"
	switch kind {
	case "artist":
		id, name = artist, artist
	case "album":
		name = "COALESCE(json_extract(track_json, '$.album'), '')"
		id = "COALESCE(NULLIF(json_extract(track_json, '$.albumID'), ''), " + artist + " || ':' || " + name + ")"
	}
	return id, name, artist
}

func (s *Store) mediaStats(ctx context.Context, roomID string, since int64, kind, username string) ([]domain.MediaStat, error) {
	id, name, artist := mediaColumns(kind)
	query := "SELECT " + id + ", " + name + ", " + artist + ", COUNT(*), COALESCE(SUM(played_seconds), 0) FROM playback_history h WHERE room_id = ? AND started_unix_ms >= ?"
	args := []any{roomID, since}
	if username != "" {
		query += " AND " + listenedHistoryFilter
		args = append(args, username, username)
	}
	query += " GROUP BY " + id + " HAVING " + name + " <> '' ORDER BY COUNT(*) DESC, " + id
	if username == "" {
		query += " LIMIT 10"
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.MediaStat{}
	for rows.Next() {
		var item domain.MediaStat
		if err := rows.Scan(&item.ID, &item.Name, &item.Artist, &item.Count, &item.Seconds); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Store) Statistics(ctx context.Context, roomID, period string) (domain.RoomStatistics, error) {
	result := domain.RoomStatistics{Period: period, HourlyDistribution: make([]domain.HourStat, 24)}
	since, err := periodStart(period, time.Now())
	if err != nil {
		return result, err
	}
	for i := range result.HourlyDistribution {
		result.HourlyDistribution[i].Hour = i
	}
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(played_seconds), 0) FROM playback_history WHERE room_id = ? AND started_unix_ms >= ?`, roomID, since).Scan(&result.TotalPlayCount, &result.TotalPlayedSeconds)
	if err != nil {
		return result, err
	}
	if err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM chat_messages WHERE room_id = ? AND created_unix_ms >= ? AND message_type <> 'system'", roomID, since).Scan(&result.TotalChatCount); err != nil {
		return result, err
	}
	if err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM members WHERE room_id = ? AND active = 1 AND last_seen_unix_ms >= ?", roomID, since).Scan(&result.ActiveMemberCount); err != nil {
		return result, err
	}
	if result.TopArtists, err = s.mediaStats(ctx, roomID, since, "artist", ""); err != nil {
		return result, err
	}
	if result.TopAlbums, err = s.mediaStats(ctx, roomID, since, "album", ""); err != nil {
		return result, err
	}
	if result.TopTracks, err = s.mediaStats(ctx, roomID, since, "track", ""); err != nil {
		return result, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT CAST(strftime('%H', started_unix_ms / 1000, 'unixepoch') AS INTEGER), COUNT(*), SUM(played_seconds)
 FROM playback_history WHERE room_id = ? AND started_unix_ms >= ? GROUP BY 1`, roomID, since)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var hour domain.HourStat
		if err = rows.Scan(&hour.Hour, &hour.Count, &hour.Seconds); err != nil {
			return result, err
		}
		result.HourlyDistribution[hour.Hour] = hour
	}
	return result, rows.Err()
}

func (s *Store) Rankings(ctx context.Context, roomID, period string) (domain.RoomRankings, error) {
	result := domain.RoomRankings{Period: period}
	since, err := periodStart(period, time.Now())
	if err != nil {
		return result, err
	}
	for i, target := range []*[]domain.MemberRank{&result.Requested, &result.Listened, &result.Chat} {
		*target = []domain.MemberRank{}
		columns := []string{"requested_count", "listened_count", "chat_count"}
		query := "SELECT s.username, COALESCE(m.display_name, s.display_name), s." + columns[i] + " FROM member_stats s LEFT JOIN members m ON m.room_id = s.room_id AND m.username = s.username WHERE s.room_id = ? AND s." + columns[i] + " > 0"
		args := []any{roomID}
		if period != "all" {
			if i == 2 {
				query = `SELECT c.sender_username, COALESCE(m.display_name, c.sender_display_name), COUNT(*) FROM chat_messages c LEFT JOIN members m ON m.room_id = c.room_id AND m.username = c.sender_username WHERE c.room_id = ? AND c.created_unix_ms >= ? AND c.message_type <> 'system' GROUP BY c.sender_username COLLATE NOCASE`
				args = append(args, since)
			} else {
				query = `SELECT a.username, COALESCE(m.display_name, a.username), COUNT(*) FROM security_audit a LEFT JOIN members m ON m.room_id = a.room_id AND m.username = a.username COLLATE NOCASE WHERE a.room_id = ? AND a.occurred_unix_ms >= ? AND a.action = ? GROUP BY a.username COLLATE NOCASE`
				args = append(args, since, []string{"queue.added", "playback.listened"}[i])
			}
		}
		rows, err := s.db.QueryContext(ctx, query+" ORDER BY 3 DESC, 1 COLLATE NOCASE LIMIT 50", args...)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			var rank domain.MemberRank
			if err = rows.Scan(&rank.Username, &rank.DisplayName, &rank.Count); err != nil {
				rows.Close()
				return result, err
			}
			*target = append(*target, rank)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return result, err
		}
	}
	if result.TopArtists, err = s.mediaStats(ctx, roomID, since, "artist", ""); err != nil {
		return result, err
	}
	if result.TopAlbums, err = s.mediaStats(ctx, roomID, since, "album", ""); err != nil {
		return result, err
	}
	result.TopTracks, err = s.mediaStats(ctx, roomID, since, "track", "")
	return result, err
}

func memberLevel(stats domain.MemberStats) domain.MemberLevel {
	thresholds := []float64{0, 60, 180, 420, 900, 1800, 3600, 7200, 14400, 28800}
	minutes := stats.StaySeconds/60 + float64(stats.ListenedCount*5+stats.RequestedCount*3+stats.ChatCount)
	index := 0
	for index+1 < len(thresholds) && minutes >= thresholds[index+1] {
		index++
	}
	level := domain.MemberLevel{Level: index + 1, Code: fmt.Sprintf("level_%d", index+1), EffectiveMinutes: minutes,
		CurrentThresholdMinutes: thresholds[index], NextThresholdMinutes: thresholds[index], Progress: 1}
	if index+1 < len(thresholds) {
		level.NextThresholdMinutes = thresholds[index+1]
		level.Progress = math.Max(0, math.Min(1, (minutes-thresholds[index])/(thresholds[index+1]-thresholds[index])))
	}
	return level
}

func (s *Store) MemberProfile(ctx context.Context, roomID, username string) (domain.MemberProfile, error) {
	profile := domain.MemberProfile{Badges: []domain.Badge{}}
	member, err := s.GetMember(ctx, roomID, username)
	if roomErr, ok := err.(*domain.Error); ok && roomErr.Code == "membership_required" {
		return profile, domain.NewError(404, "member_not_found", "Room member was not found")
	}
	if err != nil {
		return profile, err
	}
	profile.Username, profile.DisplayName, profile.Role = member.Username, member.DisplayName, member.Role
	profile.JoinedAt, profile.LastSeenAt = member.JoinedAt, member.LastSeenAt
	err = s.db.QueryRowContext(ctx, `SELECT stay_seconds, requested_count, listened_count, chat_count, selected_badge_id FROM member_stats WHERE room_id = ? AND username = ? COLLATE NOCASE`, roomID, username).Scan(&profile.Stats.StaySeconds, &profile.Stats.RequestedCount, &profile.Stats.ListenedCount, &profile.Stats.ChatCount, &profile.SelectedBadgeID)
	if err != nil && !isNotFound(err) {
		return profile, err
	}
	profile.Level = memberLevel(profile.Stats)
	addBadge := func(id, code, kind, subject string, tier, count, target int, subjectID string) error {
		badge := domain.Badge{BadgeID: id, Code: code, Kind: kind, Subject: subject, Tier: tier, Count: count, Target: target, Unlocked: count >= target}
		if badge.Unlocked {
			var err error
			badge.UnlockedAt, err = s.badgeUnlockTime(ctx, roomID, username, kind, subjectID, target)
			if err != nil {
				return err
			}
		}
		profile.Badges = append(profile.Badges, badge)
		return nil
	}
	for _, item := range []struct {
		kind    string
		count   int
		targets []int
	}{
		{"requested", profile.Stats.RequestedCount, []int{1, 10, 50}},
		{"listened", profile.Stats.ListenedCount, []int{10, 50, 200}},
		{"chat", profile.Stats.ChatCount, []int{10, 50}},
	} {
		for tier, target := range item.targets {
			id := fmt.Sprintf("%s_%d", item.kind, target)
			if err := addBadge(id, id, item.kind, "", tier+1, item.count, target, ""); err != nil {
				return profile, err
			}
		}
	}
	for _, kind := range []string{"artist", "album"} {
		items, err := s.mediaStats(ctx, roomID, 0, kind, username)
		if err != nil {
			return profile, err
		}
		for _, item := range items {
			targets := []int{10, 25, 50}
			if kind == "album" {
				targets = []int{6}
			}
			digest := sha256.Sum256([]byte(item.ID))
			for tier, target := range targets {
				if err := addBadge(fmt.Sprintf("%s_%x_%d", kind, digest[:8], target), fmt.Sprintf("%s_%d", kind, target), kind, item.Name, tier+1, item.Count, target, item.ID); err != nil {
					return profile, err
				}
			}
		}
	}
	return profile, nil
}

func (s *Store) SelectBadge(ctx context.Context, roomID, username, badgeID string) (domain.MemberProfile, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	profile, err := s.MemberProfile(ctx, roomID, username)
	if err != nil {
		return profile, err
	}
	allowed := badgeID == ""
	for _, badge := range profile.Badges {
		if badge.BadgeID == badgeID && badge.Unlocked {
			allowed = true
		}
	}
	if !allowed {
		return profile, domain.NewError(400, "badge_locked", "Select an unlocked badge or clear the selection")
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO member_stats(room_id, username, display_name, selected_badge_id, updated_unix_ms) VALUES (?, ?, ?, ?, ?)
 ON CONFLICT(room_id, username) DO UPDATE SET selected_badge_id = excluded.selected_badge_id`, roomID, username, profile.DisplayName, badgeID, unixMillis(time.Now()))
	profile.SelectedBadgeID = badgeID
	return profile, err
}

func (s *Store) badgeUnlockTime(ctx context.Context, roomID, username, kind, subjectID string, target int) (*time.Time, error) {
	query := "SELECT occurred_unix_ms FROM security_audit WHERE room_id = ? AND username = ? COLLATE NOCASE AND action = ? ORDER BY occurred_unix_ms, audit_id LIMIT 1 OFFSET ?"
	args := []any{roomID, username, "queue.added", target - 1}
	switch kind {
	case "listened":
		args[2] = "playback.listened"
	case "chat":
		query = "SELECT created_unix_ms FROM chat_messages WHERE room_id = ? AND sender_username = ? COLLATE NOCASE AND message_type <> 'system' ORDER BY created_unix_ms, message_id LIMIT 1 OFFSET ?"
		args = []any{roomID, username, target - 1}
	case "artist", "album":
		id, _, _ := mediaColumns(kind)
		query = "SELECT finished_unix_ms FROM playback_history h WHERE room_id = ? AND " + id + " = ? AND " + listenedHistoryFilter + " ORDER BY finished_unix_ms, history_id LIMIT 1 OFFSET ?"
		args = []any{roomID, subjectID, username, username, target - 1}
	}
	var timestamp int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&timestamp)
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return timePointer(fromUnixMillis(timestamp)), nil
}
