# SSH Servers — TASKS.md
_Updated 2026-11-11_

## Now
- Windows v1.1.0 released (SFTP file manager). Sadeq to try it on real servers.

## Next
- SFTP on Android (dartssh2 has SFTP) — Sadeq said Windows only for now
- Ideas offered to Sadeq, not started: Windows app installs each phone's public key on the servers it can reach (phone passwordless from day one)
- Next version upgrade (after 5 builds): bump `appVersion` (update.go) / `version:` (mobile/pubspec.yaml, build number must grow), tag `v*` (Windows) or `android-v*` (Android), update the APK link in README.md / README.fa.md

## Version count (5 builds = one upgrade)
- Windows: 0/5 since v1.1.0 (2026-11-11)
- Android: 0/5 since 1.0.0

## Later
- (optional, web app only) deploy `worker/ssh-auth.js` as Cloudflare Worker "ssh-servers-auth" (account Cartergloria168, zone corenetbase.com, route `corenetbase.com/ssh-auth/*`) — enables "Sign in with GitHub" on the web version

## Blocked
