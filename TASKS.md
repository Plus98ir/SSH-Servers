# SSH Servers — TASKS.md
_Updated 2026-11-17_

## Now
- Windows v1.2.0 released 2026-11-17: built-in terminal (tabs, side by side, JetBrains Mono, copy/paste, password fallback + key install); Windows Terminal stays as a footer option / right-click item.
- Windows v1.1.2 released 2026-11-17: in-app update reads corenetbase.com/SSH-Servers/dl/windows.json + dl/SSH.Servers.exe (GitHub only as fallback); window resizable from any edge, size remembered per view (main / file manager) in servers.json `win`, double-click the title bar = automatic size again.
- Windows v1.1.1 released 2026-11-17: fix freeze/close on save with sync on; card views (large tiles / small tiles / list, switch in the footer, per device). Exe name now carries version + build (build = git commit count, `-X main.appBuild`).

## Next
- SFTP on Android (dartssh2 has SFTP) — Sadeq said Windows only for now
- Ideas offered to Sadeq, not started: Windows app installs each phone's public key on the servers it can reach (phone passwordless from day one)
- Next version upgrade (after 5 builds): bump `appVersion` (update.go) / `version:` (mobile/pubspec.yaml, build number must grow), tag `v*` (Windows) or `android-v*` (Android), update the APK link in README.md / README.fa.md

## Version count (5 builds = one upgrade)
- Windows: 0/5 since v1.2.0 (2026-11-17)
- Android: 0/5 since 1.0.0

## Later
- (optional, web app only) deploy `worker/ssh-auth.js` as Cloudflare Worker "ssh-servers-auth" (account Cartergloria168, zone corenetbase.com, route `corenetbase.com/ssh-auth/*`) — enables "Sign in with GitHub" on the web version

## Blocked
