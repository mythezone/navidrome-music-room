package store

import (
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
		_, err := storage.AddQueueEntry(t.Context(), room, domain.QueueEntry{
			QueueID: id, RoomID: room.RoomID, Contributor: contributor, ContributorName: contributor,
			Track: domain.NavidromeTrackRef{ID: "track-" + contributor, MusicFolderID: 1, Title: contributor, DurationSeconds: 100},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	state, err := storage.ApplyPlayback(t.Context(), room.RoomID, "play", nil, 0, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if state.Revision != 1 || state.Status != domain.PlaybackPlaying || state.CurrentTrack == nil {
		t.Fatalf("unexpected playback state: %#v", state)
	}
	if _, err := storage.ApplyPlayback(t.Context(), room.RoomID, "pause", nil, 0, "admin"); err == nil {
		t.Fatal("expected stale revision conflict")
	} else if roomErr, ok := err.(*domain.Error); !ok || roomErr.Code != "revision_conflict" {
		t.Fatalf("expected revision_conflict, got %T %v", err, err)
	}
	paused, changed, err := storage.PauseForEmpty(t.Context(), room.RoomID)
	if err != nil || !changed || paused.Status != domain.PlaybackPaused || !paused.PausedForEmpty {
		t.Fatalf("empty room pause failed: %#v changed=%v err=%v", paused, changed, err)
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

func TestPluginStatePersistsLicenseChannelAndRejectsStaleGeneration(t *testing.T) {
	storage, _ := openTestStore(t)
	receivedAt := time.Now().UTC().Truncate(time.Millisecond)
	input := domain.PluginSync{
		PluginVersion:      "v1.2.3",
		Generation:         8,
		NavidromePublicURL: "https://music.example.test",
		GatewayPublicURL:   "https://rooms.example.test",
		LicenseFile:        "secrets/customer.license.json",
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
	if state.PluginVersion != input.PluginVersion || state.LicenseFile != input.LicenseFile || state.UpdateChannel != "beta" || len(state.Users) != 1 || !state.Users[0].Admin {
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
