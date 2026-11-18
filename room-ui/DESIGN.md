# Music Room Web design contract

This document describes the v1.2 room UI.

The existing FAIO room screen is the product reference. The generated desktop
concept used during the 1.0 implementation lives at:

`/home/mythezone/.codex/generated_images/01a05b22-59ae-7c73-9e4f-a0719596b70d/exec-a9bb2324-e8b7-4357-849b-8b97eeffce61.png`

The accepted mobile reference is `/tmp/music-room-after-join.png` from the FAIO
browser suite. Neither bitmap is shipped as application UI.

## Tokens

- Canvas: `#0d0e13`; elevated surface: `#15171e`; active surface: `#1d2028`.
- Primary text: `#f6f7fb`; secondary text: `#9298a8`; hairline: `#292c35`.
- Room status: `#63dfd1`; primary action/selection: `#ff647a`.
- Error: `#ff7889`; warning: `#f6bd60`; success: `#63dfd1`.
- Radius: 8px controls, 12px panels, 16px artwork; shadows remain restrained.
- UI typography: system sans stack with explicit sizes for every control.

## Information architecture

- Desktop: compact room bar, left now-playing surface, resizable divider, and right
  tabbed work area. The divider supports pointer and keyboard adjustment and
  persists the chosen width.
- Mobile: compact room bar, one content surface, fixed six-item bottom navigation.
- Tabs: Player, Queue, History, Request music, Favorites, and Chat, translated in
  the selected language. Chat is live and displays an unread count when hidden.
- The initial browser media gesture is labelled Start listening. It is not shown once
  the browser has granted playback.
- Admin/owner play, pause, seek, and previous-track controls affect the room.
  Member play/pause affects local listening. Next-track uses `canSkip`: managers,
  any member when `allowMemberSkip` is on, or the current track's contributor.
- Clicking artwork opens an immersive fullscreen layout. Lyrics follow the
  current LRC line and can expand into a lyrics-focused view; Escape exits the
  immersive view. A five-band EQ offers presets and custom gains on supported
  desktop browsers; mobile devices use native audio without the Web Audio graph.
- Room Information contains members, statistics/period leaderboards, the activity
  clock, and wishes. Member profiles show ten levels, progress, and badges.

## Localization

All controls, dialogs, errors, quick reviews, and built-in sticker labels support
`en`, `zh-CN`, `zh-TW`, `ja`, `es`, `fr`, and `de`. `lib/i18n.ts` detects the saved
`nmr:locale`, Navidrome's `i18nextLng`/`locale`, then browser preferences. The
language selector is available at login and in the room header and persists its
choice in `nmr:locale`. API error codes are translated with a safe generic fallback.

## Component inventory

- `RoomLogin`: existing-session reuse and same-origin Navidrome login.
- `RoomHeader`: room identity, presence, connection, language, information,
  settings, share, MusicMate escape, and administrator update badge.
- `NowPlaying`: immersive artwork, linked metadata, synchronized timeline,
  permission-aware transport, auto-scrolling/expandable lyrics, and EQ.
- `QueuePanel`: pending items, own-item removal, manager drag reorder, Pin-to-Top,
  Shuffle Queue, and Random Fill.
- `HistoryPanel`: paginated server playback history with one-click re-queue.
- `CatalogPanel`: OpenSubsonic search, albums, songs, artists, and queue actions.
- `FavoritesPanel`: Songs, Albums, Artists, and Playlists from the listener's account.
- `ChatPanel`: live messages, keyboard-accessible member/song suggestions, library
  cards, localized quick reviews, replies, five-minute recall, manager deletion,
  and the two-pack sticker picker.
- `RoomInfoDialog`: member list, statistics, five leaderboard periods, 24-hour
  activity clock, and wishes with manager fulfillment/resource selection.
- `MemberProfileDialog`: participation counters, level progress, achievements,
  and selection of an unlocked badge.
- `RoomSettingsDialog`: manager-editable room settings, including member skip.
- `RoomDialog` and `ShareDialog`: shared accessible modal behavior and invitations.
- `LanguageSelector`, `SongList`, `TrackActions`, and `TrackLinks`: shared language,
  library-list, queue/chat action, and artist/album navigation controls.

The screen must not add marketing copy, analytics, unrelated navigation, or a
second Navidrome shell.

## v1.2 fidelity ledger

Compared after the live browser run against both reference images:

- Desktop keeps the compact header, large left artwork/player, and bordered
  right work area. The right area defaults to the queue so it is never an empty
  duplicate of the persistent player.
- Mobile keeps one player surface, edge-safe spacing, and six fixed bottom
  destinations. The 390px acceptance viewport has a 390px scroll width.
- Cyan remains status/synchronization feedback; pink remains selection and
  transport. Typography, borders, radii, and muted metadata match the reference
  hierarchy without copying its unrelated FAIO navigation.
- Intentional differences are the Navidrome room identity, MusicMate escape,
  explicit browser-autoplay action, and the implemented community dialogs/chat.
  These correspond to real provider and permission behavior rather than
  decoration.

The historical v1.1 production captures are `docs/assets/web-room-live.png`,
`docs/assets/web-room-catalog-live.png`, and `docs/assets/web-room-mobile.png`.
They are generated by `e2e/live.mjs` from the running product, not assembled or
rewritten after capture. Set `NMR_SCREENSHOT_QUERY`, `NMR_SCREENSHOT_TRACK`, and
`NMR_SCREENSHOT_ROOM_NAME` to choose documentation data. The capture waits for
all cover images and rejects excluded artist text before writing each PNG.
The v1.2 `e2e/ui.mjs` suite exercises the production bundle with deterministic
API fixtures, including languages, responsive layout, immersive mode, community
flows, member/manager controls, and content-security-policy checks.
