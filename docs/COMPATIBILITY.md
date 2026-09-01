# Compatibility policy

Navidrome v0.63.2 is the minimum version because it contains the plugin host services and administration behavior used by this project. CI tests the pinned minimum and current stable release.

The REST API is namespaced at `/api/v1`. Additive JSON fields are allowed in minor releases. Removing fields, changing permissions, or changing event meaning requires `/api/v2`.

The `.ndp` file name is the plugin ID and must remain `navidrome-music-room.ndp`. Renaming it creates a separate Navidrome plugin identity and breaks updater activation.

Room storage migrations are forward-only during activation. The launcher preserves a pre-switch database backup and restores it if the new gateway fails health checks. Operators should not run two gateway versions against the same SQLite file.

FAIO and Navidrome room providers coexist. v1 does not copy rooms, members, history, or sessions between them.
