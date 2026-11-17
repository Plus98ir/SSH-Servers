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
- `window.go` frameless window, auto-size, drag, edge resize (size kept in `cfg.Win`, per view)
- `update.go` update check (site `dl/windows.json` first, GitHub API fallback) + self-update
- `ui.html` the whole UI (strings in `L.fa` / `L.en`)
- `term.go` built-in terminal (default on card click; `extTerm` = Windows Terminal instead): one SSH shell + PTY per tab (tid), output batched every 12 ms → `onTermData` (base64), keystrokes `termInput` (in order via a channel), `termResize`, `termClose`; password kept in memory per server (`termPass`); Win32 clipboard `clipGet`/`clipSet`. Page side: `TM` in ui.html (tabs, side-by-side grid, Ctrl+Shift+T/W, Ctrl+Tab, Ctrl+±/0 font, right-click copy/paste). xterm.js 5.5 + fit + web-links in `xterm/` (MIT), font JetBrains Mono in `fonts/` (OFL), injected by `uiPage()` at `/*XTERM*/`. Never name a JS function like a Go binding (it overwrites `window.<name>`). `term_test.go` runs it against an in-process SSH server.
- `sftp.go` built-in SFTP file manager (pkg/sftp): one session per server (key, else password kept in memory only; auto-reconnect + keepalive), list/mkdir/touch/rename/remove/chmod/read/write via `fsDo` → `onFs`, downloads straight to `%USERPROFILE%\Downloads`, uploads streamed from the page in 1 MB base64 chunks (`upStart`/`upChunk`/`upEnd`, flow control via `onUpAck`). `sftp_test.go` runs it against an in-process SSH+SFTP server.
- `sync.go` Gist sync: merge rules, AES-GCM, GitHub API, sync link + QR, "Sign in with GitHub" (device flow, `githubClientID`)
- `docs/index.html` website / download page (bilingual); redirects sync links, installed app and returning phones to `docs/app/`
- `docs/app/` phone PWA (GitHub Pages serves /docs) (index.html, app.js, sync.js = same format + merge as sync.go, sw.js, manifest). Hosted at `syncPage` (sync.go).
- `rsrc_windows_amd64.syso` icon + manifest (icon group id 2)

## Build (Linux sandbox)
```
export GOPROXY=direct GOSUMDB=off GOFLAGS=-mod=mod GOTOOLCHAIN=local
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w -X main.appBuild=$(git rev-list --count HEAD)" -o "SSH Servers.exe" .
```
Windows release: tag `v*` + `gh release create` with `SSH.Servers.exe` (fixed name, used by the site and the in-app update) and `SSH-Servers-v<ver>-build<N>.exe`. A release starts pages.yml on the tag, which only hands off to a run on main (the github-pages environment deploys from main only); that run also writes `dl/windows.json` (version + notes), which the app's update check reads first.
golang.org/x/* are fetched from their GitHub mirrors via `replace` in go.mod (golang.org itself is blocked). Go 1.24 → x/crypto v0.41.0.

- `mobile/` Android app (Flutter): built-in SSH terminal (dartssh2 + xterm), own ed25519 key per phone (secure storage), TOFU host keys, password fallback + key install, GitHub device-flow sign-in (no Worker needed natively), QR scan, same sync format (lib/sync.dart). APK built by GitHub Actions (`.github/workflows/android.yml`). `flutter pub get` / `flutter analyze` / `flutter test` do work locally through the mirror set in PUB_HOSTED_URL (pub.flutter-io.cn); `pub get` rewrites mobile/analysis_options.yaml, revert it. Version = `1.<run_number // 10>`, build = run number (`--build-name/--build-number` in CI); release assets `SSH-Servers.apk` + `SSH-Servers-v<ver>-build<N>.apk`, release text from `mobile/release-notes.md` (update it before tagging). pages.yml writes `dl/android.json` (version, build, notes) that the app's update check reads first; the site's download buttons show the version from dl/*.json. `android/` is generated in CI + `tool/patch_android.py`. Signing: PKCS12 `%USERPROFILE%\.android-keys\ssh-servers.p12` (+ `ssh-servers-keystore.txt`), repo secrets KEYSTORE_BASE64 / KEYSTORE_PASSWORD / KEY_ALIAS — never regenerate. Tag `android-v*` → release with SSH-Servers.apk (`make_latest: false`, so `releases/latest` stays the Windows `v*` release). Downloads (2026-11-15): page + README link https://corenetbase.com/SSH-Servers/dl/SSH.Servers.exe and dl/SSH-Servers.apk; `.github/workflows/pages.yml` (Pages build_type=workflow) copies the exe from the latest v* release and the APK from the newest android-v* release on every docs push, published release, or android-v* build (android.yml dispatches it). Never link github.com/…/releases/download or api.github.com from pages: blocked/filtered in Iran without VPN. Test on Sadeq's A55 via wireless adb (`adb mdns services`).

## Gotchas
- Sync: merge in sync.go and docs/app/sync.js must stay identical (tested Go↔JS encrypt/decrypt + merge). keyOK/autoTrust are device-only, never synced. Every edit stamps `u`, delete → `deleted[id]`, reorder → `orderU`.
- Sync link = `<syncPage>#g=<gist>&k=<key>&t=<token>`; fragment never reaches a server. iOS home-screen apps have their own storage → paste the link once inside the installed app.
- Host key changed (server rebuilt): the app checks before every connect, shows the new fingerprint, and one click replaces the old known_hosts entry. Per-server `autoTrust` skips the question.
- After a rebuild the key on the server is gone too → app asks for the password once and reinstalls the key.
- `legacy: true` (Modem/Dropbear) adds old ssh-rsa/kex options to the ssh command.
- Bound functions run on the UI thread — anything slow must run in a goroutine and answer via `js("onX", …)`.
- The page is loaded from a string, so its localStorage is gone after a restart: keep page settings with `P.get/P.set` (ui.html) → `setPref` → `cfg.Prefs` in servers.json, injected back as `window.PREFS` by `uiPage()`.
- The old exe's source was lost (built in a temp sandbox, Oct 2026); ui.html was recovered from inside that exe.

## Working rules
- Script/CLI output in English; end-user pages bilingual FA + EN, dark, modern, mobile-ready.
- Tools universal — not tied to Sadeq's own servers.
- Build smart instead of asking how; ask only before destructive actions.
- Versioning: every 5 builds = one version upgrade (Sadeq, 2026-11-11). Count builds in TASKS.md ("Version count"); no bump in between.
- No README unless asked.
- Get it right the first time — he is short on time and tokens.
