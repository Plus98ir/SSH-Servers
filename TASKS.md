# SSH Servers — TASKS.md
_Updated 2026-11-15_

## Now
- [ ] LATER (Sadeq 2026-11-15: "later, not now"; don't bring it up unless he asks): the exe's own update check (update.go) still uses api.github.com + GitHub downloads, both blocked in Iran without VPN. Fix = read corenetbase.com/SSH-Servers/dl/ instead; needs a new exe release (v1.1.1), so only with his OK.
- Windows v1.1.0 released (SFTP file manager). Sadeq to try it on real servers.

## Next
- SFTP on Android (dartssh2 has SFTP) — Sadeq said Windows only for now
- Ideas offered to Sadeq, not started: Windows app installs each phone's public key on the servers it can reach (phone passwordless from day one)
- Next version upgrade (after 5 builds): bump `appVersion` (update.go) / `version:` (mobile/pubspec.yaml, build number must grow), tag `v*` (Windows) or `android-v*` (Android), update the APK link in README.md / README.fa.md

## Version count (5 builds = one upgrade)
- Windows: 1/5 since v1.1.0 (2026-11-11) — 2026-11-17 fix: app froze on save/delete/move when sync was on
- Android: 0/5 since 1.0.0

## Later
- (optional, web app only) deploy `worker/ssh-auth.js` as Cloudflare Worker "ssh-servers-auth" (account Cartergloria168, zone corenetbase.com, route `corenetbase.com/ssh-auth/*`) — enables "Sign in with GitHub" on the web version

## Blocked
