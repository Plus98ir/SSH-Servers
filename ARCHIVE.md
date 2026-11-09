# SSH Servers — ARCHIVE.md

## 2026-11-09
- Released Android 1.0.0 (tag android-v1.0.0, signed SSH-Servers.apk); calmer gesture swipes (axis lock). Site + README: Download for Android.
- Android app (mobile/): built-in SSH terminal tested on the A55 (password + passwordless OK). Terminal: floating session bar (back / name+close / + / grid), key row, sticky ctrl, gesture mode (swipe = arrows), double tap = Tab, one-press Enter. Home: ⋮ menu (Add server, tile/list view, sync, settings).
- Update check: Windows app self-updates from the latest `v*` release (footer version = manual check, pill when newer, `appVersion` in update.go); Android prompts from `android-v*` releases (Settings → Check for updates). Bump `appVersion` / pubspec version only after asking.
- Website: corenetbase.com/SSH-Servers/ is now a bilingual landing/download page; phone app moved to /SSH-Servers/app/ (syncPage updated; old QR links redirect).
- Auto sync every minute (PC + phone) on top of open/after-change/manual; FA/EN sync guide in the PC sync dialog and on the phone (setup box + settings); phone: "Sign in with GitHub" via /ssh-auth/ bridge (worker/ssh-auth.js, not deployed yet), sync button jumps to the setup box when not connected.
- GitHub OAuth App "SSH Servers" (owner Plus98ir, id 4003993, device flow on, non-expiring tokens) → Client ID in sync.go; "Sign in with GitHub" live on the PC.
- Published: public repo Plus98ir/SSH-Servers, Pages from /docs (https://corenetbase.com/SSH-Servers/), cover SSH-Servers.png, README EN+FA, release v1.0.0 (SSH.Servers.exe), site descriptions.json entry. New exe installed to K: (old kept as "SSH Servers.old.exe").
- Built server-list sync PC ↔ phone: sync.go (encrypted secret Gist, newest-wins merge, QR/link, token or sync-link setup, GitHub device-flow sign-in ready), sync button + dialog in ui.html, phone PWA in docs/ (cards → ssh://, add/edit/delete/reorder, FA/EN, offline, start from phone with a token). Go↔JS crypto + merge tested; exe built.
- Tested on a rebuilt server: new host key accepted from the card, connects. ✔

## 2026-10-27
- Rebuilt the app's Go source (old source was lost; UI recovered from the old exe) and saved it here.
- Changed host key (rebuilt server) is now handled in the app: fingerprint dialog, one-click accept, optional per-server auto-accept, auto prompt to reinstall the key when key login broke after a rebuild. Works for single connect, split/tabs, key install and run-on-many.
