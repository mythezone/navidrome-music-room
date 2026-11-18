package store

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

func openTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	storage, err := Open(t.Context(), filepath.Join(dir, "rooms.sqlite3"), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = storage.Close() })
	return storage, dir
}

func createTestRoom(t *testing.T, storage *Store) domain.Room {
	t.Helper()
	room, err := storage.CreateRoom(t.Context(), domain.Room{
		RoomID: "aaaaaaaaaaaaaaaa", Name: "Test Room", OwnerUsername: "admin", OwnerDisplayName: "Admin",
		QueueLimit: 2, PlaybackMode: domain.QueueFIFO, MusicFolderIDs: []int{1}, PreloadNextTrack: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return room
}

func TestInviteRedemptionIsPersistentAndIdempotent(t *testing.T) {
	storage, _ := openTestStore(t)
	room := createTestRoom(t, storage)
	token := "a-secret-invitation-token-with-enough-entropy"
	invite := domain.Invite{
		InviteID: "bbbbbbbbbbbbbbbb", RoomID: room.RoomID, ExpiresAt: time.Now().Add(time.Hour),
		MaxUses: 2, CreatedBy: "admin", CreatedAt: time.Now().UTC(),
	}
	if _, err := storage.CreateInvite(t.Context(), invite, domain.TokenHash(token)); err != nil {
		t.Fatal(err)
	}
	user := domain.User{Username: "member", DisplayName: "Member", MusicFolderIDs: []int{1}}
	for range 2 {
		if _, err := storage.RedeemInvite(t.Context(), room.RoomID, domain.TokenHash(token), user); err != nil {
			t.Fatal(err)
		}
	}
	invites, err := storage.ListInvites(t.Context(), room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	if invites[0].UseCount != 1 {
		t.Fatalf("repeat redemption should not consume another use, got %d", invites[0].UseCount)
	}
	if member, err := storage.ActiveMember(t.Context(), room.RoomID, "MEMBER"); err != nil || !member.Active {
		t.Fatalf("persistent case-insensitive membership missing: %#v %v", member, err)
	}
}

func TestPlaybackRevisionQueueAndEmptyPause(t *testing.T) {
	storage, _ := openTestStore(t)
	room := createTestRoom(t, storage)
	for index, contributor := range []string{"alice", "bob"} {
		id := []string{"cccccccccccccccc", "dddddddddddddddd"}[index]
		_, primed, err := storage.AddQueueEntry(t.Context(), room, domain.QueueEntry{
			QueueID: id, RoomID: room.RoomID, Contributor: contributor, ContributorName: contributor,
			Track: domain.NavidromeTrackRef{ID: "track-" + contributor, MusicFolderID: 1, Title: contributor, DurationSeconds: 100},
		})
		if err != nil {
			t.Fatal(err)
		}
		if index == 0 && (primed == nil || primed.Revision != 1 || primed.Status != domain.PlaybackPaused || primed.CurrentTrack == nil) {
			t.Fatalf("first queue item was not promoted to paused playback: %#v", primed)
		}
		if index == 1 && primed != nil {
			t.Fatalf("adding to an active room unexpectedly replaced playback: %#v", primed)
		}
	}
	beforePlay, err := storage.Playback(t.Context(), room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	if beforePlay.Revision != 1 || beforePlay.Status != domain.PlaybackPaused || beforePlay.CurrentTrack == nil || beforePlay.CurrentTrack.ID != "track-alice" || beforePlay.NextTrack == nil || beforePlay.NextTrack.ID != "track-bob" {
		t.Fatalf("queue priming did not preserve current/next ordering: %#v", beforePlay)
	}
	if history, err := storage.History(t.Context(), room.RoomID, 10, 0); err != nil || len(history) != 0 {
		t.Fatalf("a paused primed track must not enter history before playback: %#v err=%v", history, err)
	}
	state, err := storage.ApplyPlayback(t.Context(), room.RoomID, "play", nil, 1, "admin", nil)
	if err != nil {
		t.Fatal(err)
	}
	if state.Revision != 2 || state.Status != domain.PlaybackPlaying || state.CurrentTrack == nil {
		t.Fatalf("unexpected playback state: %#v", state)
	}
	if history, err := storage.History(t.Context(), room.RoomID, 10, 0); err != nil || len(history) != 1 || history[0].Track.ID != "track-alice" {
		t.Fatalf("playing the primed track did not create history: %#v err=%v", history, err)
	}
	if _, err := storage.ApplyPlayback(t.Context(), room.RoomID, "pause", nil, 1, "admin", nil); err == nil {
		t.Fatal("expected stale revision conflict")
	} else if roomErr, ok := err.(*domain.Error); !ok || roomErr.Code != "revision_conflict" {
		t.Fatalf("expected revision_conflict, got %T %v", err, err)
	}
	paused, changed, err := storage.PauseForEmpty(t.Context(), room.RoomID)
	if err != nil || !changed || paused.Status != domain.PlaybackPaused || !paused.PausedForEmpty {
		t.Fatalf("empty room pause failed: %#v changed=%v err=%v", paused, changed, err)
	}
}

func TestPrimePlaybackIfIdleRepairsLegacyPendingQueue(t *testing.T) {
	storage, _ := openTestStore(t)
	room := createTestRoom(t, storage)
	trackJSON, err := json.Marshal(domain.NavidromeTrackRef{
		ID: "legacy-track", MusicFolderID: 1, Title: "Legacy", DurationSeconds: 120,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := storage.db.ExecContext(t.Context(), `
INSERT INTO queue(queue_id, room_id, position, track_json, contributor_username, contributor_display_name, created_unix_ms)
VALUES (?, ?, 1, ?, ?, ?, ?)`, "eeeeeeeeeeeeeeee", room.RoomID, string(trackJSON), "member", "Member", unixMillis(time.Now().UTC())); err != nil {
		t.Fatal(err)
	}

	state, changed, err := storage.PrimePlaybackIfIdle(t.Context(), room.RoomID, "__test_recovery__")
	if err != nil || !changed || state.Revision != 1 || state.Status != domain.PlaybackPaused || state.CurrentTrack == nil || state.CurrentTrack.ID != "legacy-track" {
		t.Fatalf("legacy queue recovery failed: state=%#v changed=%v err=%v", state, changed, err)
	}
	if queue, err := storage.ListQueue(t.Context(), room.RoomID); err != nil || len(queue) != 0 {
		t.Fatalf("recovered current item remained pending: %#v err=%v", queue, err)
	}
	state, changed, err = storage.PrimePlaybackIfIdle(t.Context(), room.RoomID, "__test_recovery__")
	if err != nil || changed || state.Revision != 1 {
		t.Fatalf("idle recovery was not idempotent: state=%#v changed=%v err=%v", state, changed, err)
	}
}

func TestBackupAndRestartPersistence(t *testing.T) {
	storage, dir := openTestStore(t)
	room := createTestRoom(t, storage)
	backup, err := storage.Backup(t.Context(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(backup); err != nil || info.Size() == 0 || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("backup should exist with private permissions: info=%v err=%v", info, err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(t.Context(), filepath.Join(dir, "rooms.sqlite3"), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if persisted, err := reopened.GetRoom(t.Context(), room.RoomID); err != nil || persisted.Name != room.Name {
		t.Fatalf("room did not survive restart: %#v %v", persisted, err)
	}
}

func TestPluginStatePersistsUpdateChannelAndRejectsStaleGeneration(t *testing.T) {
	storage, _ := openTestStore(t)
	receivedAt := time.Now().UTC().Truncate(time.Millisecond)
	input := domain.PluginSync{
		PluginVersion:      "v1.2.3",
		Generation:         8,
		NavidromePublicURL: "https://music.example.test",
		GatewayPublicURL:   "https://rooms.example.test",
		UpdateChannel:      "beta",
		Users:              []domain.PluginUser{{Username: "admin", DisplayName: "Admin", Admin: true}},
	}
	if _, err := storage.SavePluginSync(t.Context(), input, receivedAt); err != nil {
		t.Fatal(err)
	}
	state, err := storage.PluginState(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if state.PluginVersion != input.PluginVersion || state.UpdateChannel != "beta" || len(state.Users) != 1 || !state.Users[0].Admin {
		t.Fatalf("plugin state was not preserved: %#v", state)
	}
	input.Generation = 7
	if _, err := storage.SavePluginSync(t.Context(), input, receivedAt.Add(time.Second)); err == nil {
		t.Fatal("stale plugin generation was accepted")
	} else if roomError, ok := err.(*domain.Error); !ok || roomError.Code != "stale_plugin_generation" {
		t.Fatalf("unexpected stale generation error: %T %v", err, err)
	}
}

func TestDiagnosticsContainOnlyAggregateDatabaseHealth(t *testing.T) {
	storage, _ := openTestStore(t)
	createTestRoom(t, storage)
	summary, err := storage.Diagnostics(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if summary.QuickCheck != "ok" || summary.SchemaVersion != len(migrations) {
		t.Fatalf("unexpected database health: %#v", summary)
	}
	if summary.TableCounts["rooms"] != 1 || summary.TableCounts["members"] != 1 {
		t.Fatalf("aggregate counts are wrong: %#v", summary.TableCounts)
	}
	if summary.PageCount <= 0 || summary.PageSize <= 0 {
		t.Fatalf("database size metadata is missing: %#v", summary)
	}
}

func TestCollaborativeMigrationPreservesExistingRoom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rooms.sqlite3")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, name TEXT, applied_unix_ms INTEGER)`); err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations[:4] {
		if _, err = db.Exec(migration.sql); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec("INSERT INTO schema_migrations VALUES (?, ?, 0)", migration.version, migration.name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec(`INSERT INTO rooms(room_id,name,owner_username,owner_display_name,status,queue_limit,playback_mode,music_folder_ids_json,created_unix_ms,updated_unix_ms) VALUES ('legacy-room-12345','Legacy','owner','Owner','open',20,'fifo','[1]',0,0)`); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	storage, err := Open(t.Context(), path, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	room, err := storage.GetRoom(t.Context(), "legacy-room-12345")
	if err != nil || !room.AllowMemberSkip || room.Name != "Legacy" {
		t.Fatalf("migration lost room/default: %+v %v", room, err)
	}
	room.AllowMemberSkip = false
	if _, err = storage.UpdateRoom(t.Context(), room, "owner"); err != nil {
		t.Fatal(err)
	}
	room, err = storage.GetRoom(t.Context(), room.RoomID)
	if err != nil || room.AllowMemberSkip {
		t.Fatalf("skip setting did not persist: %+v %v", room, err)
	}
	summary, err := storage.Diagnostics(t.Context())
	if err != nil || summary.SchemaVersion != 5 {
		t.Fatalf("migration version: %+v %v", summary, err)
	}
	for _, table := range []string{"chat_messages", "room_wishes", "member_stats"} {
		if _, ok := summary.TableCounts[table]; !ok {
			t.Fatalf("missing diagnostic table %s", table)
		}
	}
	backups, err := filepath.Glob(filepath.Join(dir, "backups", "*.sqlite3"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected pre-migration backup: %v %v", backups, err)
	}
}

func TestPreviousPlayNextAndListeningAccounting(t *testing.T) {
	storage, _ := openTestStore(t)
	room := createTestRoom(t, storage)
	room.QueueLimit = 20
	if _, err := storage.UpdateRoom(t.Context(), room, "admin"); err != nil {
		t.Fatal(err)
	}
	for _, username := range []string{"alice", "bob"} {
		if _, err := storage.UpsertMember(t.Context(), room.RoomID, domain.User{Username: username, DisplayName: username}, "member", "admin", "member.joined"); err != nil {
			t.Fatal(err)
		}
	}
	add := func(title string, position int) {
		t.Helper()
		id, _ := domain.NewID()
		if _, _, err := storage.AddQueueEntry(t.Context(), room, domain.QueueEntry{QueueID: id, Position: position, Contributor: "alice", ContributorName: "Alice", Track: domain.NavidromeTrackRef{ID: title, Title: title, Artist: "Artist", Album: "Album", DurationSeconds: 100, MusicFolderID: 1}}); err != nil {
			t.Fatal(err)
		}
	}
	apply := func(command string, position *float64) domain.PlaybackState {
		t.Helper()
		state, err := storage.Playback(t.Context(), room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		state, err = storage.ApplyPlayback(t.Context(), room.RoomID, command, position, state.Revision, "admin", map[string]bool{"ALICE": true, "alice": true, "bob": true})
		if err != nil {
			t.Fatal(err)
		}
		return state
	}
	add("first", 0)
	add("second", 0)
	add("third", 0)
	add("next", 1)
	queue, err := storage.ListQueue(t.Context(), room.RoomID)
	if err != nil || len(queue) != 3 || queue[0].Track.ID != "next" || queue[1].Position != 2 || queue[2].Track.ID != "third" {
		t.Fatalf("playNext failed: %+v %v", queue, err)
	}
	apply("play", nil)
	position := 40.0
	apply("seek", &position)
	state := apply("previous", nil)
	if state.CurrentTrack.ID != "first" || state.PositionSeconds > 1 || state.Status != domain.PlaybackPlaying || state.AnchorServerTime == nil {
		t.Fatalf("previous should restart: %+v", state)
	}
	apply("seek", &position)
	state = apply("next", nil)
	if state.CurrentTrack.ID != "next" {
		t.Fatalf("wrong next: %+v", state)
	}
	apply("pause", nil)
	state = apply("previous", nil)
	if state.CurrentTrack.ID != "first" || state.Status != domain.PlaybackPaused || state.AnchorServerTime != nil {
		t.Fatalf("previous failed: %+v", state)
	}
	queue, err = storage.ListQueue(t.Context(), room.RoomID)
	if err != nil || len(queue) != 3 || queue[0].Track.ID != "next" || queue[1].Track.ID != "second" {
		t.Fatalf("previous lost queue: %+v %v", queue, err)
	}
	history, err := storage.History(t.Context(), room.RoomID, 20, 0)
	if err != nil || len(history) != 3 {
		t.Fatalf("previous destroyed history: %+v %v", history, err)
	}
	alice, err := storage.MemberProfile(t.Context(), room.RoomID, "alice")
	if err != nil || alice.Stats.RequestedCount != 4 || alice.Stats.ListenedCount != 1 {
		t.Fatalf("contributor counted twice: %+v %v", alice, err)
	}
	bob, err := storage.MemberProfile(t.Context(), room.RoomID, "bob")
	if err != nil || bob.Stats.ListenedCount != 1 {
		t.Fatalf("listener not counted: %+v %v", bob, err)
	}
	if err = storage.IncrementStay(t.Context(), map[string][]string{room.RoomID: {"bob", "BOB", "unknown"}}, 15); err != nil {
		t.Fatal(err)
	}
	bob, err = storage.MemberProfile(t.Context(), room.RoomID, "bob")
	if err != nil || bob.Stats.StaySeconds != 15 {
		t.Fatalf("stay dedup: %+v %v", bob, err)
	}
	rankings, err := storage.Rankings(t.Context(), room.RoomID, "day")
	if err != nil || len(rankings.Listened) != 2 || rankings.Requested[0].Count != 4 {
		t.Fatalf("period rankings: %+v %v", rankings, err)
	}
	statistics, err := storage.Statistics(t.Context(), room.RoomID, "day")
	if err != nil || statistics.TotalPlayCount != 3 || statistics.TotalPlayedSeconds < 40 || len(statistics.HourlyDistribution) != 24 {
		t.Fatalf("statistics: %+v %v", statistics, err)
	}
	if _, err = storage.SelectBadge(t.Context(), room.RoomID, "alice", "listened_200"); err == nil {
		t.Fatal("selected locked badge")
	}
	alice, err = storage.SelectBadge(t.Context(), room.RoomID, "alice", "requested_1")
	if err != nil || alice.SelectedBadgeID != "requested_1" {
		t.Fatalf("select unlocked badge: %+v %v", alice, err)
	}
}

func TestChatPaginationRecallAndDeletion(t *testing.T) {
	storage, _ := openTestStore(t)
	room := createTestRoom(t, storage)
	ids := []string{"aaaaaaaaaaaaaaa1", "aaaaaaaaaaaaaaa2", "aaaaaaaaaaaaaaa3"}
	for _, id := range ids {
		_, err := storage.AddChat(t.Context(), domain.ChatMessage{MessageID: id, RoomID: room.RoomID, SenderUsername: "admin", SenderDisplayName: "Admin", MessageType: "chat", Body: "hello"})
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := storage.db.Exec("UPDATE chat_messages SET created_unix_ms = ?", unixMillis(time.Now())); err != nil {
		t.Fatal(err)
	}
	page, err := storage.Chat(t.Context(), room.RoomID, "", 2)
	if err != nil || !page.HasMore || page.NextBefore != ids[1] || page.Messages[0].MessageID != ids[1] || page.Messages[1].MessageID != ids[2] {
		t.Fatalf("page order: %+v %v", page, err)
	}
	older, err := storage.Chat(t.Context(), room.RoomID, page.NextBefore, 2)
	if err != nil || older.HasMore || len(older.Messages) != 1 || older.Messages[0].MessageID != ids[0] {
		t.Fatalf("cursor: %+v %v", older, err)
	}
	reply, err := storage.AddChat(t.Context(), domain.ChatMessage{MessageID: "bbbbbbbbbbbbbbbb", RoomID: room.RoomID, SenderUsername: "admin", MessageType: "review", Body: "reply", ReplyToID: ids[0]})
	if err != nil || reply.ReplyPreview == nil || reply.ReplyPreview.Body != "hello" {
		t.Fatalf("reply: %+v %v", reply, err)
	}
	if _, err = storage.UpdateChat(t.Context(), room.RoomID, ids[0], "other", true, true); err == nil {
		t.Fatal("manager recalled someone else's message")
	}
	recalled, err := storage.UpdateChat(t.Context(), room.RoomID, ids[0], "ADMIN", false, true)
	if err != nil || recalled.RecalledAt == nil || recalled.Body != "" || recalled.References == nil {
		t.Fatalf("recall: %+v %v", recalled, err)
	}
	page, err = storage.Chat(t.Context(), room.RoomID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, msg := range page.Messages {
		if msg.ReplyToID != "" && (msg.ReplyPreview.Body != "" || msg.ReplyPreview.RecalledAt == nil) {
			t.Fatalf("recalled preview leaked: %+v", msg)
		}
	}
	if _, err = storage.db.Exec("UPDATE chat_messages SET created_unix_ms = ? WHERE message_id = ?", unixMillis(time.Now().Add(-6*time.Minute)), ids[1]); err != nil {
		t.Fatal(err)
	}
	if _, err = storage.UpdateChat(t.Context(), room.RoomID, ids[1], "admin", false, true); err == nil {
		t.Fatal("recall allowed past deadline")
	}
	if _, err = storage.UpdateChat(t.Context(), room.RoomID, ids[1], "other", false, false); err == nil {
		t.Fatal("non-manager deleted another message")
	}
	deleted, err := storage.UpdateChat(t.Context(), room.RoomID, ids[1], "manager", true, false)
	if err != nil || deleted.DeletedAt == nil || deleted.Body != "" {
		t.Fatalf("delete: %+v %v", deleted, err)
	}
	if _, err = storage.Chat(t.Context(), "another-room-1234", ids[0], 10); err == nil {
		t.Fatal("cross-room cursor accepted")
	}
	profile, err := storage.MemberProfile(t.Context(), room.RoomID, "admin")
	if err != nil || profile.Stats.ChatCount != 4 {
		t.Fatalf("chat count: %+v %v", profile, err)
	}
}

func TestWishLimitPermissionsAndPersistence(t *testing.T) {
	storage, _ := openTestStore(t)
	room := createTestRoom(t, storage)
	var first domain.Wish
	for i := 0; i < 51; i++ {
		id, _ := domain.NewID()
		wish, err := storage.AddWish(t.Context(), domain.Wish{WishID: id, RoomID: room.RoomID, AuthorUsername: "admin", AuthorDisplayName: "Admin", Kind: "track", Title: "Wish"})
		if i == 50 {
			if e, ok := err.(*domain.Error); !ok || e.Code != "wish_limit_reached" {
				t.Fatalf("limit: %v", err)
			}
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = wish
		}
	}
	if err := storage.DeleteWish(t.Context(), room.RoomID, first.WishID, "other", false); err == nil {
		t.Fatal("unauthorized delete")
	}
	wish, err := storage.FulfillWish(t.Context(), room.RoomID, first.WishID, domain.User{Username: "admin", DisplayName: "Admin"}, "ready", []domain.WishResource{{Title: "Track", URL: "https://music.test/track"}})
	if err != nil || wish.Status != "fulfilled" || len(wish.Resources) != 1 {
		t.Fatalf("fulfill: %+v %v", wish, err)
	}
	if err = storage.DeleteWish(t.Context(), room.RoomID, first.WishID, "admin", false); err == nil {
		t.Fatal("author deleted fulfilled wish without manager permission")
	}
	if err = storage.DeleteWish(t.Context(), room.RoomID, first.WishID, "admin", true); err != nil {
		t.Fatal(err)
	}
	pending, err := storage.Wishes(t.Context(), room.RoomID, "pending")
	if err != nil || len(pending) != 49 {
		t.Fatalf("wishes: %d %v", len(pending), err)
	}
	if err = storage.DeleteRoom(t.Context(), room.RoomID, "admin"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = storage.db.QueryRow("SELECT COUNT(*) FROM room_wishes").Scan(&count); err != nil || count != 0 {
		t.Fatalf("wishes did not cascade: %d %v", count, err)
	}
}

func TestMemberLevelThresholds(t *testing.T) {
	for i, threshold := range []float64{0, 60, 180, 420, 900, 1800, 3600, 7200, 14400, 28800} {
		level := memberLevel(domain.MemberStats{StaySeconds: threshold * 60})
		if level.Level != i+1 || level.CurrentThresholdMinutes != threshold || (i < 9 && level.Progress != 0) {
			t.Fatalf("threshold %v: %+v", threshold, level)
		}
	}
	level := memberLevel(domain.MemberStats{StaySeconds: 30 * 60, ListenedCount: 2, RequestedCount: 5, ChatCount: 5})
	if level.EffectiveMinutes != 60 || level.Level != 2 {
		t.Fatalf("weighted level: %+v", level)
	}
}

func TestListeningThresholdsClockAndArtistBadges(t *testing.T) {
	storage, _ := openTestStore(t)
	room := createTestRoom(t, storage)
	var firstUnlock *time.Time
	for i := 0; i < 10; i++ {
		id, _ := domain.NewID()
		_, _, err := storage.AddQueueEntry(t.Context(), room, domain.QueueEntry{QueueID: id, Contributor: "admin", ContributorName: "Admin", Track: domain.NavidromeTrackRef{ID: "short-track", Title: "Short", Artist: "Artist", Album: "Album", AlbumID: "album", DurationSeconds: 20, MusicFolderID: 1}})
		if err != nil {
			t.Fatal(err)
		}
		state, err := storage.Playback(t.Context(), room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		state, err = storage.ApplyPlayback(t.Context(), room.RoomID, "play", nil, state.Revision, "admin", nil)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if _, err = storage.db.Exec("UPDATE rooms SET anchor_unix_ms = ? WHERE room_id = ?", unixMillis(time.Now().Add(-21*time.Second)), room.RoomID); err != nil {
				t.Fatal(err)
			}
			if _, changed, err := storage.AdvanceFinished(t.Context(), room.RoomID, nil); err != nil || !changed {
				t.Fatalf("clock advance: %v %v", changed, err)
			}
			profile, err := storage.MemberProfile(t.Context(), room.RoomID, "admin")
			if err != nil {
				t.Fatal(err)
			}
			firstUnlock = profile.Badges[0].UnlockedAt
		} else {
			position := 10.0
			state, err = storage.ApplyPlayback(t.Context(), room.RoomID, "seek", &position, state.Revision, "admin", nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = storage.ApplyPlayback(t.Context(), room.RoomID, "stop", nil, state.Revision, "admin", nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	profile, err := storage.MemberProfile(t.Context(), room.RoomID, "admin")
	if err != nil || profile.Stats.ListenedCount != 10 {
		t.Fatalf("half-duration listens: %+v %v", profile, err)
	}
	unlocked := map[string]bool{}
	for _, badge := range profile.Badges {
		if badge.Unlocked {
			unlocked[badge.Code] = true
			if badge.UnlockedAt == nil {
				t.Fatalf("missing unlock timestamp: %+v", badge)
			}
		}
	}
	if !unlocked["listened_10"] || !unlocked["artist_10"] || !unlocked["album_6"] || unlocked["artist_25"] {
		t.Fatalf("participation badges: %+v", profile.Badges)
	}
	if firstUnlock == nil || !profile.Badges[0].UnlockedAt.Equal(*firstUnlock) {
		t.Fatal("badge unlock timestamp changed after later activity")
	}
	state, err := storage.Playback(t.Context(), room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := storage.ApplyPlayback(t.Context(), room.RoomID, "previous", nil, state.Revision, "admin", nil)
	if err != nil || previous.Status != domain.PlaybackPaused || previous.CurrentTrack == nil {
		t.Fatalf("previous from stopped: %+v %v", previous, err)
	}
}
