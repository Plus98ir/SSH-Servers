# SSH Servers — PLAN.md

## Goal
One-click SSH into Sadeq's servers from Windows, without .bat files or typing commands.

## Architecture
Go + WebView2 (jchv/go-webview2), single exe, config in servers.json next to the exe.
Connect = precheck with Go ssh (host key + key login) → launch `wt.exe` with `ssh`.

## Decisions
- 2026-10-27: When a server is rebuilt the app must handle the changed host key itself — no remembering `ssh-keygen -R`.

- 2026-11-09: Sync PC ↔ phone through a secret GitHub Gist (each user's own GitHub = the "login"; no server of ours). File encrypted AES-256-GCM, key only inside the sync link/QR. Per-server timestamps, newest edit wins, deletes kept as tombstones (180 days), order from the last reorder — a device opened after days never brings old data back.
- 2026-11-09: Phone side = PWA (docs/), not a native app: works on Android + iPhone, opens `ssh://user@host:port` in Termius/ConnectBot, copies the ssh command as fallback.

## Roadmap
- (none open)
