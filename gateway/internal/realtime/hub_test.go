package realtime

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestReconnectCancelsEmptyRoomPause(t *testing.T) {
	called := make(chan string, 1)
	hub := NewHub(40*time.Millisecond, func(_ context.Context, roomID string) {
		called <- roomID
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	first := &client{username: "alice"}
	hub.add("room", first)
	hub.remove("room", first)
	second := &client{username: "bob"}
	hub.add("room", second)
	select {
	case roomID := <-called:
		t.Fatalf("room %q paused after a listener reconnected", roomID)
	case <-time.After(90 * time.Millisecond):
	}
}

func TestEmptyRoomPauseRunsOnce(t *testing.T) {
	called := make(chan string, 2)
	hub := NewHub(10*time.Millisecond, func(_ context.Context, roomID string) {
		called <- roomID
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	item := &client{username: "alice"}
	hub.add("room", item)
	hub.remove("room", item)
	select {
	case roomID := <-called:
		if roomID != "room" {
			t.Fatalf("unexpected room callback: %q", roomID)
		}
	case <-time.After(time.Second):
		t.Fatal("empty-room callback did not run")
	}
	select {
	case <-called:
		t.Fatal("empty-room callback ran more than once")
	case <-time.After(30 * time.Millisecond):
	}
}

func TestAllRoomsOnlineUsersDeduplicatesAndReturnsCopy(t *testing.T) {
	hub := NewHub(time.Second, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	hub.add("one", &client{username: "alice"})
	hub.add("one", &client{username: "ALICE"})
	hub.add("one", &client{username: "bob"})
	hub.add("two", &client{username: "alice"})
	rooms := hub.AllRoomsOnlineUsers()
	if len(rooms) != 2 || len(rooms["one"]) != 2 || len(rooms["two"]) != 1 {
		t.Fatalf("online users: %+v", rooms)
	}
	rooms["one"][0] = "changed"
	if next := hub.AllRoomsOnlineUsers(); next["one"][0] == "changed" {
		t.Fatal("caller mutated hub state")
	}
}
