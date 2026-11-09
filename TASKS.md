# SSH Servers — TASKS.md
_Updated 2026-11-09_

## Now
- (nothing open — Windows v1.0.0 + Android 1.0.0 released and working)

## Next
- Ideas offered to Sadeq, not started: Windows app installs each phone's public key on the servers it can reach (phone passwordless from day one)
- Next release: ask Sadeq for the version number, bump `appVersion` (update.go) / `version:` (mobile/pubspec.yaml, build number must grow), tag `v*` (Windows) or `android-v*` (Android), update the APK link in README.md / README.fa.md

## Later
- (optional, web app only) deploy `worker/ssh-auth.js` as Cloudflare Worker "ssh-servers-auth" (account Cartergloria168, zone corenetbase.com, route `corenetbase.com/ssh-auth/*`) — enables "Sign in with GitHub" on the web version

## Blocked
