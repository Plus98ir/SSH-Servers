# SSH Servers — CLAUDE.md

## What it is
Windows desktop launcher for SSH servers (Go + WebView2, frameless dark UI, FA/EN).
Click a card → `ssh` opens straight in Windows Terminal. Also: split/tab multi-open, run one command on many servers, passwordless login (SSH key install).

## Paths
- App (what Sadeq runs): `K:\Desktop Mohem\PSTools\SSH\SSH Servers.exe` + `servers.json` next to it (server list + settings)
- Source + these files: `Documents\Cloude\SSH-Servers\`
- WebView2 data: `%LOCALAPPDATA%\SSHLauncher`
- SSH keys / known_hosts: `%USERPROFILE%\.ssh` (id_ed25519, id_rsa fallback for old routers)

## Code map
- `main.go` config (servers.json), JS bridge, window setup
- `bind.go` every function the page calls (connect, openSplit/openTabs, trustHost, installKey, runMulti…)
- `sshx.go` known_hosts check/repair, key install, ping
- `launch.go` Windows Terminal command lines (single / tabs / split grid)
- `window.go` frameless window, auto-size, drag
- `ui.html` the whole UI (strings in `L.fa` / `L.en`)
- `sync.go` Gist sync: merge rules, AES-GCM, GitHub API, sync link + QR, "Sign in with GitHub" (device flow, `githubClientID`)
- `docs/index.html` website / download page (bilingual); redirects sync links, installed app and returning phones to `docs/app/`
- `docs/app/` phone PWA (GitHub Pages serves /docs) (index.html, app.js, sync.js = same format + merge as sync.go, sw.js, manifest). Hosted at `syncPage` (sync.go).
- `rsrc_windows_amd64.syso` icon + manifest (icon group id 2)

## Build (Linux sandbox)
```
export GOPROXY=direct GOSUMDB=off GOFLAGS=-mod=mod GOTOOLCHAIN=local
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o "SSH Servers.exe" .
```
golang.org/x/* are fetched from their GitHub mirrors via `replace` in go.mod (golang.org itself is blocked). Go 1.24 → x/crypto v0.41.0.

- `mobile/` Android app (Flutter): built-in SSH terminal (dartssh2 + xterm), own ed25519 key per phone (secure storage), TOFU host keys, password fallback + key install, GitHub device-flow sign-in (no Worker needed natively), QR scan, same sync format (lib/sync.dart). APK built only by GitHub Actions (`.github/workflows/android.yml`; pub.dev is blocked on this PC). `android/` is generated in CI + `tool/patch_android.py`. Signing: PKCS12 `%USERPROFILE%\.android-keys\ssh-servers.p12` (+ `ssh-servers-keystore.txt`), repo secrets KEYSTORE_BASE64 / KEYSTORE_PASSWORD / KEY_ALIAS — never regenerate. Tag `android-v*` → release with SSH-Servers.apk. Test on Sadeq's A55 via wireless adb (`adb mdns services`).

## Gotchas
- Sync: merge in sync.go and docs/app/sync.js must stay identical (tested Go↔JS encrypt/decrypt + merge). keyOK/autoTrust are device-only, never synced. Every edit stamps `u`, delete → `deleted[id]`, reorder → `orderU`.
- Sync link = `<syncPage>#g=<gist>&k=<key>&t=<token>`; fragment never reaches a server. iOS home-screen apps have their own storage → paste the link once inside the installed app.
- Host key changed (server rebuilt): the app checks before every connect, shows the new fingerprint, and one click replaces the old known_hosts entry. Per-server `autoTrust` skips the question.
- After a rebuild the key on the server is gone too → app asks for the password once and reinstalls the key.
- `legacy: true` (Modem/Dropbear) adds old ssh-rsa/kex options to the ssh command.
- Bound functions run on the UI thread — anything slow must run in a goroutine and answer via `js("onX", …)`.
- The old exe's source was lost (built in a temp sandbox, Oct 2026); ui.html was recovered from inside that exe.

## Working rules
- Script/CLI output in English; end-user pages bilingual FA + EN, dark, modern, mobile-ready.
- Tools universal — not tied to Sadeq's own servers.
- Build smart instead of asking how; ask only before destructive actions.
- Never bump a version number without asking.
- No README unless asked.
- Get it right the first time — he is short on time and tokens.
