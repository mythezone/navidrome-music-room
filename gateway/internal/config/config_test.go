package config

import (
	"testing"
	"time"
)

func TestUpdateConfiguration(t *testing.T) {
	t.Setenv("MUSIC_ROOM_DATA_DIR", t.TempDir())
	t.Setenv("MUSIC_ROOM_NAVIDROME_INTERNAL_URL", "http://navidrome:4533")
	t.Setenv("MUSIC_ROOM_AUTO_UPDATE_CHECK", "")
	t.Setenv("MUSIC_ROOM_UPDATE_CHECK_INTERVAL", "")
	t.Setenv("MUSIC_ROOM_UPDATE_WEBHOOK_URL", "")
	cfg, err := Load()
	if err != nil || !cfg.AutoUpdateCheck || cfg.UpdateCheckInterval != 6*time.Hour || cfg.UpdateWebhookURL != "" {
		t.Fatalf("defaults: %+v %v", cfg, err)
	}
	t.Setenv("MUSIC_ROOM_AUTO_UPDATE_CHECK", "false")
	t.Setenv("MUSIC_ROOM_UPDATE_CHECK_INTERVAL", "30m")
	t.Setenv("MUSIC_ROOM_UPDATE_WEBHOOK_URL", "https://hooks.test/update?token=secret")
	cfg, err = Load()
	if err != nil || cfg.AutoUpdateCheck || cfg.UpdateCheckInterval != 30*time.Minute || cfg.UpdateWebhookURL != "https://hooks.test/update?token=secret" {
		t.Fatalf("overrides: %+v %v", cfg, err)
	}
	for _, invalid := range []string{"ftp://hooks.test/update", "/relative", "https://", "https://user:secret@hooks.test/update", "https://hooks.test/#fragment", ":bad"} {
		t.Setenv("MUSIC_ROOM_UPDATE_WEBHOOK_URL", invalid)
		if _, err = Load(); err == nil {
			t.Fatalf("invalid webhook %q accepted", invalid)
		}
	}
}
