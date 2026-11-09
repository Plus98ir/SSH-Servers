// SSH Servers — phone web app. Server list synced through the encrypted Gist (see sync.js).
(() => {
'use strict';
const $ = s => document.querySelector(s);
const S = window.SSHSync;

const L = {
  fa: {
    title: 'سرورهای SSH', subtitle: 'روی سرور بزن تا توی اپ SSH باز بشه',
    newServer: 'سرور جدید', empty: 'هنوز سروری نداری.',
    edit: 'ویرایش', copyCmd: 'کپی دستور ssh', up: 'جابه‌جایی به قبل', down: 'جابه‌جایی به بعد', del: 'حذف',
    delQ: '«{n}» حذف بشه؟ از بقیهٔ دستگاه‌ها هم حذف می‌شه.', cancel: 'انصراف', save: 'ذخیره', add: 'اضافه کن', close: 'بستن',
    name: 'اسم', namePh: 'مثلاً سرور آلمان', user: 'یوزر', host: 'آدرس (IP یا دامنه)', port: 'پورت', color: 'رنگ',
    badHost: 'آدرس سرور رو درست بنویس', badUser: 'یوزر نباید فاصله یا @ داشته باشه',
    copied: 'کپی شد', openFail: 'اپ SSH باز نشد؟ دستور کپی شد؛ توی اپ SSH خودت پیستش کن.',
    setTitle: 'تنظیمات', lang: 'زبان', sync: 'سینک', syOn: 'سینک روشنه · آخرین سینک: {n}', syOff: 'سینک خاموشه',
    syNow: 'همین الان سینک کن', syStop: 'قطع سینک روی این گوشی', syStopQ: 'سینک روی این گوشی قطع بشه؟ لیست همین‌جا می‌مونه.',
    setupH: 'روشن کردن سینک',
    setupP: 'توی برنامهٔ SSH Servers روی کامپیوتر، دکمهٔ سینک (بالا) رو بزن و QR رو با دوربین گوشی اسکن کن. اگه اسکن نمی‌شه، «لینک سینک» رو کپی کن، برای خودت بفرست و اینجا بذار:',
    linkPh: 'لینک سینک یا توکن GitHub', connect: 'وصل کن', badLink: 'این نه لینک سینک درسته، نه توکن GitHub',
    noPc: 'کامپیوتر نداری؟ با یه توکن GitHub (فقط تیک gist) از همین گوشی شروع کن:', makeToken: 'ساخت توکن ↗',
    exists: 'روی این حساب GitHub قبلاً سینک ساخته شده. QR دستگاه دیگه (بخش سینک) رو اسکن کن یا «لینک سینک»ش رو اینجا بذار، یا از نو بساز (لیست این گوشی جای قبلی رو می‌گیره).', restart: 'از نو بساز',
    myLink: 'لینک سینک (برای اضافه کردن دستگاه دیگه؛ مثل رمزه، به کسی نده):', copy: 'کپی', support: 'حمایت',
    synced: 'سینک شد', got: 'لیست از دستگاه دیگه به‌روز شد', busy: 'در حال سینک…',
    eAuth: 'توکن GitHub قبول نشد (منقضی شده یا پاک شده). لینک سینک جدید رو از کامپیوتر بگیر.',
    eNotFound: 'فایل سینک روی GitHub پیدا نشد.', eKey: 'کلید سینک با فایل نمی‌خونه؛ لینک سینک رو دوباره از کامپیوتر بگیر.',
    eNet: 'به GitHub وصل نشد؛ تغییرات روی گوشی می‌مونه و دفعهٔ بعد سینک می‌شه.',
    never: 'هنوز نه', justNow: 'همین الان', minAgo: '{n} دقیقه پیش',

    ghLogin: 'ورود با GitHub', ghCode: 'این کد رو توی صفحهٔ GitHub وارد کن و Authorize رو بزن (کد برات کپی شد):', ghOpen: 'باز کردن GitHub ↗',
    ghWait: 'بعد از Authorize خودش وصل می‌شه؛ این صفحه رو نبند. حساب GitHub نداری؟ توی همون صفحه رایگان بساز.',
    ghCancel: 'ورود با GitHub لغو شد یا وقتش تموم شد', loginNA: 'ورود با GitHub الان در دسترس نیست؛ از QR کامپیوتر یا توکن استفاده کن.',
    orLink: 'یا QR برنامهٔ کامپیوتر رو با دوربین اسکن کن، یا لینک سینک / توکن GitHub رو اینجا بذار:',
    help: 'راهنما', helpTitle: 'راهنمای راه‌اندازی',
    helpBody: `<h4>۱. وصل شدن</h4><ol>
      <li><b>برنامهٔ ویندوز رو داری؟</b> توی برنامه دکمهٔ سینک (فلش‌های گرد، بالا) ← «ورود با GitHub» ← کد رو توی GitHub بزن و Authorize. بعد QR رو با دوربین گوشی اسکن کن. تمام.</li>
      <li><b>فقط گوشی؟</b> همین‌جا «ورود با GitHub» رو بزن، کد رو توی GitHub وارد کن و Authorize.</li>
      <li>حساب GitHub نداری؟ توی همون صفحهٔ ورود رایگان بساز (فقط ایمیل لازمه).</li></ol>
      <h4>۲. نصب مثل اپ</h4><ol>
      <li><b>اندروید (Chrome):</b> منوی ⋮ ← Add to Home screen یا Install app.</li>
      <li><b>آیفون (Safari):</b> دکمهٔ Share ← Add to Home Screen. بعد از نصب، یه بار دیگه از توی خود اپ وصل شو (آیفون حافظهٔ اپ رو از Safari جدا نگه می‌داره).</li></ol>
      <h4>۳. باز کردن سرور</h4><ol>
      <li>یه اپ SSH نصب کن: اندروید <b>Termius</b> یا <b>ConnectBot</b>، آیفون <b>Termius</b>.</li>
      <li>روی کارت سرور بزن تا توی اون اپ باز بشه. اگه باز نشد، دستور ssh کپی می‌شه؛ توی اپ پیستش کن.</li></ol>
      <h4>۴. سینک</h4><ol>
      <li>خودکاره: موقع باز کردن، چند ثانیه بعد از هر تغییر و هر ۱ دقیقه. دکمهٔ سینک بالا هم برای سینک دستیه.</li>
      <li>همیشه آخرین تغییر برنده‌ست؛ دستگاهی که چند روز بسته بوده اطلاعات قدیمی رو برنمی‌گردونه.</li>
      <li>فقط اسم، آدرس، پورت، یوزر و رنگ سینک می‌شه، رمزنگاری‌شده و روی GitHub خودت. پسورد هیچ‌جا نمی‌ره.</li></ol>`,
    tip: 'برای باز شدن مستقیم، یه اپ SSH مثل Termius یا ConnectBot لازمه. بعد از وصل شدن، از منوی مرورگر «Add to Home screen» رو بزن تا مثل اپ باشه.',
  },
  en: {
    title: 'SSH Servers', subtitle: 'Tap a server to open it in your SSH app',
    newServer: 'New server', empty: 'No servers yet.',
    edit: 'Edit', copyCmd: 'Copy ssh command', up: 'Move earlier', down: 'Move later', del: 'Delete',
    delQ: 'Delete “{n}”? It is removed from your other devices too.', cancel: 'Cancel', save: 'Save', add: 'Add', close: 'Close',
    name: 'Name', namePh: 'e.g. Germany server', user: 'User', host: 'Address (IP or domain)', port: 'Port', color: 'Color',
    badHost: 'Enter a valid server address', badUser: 'The user can’t contain spaces or @',
    copied: 'Copied', openFail: 'SSH app didn’t open? The command is copied — paste it in your SSH app.',
    setTitle: 'Settings', lang: 'Language', sync: 'Sync', syOn: 'Sync is on · last sync: {n}', syOff: 'Sync is off',
    syNow: 'Sync now', syStop: 'Disconnect this phone', syStopQ: 'Stop syncing on this phone? The list stays here.',
    setupH: 'Turn on sync',
    setupP: 'In SSH Servers on your PC, press the sync button (top) and scan the QR with your phone camera. If scanning doesn’t work, copy the “sync link”, send it to yourself and paste it here:',
    linkPh: 'Sync link or GitHub token', connect: 'Connect', badLink: 'That’s neither a valid sync link nor a GitHub token',
    noPc: 'No PC? Start from this phone with a GitHub token (only the gist box):', makeToken: 'Create token ↗',
    exists: 'Sync already exists on this GitHub account. Scan the QR from your other device (its Sync section) or paste its “sync link” here, or start over (this phone’s list replaces the old one).', restart: 'Start over',
    myLink: 'Sync link (to add another device; it works like a password — don’t share it):', copy: 'Copy', support: 'Support',
    synced: 'Synced', got: 'List updated from another device', busy: 'Syncing…',
    eAuth: 'GitHub didn’t accept the token (expired or deleted). Get a new sync link from the PC.',
    eNotFound: 'The sync file wasn’t found on GitHub.', eKey: 'The sync key doesn’t match the file; get the sync link again from the PC.',
    eNet: 'Couldn’t reach GitHub; changes stay on the phone and sync next time.',
    never: 'not yet', justNow: 'just now', minAgo: '{n} min ago',

    ghLogin: 'Sign in with GitHub', ghCode: 'Enter this code on the GitHub page and press Authorize (it’s already copied):', ghOpen: 'Open GitHub ↗',
    ghWait: 'It connects by itself after you authorize; keep this page open. No GitHub account? Create one for free on that page.',
    ghCancel: 'GitHub sign-in was cancelled or timed out', loginNA: 'Sign in with GitHub isn’t available right now; use the PC’s QR or a token.',
    orLink: 'Or scan the PC app’s QR with your camera, or paste a sync link / GitHub token here:',
    help: 'Guide', helpTitle: 'Setup guide',
    helpBody: `<h4>1. Connect</h4><ol>
      <li><b>Have the Windows app?</b> In the app press the sync button (round arrows, top) → “Sign in with GitHub” → enter the code on GitHub and Authorize. Then scan the QR with your phone camera. Done.</li>
      <li><b>Phone only?</b> Press “Sign in with GitHub” right here, enter the code on GitHub and Authorize.</li>
      <li>No GitHub account? Create one for free on that sign-in page (just an email).</li></ol>
      <h4>2. Install it like an app</h4><ol>
      <li><b>Android (Chrome):</b> ⋮ menu → Add to Home screen / Install app.</li>
      <li><b>iPhone (Safari):</b> Share button → Add to Home Screen. After installing, connect once more from inside the app (iPhone keeps the app’s storage apart from Safari).</li></ol>
      <h4>3. Open a server</h4><ol>
      <li>Install an SSH app: Android <b>Termius</b> or <b>ConnectBot</b>, iPhone <b>Termius</b>.</li>
      <li>Tap a server card and it opens in that app. If it doesn’t, the ssh command is copied — paste it in the app.</li></ol>
      <h4>4. Sync</h4><ol>
      <li>Automatic: on open, a few seconds after every change, and every minute. The sync button on top syncs by hand.</li>
      <li>The newest change always wins; a device closed for days never brings old data back.</li>
      <li>Only name, address, port, user and color are synced — encrypted, on your own GitHub. Passwords never leave your devices.</li></ol>`,
    tip: 'To open servers directly you need an SSH app such as Termius or ConnectBot. Once connected, use “Add to Home screen” in the browser menu so it works like an app.',
  },
};
const COLORS = ['#C00000', '#E8590C', '#F2B705', '#008000', '#12B886', '#1C7ED6', '#7048E8', '#D6336C', '#495057'];

// ---------------------------------------------------------------- storage
const store = {
  get(k, d) { try { const v = localStorage.getItem('ssh.' + k); return v ? JSON.parse(v) : d; } catch (e) { return d; } },
  set(k, v) { try { v == null ? localStorage.removeItem('ssh.' + k) : localStorage.setItem('ssh.' + k, JSON.stringify(v)); } catch (e) {} },
};
let lang = store.get('lang', (navigator.language || '').startsWith('fa') ? 'fa' : (store.get('lang') || 'fa'));
let doc = store.get('doc', S.emptyDoc());
let sc = store.get('sync', null);          // {g,k,t}
let last = store.get('last', 0);
let state = { busy: false, err: '' };

const t = (k, n) => { const v = (L[lang] && L[lang][k]) || L.fa[k] || k; return n === undefined ? v : v.replace('{n}', n); };
const esc = s => String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const faNum = n => lang === 'fa' ? String(n).replace(/\d/g, d => '۰۱۲۳۴۵۶۷۸۹'[d]) : String(n);
const addr = s => (s.user ? s.user + '@' : '') + s.host + (s.port && s.port != 22 ? ':' + s.port : '');
const cmd = s => 'ssh ' + (s.port && s.port != 22 ? '-p ' + s.port + ' ' : '') + (s.user ? s.user + '@' : '') + s.host;
function shade(hex, amt) { const n = parseInt((hex || '#495057').slice(1), 16); const c = v => Math.max(0, Math.min(255, v));
  return `rgb(${c((n >> 16) + amt)},${c(((n >> 8) & 255) + amt)},${c((n & 255) + amt)})`; }
function newID() { const a = '0123456789abcdefghijklmnopqrstuvwxyz', r = crypto.getRandomValues(new Uint8Array(12)); return [...r].map(x => a[x % 36]).join(''); }
function ago(ms) { if (!ms) return t('never'); const m = Math.round((Date.now() - ms) / 60000); if (m < 1) return t('justNow'); if (m < 60) return t('minAgo', faNum(m));
  return new Date(ms).toLocaleString(lang === 'fa' ? 'fa-IR' : 'en-GB', { dateStyle: 'short', timeStyle: 'short' }); }
let tt; function toast(m, k = '', ms = 3000) { const el = $('#toast'); el.textContent = m; el.className = 'toast show ' + k; clearTimeout(tt); tt = setTimeout(() => el.className = 'toast ' + k, ms); }
async function copy(s) { try { await navigator.clipboard.writeText(s); return true; } catch (e) {
  const i = document.createElement('textarea'); i.value = s; document.body.appendChild(i); i.select(); let ok = false; try { ok = document.execCommand('copy'); } catch (x) {} i.remove(); return ok; } }

// ---------------------------------------------------------------- render
const ICON_SRV = '<svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="7" rx="2"/><rect x="3" y="13" width="18" height="7" rx="2"/><path d="M7 7.5h.01M7 16.5h.01"/></svg>';
const ICON_MORE = '<svg viewBox="0 0 24 24" fill="currentColor"><circle cx="5" cy="12" r="1.8"/><circle cx="12" cy="12" r="1.8"/><circle cx="19" cy="12" r="1.8"/></svg>';
const ICON_PLUS = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>';
const ICON_GH = '<svg viewBox="0 0 24 24" fill="currentColor" width="20" height="20"><path d="M12 .5a11.5 11.5 0 0 0-3.6 22.4c.6.1.8-.3.8-.6v-2c-3.2.7-3.9-1.5-3.9-1.5-.5-1.3-1.3-1.7-1.3-1.7-1-.7.1-.7.1-.7 1.2.1 1.8 1.2 1.8 1.2 1 1.8 2.8 1.3 3.5 1 .1-.8.4-1.3.7-1.6-2.6-.3-5.3-1.3-5.3-5.7 0-1.3.5-2.3 1.2-3.1-.1-.3-.5-1.5.1-3.1 0 0 1-.3 3.2 1.2a11 11 0 0 1 5.8 0c2.2-1.5 3.2-1.2 3.2-1.2.6 1.6.2 2.8.1 3.1.7.8 1.2 1.8 1.2 3.1 0 4.4-2.7 5.4-5.3 5.7.4.4.8 1.1.8 2.2v3.2c0 .3.2.7.8.6A11.5 11.5 0 0 0 12 .5Z"/></svg>';
const I = p => `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${p}</svg>`;

function applyLang() {
  document.documentElement.lang = lang; document.documentElement.dir = lang === 'fa' ? 'rtl' : 'ltr';
  document.title = t('title');
  document.querySelectorAll('[data-i18n]').forEach(el => el.textContent = t(el.dataset.i18n));
}
function badge() {
  const b = $('#syncBtn');
  b.classList.toggle('busy', state.busy);
  b.classList.toggle('on', !!sc && !state.err && !state.busy);
  b.classList.toggle('err', !!sc && !!state.err && !state.busy);
}
function render() {
  applyLang(); badge();
  $('#setup').innerHTML = sc ? '' : `<div class="setup"><h2>${t('setupH')}</h2>
    <button class="btn ghb" id="ghBtn">${ICON_GH}${t('ghLogin')}</button>
    <p style="margin-top:14px">${t('orLink')}</p>
    <div class="row" style="flex-wrap:nowrap"><input id="lnk" class="ltr" placeholder="${t('linkPh')}" autocomplete="off" spellcheck="false" style="flex:1"><button class="btn primary" id="lnkGo" style="flex:none">${t('connect')}</button></div>
    <p class="note" style="margin:12px 0 0">${t('noPc')} <a href="https://github.com/settings/tokens/new?scopes=gist&description=SSH%20Servers%20sync" target="_blank" rel="noopener" style="color:var(--accent);text-decoration:none">${t('makeToken')}</a></p>
    <p class="note" style="margin:10px 0 0"><a href="#" id="helpLnk" style="color:var(--accent);text-decoration:none">📖 ${t('help')}</a></p></div>`;
  if (!sc) { $('#lnkGo').onclick = () => useLink($('#lnk').value.trim()); $('#ghBtn').onclick = ghLogin; $('#helpLnk').onclick = e => { e.preventDefault(); guide(); }; }
  let h = '';
  doc.servers.forEach((s, i) => {
    const c = s.color || '#495057';
    h += `<div class="card" data-id="${esc(s.id)}" style="animation-delay:${i * 25}ms">
      <button class="more" data-more="${esc(s.id)}" aria-label="menu">${ICON_MORE}</button>
      <div class="avatar" style="background:linear-gradient(140deg,${shade(c, 40)},${shade(c, -25)})">${esc(([...(s.name || '?')][0] || '?').toUpperCase())}</div>
      <div class="name">${esc(s.name)}</div><div class="addr">${esc(addr(s))}</div></div>`;
  });
  h += `<div class="card add" id="addCard"><div class="plus">${ICON_PLUS}</div><div>${t('newServer')}</div></div>`;
  if (!doc.servers.length) h += `<div class="empty">${t('empty')}</div>`;
  $('#grid').innerHTML = h;
}

// ---------------------------------------------------------------- local edits (every edit is stamped, newest wins on every device)
function commit() { store.set('doc', doc); render(); syncSoon(); }
function saveServer(s) {
  s.u = Date.now();
  const i = doc.servers.findIndex(x => x.id === s.id);
  if (i >= 0) doc.servers[i] = s; else doc.servers.push(s);
  commit();
}
function delServer(id) {
  doc.servers = doc.servers.filter(x => x.id !== id);
  doc.deleted = doc.deleted || {}; doc.deleted[id] = Date.now();
  commit();
}
function move(id, d) {
  const i = doc.servers.findIndex(x => x.id === id), j = i + d;
  if (i < 0 || j < 0 || j >= doc.servers.length) return;
  [doc.servers[i], doc.servers[j]] = [doc.servers[j], doc.servers[i]];
  doc.orderU = Date.now();
  commit();
}

// ---------------------------------------------------------------- sync
let syncTimer, again = false;
function syncSoon(ms = 900) { if (!sc) return; clearTimeout(syncTimer); syncTimer = setTimeout(runSync, ms); }
function errText(e) { const m = e && e.message; return m === 'auth' ? t('eAuth') : m === 'notfound' ? t('eNotFound') : m === 'badkey' ? t('eKey') : m === 'net' ? t('eNet') : String(m || e); }
async function runSync(manual) {
  if (!sc) return;
  if (state.busy) { again = true; return; }
  state.busy = true; badge();
  try {
    const before = JSON.stringify(doc);
    const r = await S.syncDoc(sc, doc);
    // keep any edit made while the request was in flight
    if (JSON.stringify(doc) !== before) { again = true; doc = S.merge(doc, r.doc); } else doc = r.doc;
    store.set('doc', doc);
    last = Date.now(); store.set('last', last);
    state.err = '';
    if (r.changed) { render(); toast(t('got')); } else if (manual) toast(t('synced'));
  } catch (e) {
    state.err = errText(e);
    if (manual || e.message !== 'net') toast(state.err, 'err', 6000);
  } finally {
    state.busy = false; badge();
    if ($('#scrim').classList.contains('show') && $('#sheet').dataset.kind === 'settings') settings();
    if (again) { again = false; syncSoon(300); }
  }
}
async function useLink(s, replace) {
  s = (s || '').trim();
  let p = S.parseLink(s);
  if (!p && /^[A-Za-z0-9_]{20,255}$/.test(s)) {
    const b = $('#lnkGo'); if (b) { b.disabled = true; b.textContent = t('busy'); }
    try {
      const r = await S.fromToken(s, doc, replace);
      if (r.exists) {
        sheet(`<h3>${t('sync')}</h3><p>${t('exists')}</p><div class="row"><button class="btn danger" id="xGo">${t('restart')}</button><button class="btn" id="xNo">${t('cancel')}</button></div>`);
        $('#xGo').onclick = () => { closeSheet(); useLink(s, true); };
        $('#xNo').onclick = closeSheet;
        return;
      }
      sc = r.sc; store.set('sync', sc); state.err = ''; last = Date.now(); store.set('last', last);
      render(); toast(t('synced')); return;
    } catch (e) { toast(errText(e), 'err', 6000); return; }
    finally { const b2 = $('#lnkGo'); if (b2) { b2.disabled = false; b2.textContent = t('connect'); } }
  }
  if (!p) { toast(t('badLink'), 'err'); return; }
  sc = p; store.set('sync', sc); state.err = '';
  render(); await runSync(true);
}

// ---------------------------------------------------------------- sheets
function sheet(h, kind = '') { const el = $('#sheet'); el.innerHTML = h; el.dataset.kind = kind; $('#scrim').classList.add('show'); }
function closeSheet() { $('#scrim').classList.remove('show'); $('#sheet').dataset.kind = ''; }
$('#scrim').onclick = e => { if (e.target.id === 'scrim') closeSheet(); };

function menu(id) {
  const s = doc.servers.find(x => x.id === id); if (!s) return;
  sheet(`<h3>${esc(s.name)}</h3><div class="cmd">${esc(cmd(s))}</div><div class="list">
    <button id="mEdit">${I('<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/>')}${t('edit')}</button>
    <button id="mCopy">${I('<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15V5a2 2 0 0 1 2-2h10"/>')}${t('copyCmd')}</button>
    <button id="mUp">${I('<path d="m18 15-6-6-6 6"/>')}${t('up')}</button>
    <button id="mDown">${I('<path d="m6 9 6 6 6-6"/>')}${t('down')}</button>
    <button id="mDel" class="del">${I('<path d="M3 6h18"/><path d="M8 6V4h8v2"/><path d="M19 6l-1 14H6L5 6"/>')}${t('del')}</button></div>`);
  $('#mEdit').onclick = () => editor(s);
  $('#mCopy').onclick = async () => { closeSheet(); if (await copy(cmd(s))) toast(t('copied')); };
  $('#mUp').onclick = () => { closeSheet(); move(id, -1); };
  $('#mDown').onclick = () => { closeSheet(); move(id, 1); };
  $('#mDel').onclick = () => {
    sheet(`<h3>${t('del')}</h3><p>${esc(t('delQ', s.name))}</p><div class="row"><button class="btn danger" id="dGo">${t('del')}</button><button class="btn" id="dNo">${t('cancel')}</button></div>`);
    $('#dGo').onclick = () => { closeSheet(); delServer(id); };
    $('#dNo').onclick = closeSheet;
  };
}

function editor(s) {
  const isNew = !s; s = s ? { ...s } : { id: '', name: '', user: 'root', host: '', port: 22, color: COLORS[doc.servers.length % COLORS.length] };
  let color = s.color || COLORS[0];
  sheet(`<h3>${isNew ? t('newServer') : t('edit')}</h3>
    <div class="f"><label>${t('name')}</label><input id="eName" maxlength="32" value="${esc(s.name)}" placeholder="${t('namePh')}"></div>
    <div class="three f">
      <div><label>${t('user')}</label><input id="eUser" class="ltr" value="${esc(s.user)}" autocapitalize="off" autocorrect="off" spellcheck="false"></div>
      <div><label>${t('host')}</label><input id="eHost" class="ltr" value="${esc(s.host)}" placeholder="1.2.3.4" inputmode="url" autocapitalize="off" autocorrect="off" spellcheck="false"></div>
      <div><label>${t('port')}</label><input id="ePort" class="ltr" value="${esc(s.port || 22)}" inputmode="numeric"></div>
    </div>
    <div class="f"><label>${t('color')}</label><div class="sw-row" id="eCol">${COLORS.map(c => `<span class="swatch ${c === color ? 'sel' : ''}" data-c="${c}" style="background:${c}"></span>`).join('')}</div></div>
    <div class="row" style="margin-top:6px"><button class="btn primary" id="eSave">${isNew ? t('add') : t('save')}</button><button class="btn" id="eNo">${t('cancel')}</button></div>`);
  $('#eCol').onclick = e => { const w = e.target.closest('.swatch'); if (!w) return; color = w.dataset.c; $('#eCol').querySelectorAll('.swatch').forEach(x => x.classList.toggle('sel', x === w)); };
  $('#eNo').onclick = closeSheet;
  $('#eSave').onclick = () => {
    const host = $('#eHost').value.trim(), user = $('#eUser').value.trim();
    if (!host || /[\s@;"']/.test(host)) { toast(t('badHost'), 'err'); return; }
    if (/[\s@;"']/.test(user)) { toast(t('badUser'), 'err'); return; }
    let port = parseInt($('#ePort').value, 10); if (!(port > 0 && port < 65536)) port = 22;
    const out = { ...s, id: s.id || newID(), name: $('#eName').value.trim() || host, user, host, port, color };
    if (!isNew && (out.host !== s.host || out.user !== s.user || out.port !== s.port)) out.legacy = false;
    closeSheet(); saveServer(out);
  };
  if (isNew) $('#eHost').focus();
}

function guide() {
  sheet(`<h3>${t('helpTitle')}</h3><div class="guide">${t('helpBody')}</div><div class="row" style="margin-top:6px"><button class="btn" id="hClose">${t('close')}</button></div>`);
  $('#hClose').onclick = closeSheet;
}

// "Sign in with GitHub" (device flow through the /ssh-auth/ bridge on the same domain)
async function ghLogin() {
  let dc;
  try {
    const r = await fetch('/ssh-auth/device', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' });
    dc = r.ok ? await r.json() : null;
  } catch (e) { dc = null; }
  if (!dc || !dc.device_code) { toast(t('loginNA'), 'err', 6000); return; }
  copy(dc.user_code);
  sheet(`<h3>${t('ghLogin')}</h3><p>${t('ghCode')}</p><div class="ucode">${esc(dc.user_code)}</div>
    <div class="row"><a class="btn primary" href="${esc(dc.verification_uri)}" target="_blank" rel="noopener" style="text-decoration:none">${t('ghOpen')}</a><button class="btn" id="gNo">${t('cancel')}</button></div>
    <p class="note" style="margin-top:12px">${t('ghWait')}</p>`, 'login');
  let alive = true;
  $('#gNo').onclick = () => { alive = false; closeSheet(); };
  let iv = Math.max(5, dc.interval || 5) * 1000;
  const end = Date.now() + (dc.expires_in || 900) * 1000;
  while (alive && Date.now() < end) {
    await new Promise(r => setTimeout(r, iv));
    if (!alive || $('#sheet').dataset.kind !== 'login') return;
    let j;
    try { j = await (await fetch('/ssh-auth/token', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ device_code: dc.device_code }) })).json(); }
    catch (e) { continue; }
    if (j.access_token) { closeSheet(); await useLink(j.access_token); return; }
    if (j.error === 'slow_down') { iv += 5000; continue; }
    if (j.error === 'authorization_pending') continue;
    break;
  }
  if ($('#sheet').dataset.kind === 'login') { closeSheet(); toast(t('ghCancel'), 'err'); }
}

function settings() {
  const st = !sc ? `<div class="sstat off"><span class="d"></span>${t('syOff')}</div>`
    : state.err ? `<div class="sstat err"><span class="d"></span>${esc(state.err)}</div>`
    : `<div class="sstat"><span class="d"></span>${t('syOn', ago(last))}</div>`;
  sheet(`<h3>${t('setTitle')}</h3>
    <div class="f"><label>${t('lang')}</label></div>
    <div class="seg" id="sLang"><button data-l="fa" class="${lang === 'fa' ? 'on' : ''}">فارسی</button><button data-l="en" class="${lang === 'en' ? 'on' : ''}">English</button></div>
    <div class="f"><label>${t('sync')}</label></div>${st}
    ${sc ? `<p class="note" style="margin-bottom:6px">${t('myLink')}</p>
      <div class="row" style="flex-wrap:nowrap;margin-bottom:12px"><input id="sLink" class="ltr" readonly value="${esc(S.linkOf(sc, location.origin + location.pathname))}" style="flex:1;font-size:12px;font-family:var(--mono)"><button class="btn" id="sCopy" style="flex:none">${t('copy')}</button></div>
      <div class="row" style="margin-bottom:10px"><button class="btn primary" id="sNow" ${state.busy ? 'disabled' : ''}>${state.busy ? t('busy') : t('syNow')}</button></div>
      <div class="row" style="margin-bottom:14px"><button class="btn" id="sStop" style="color:var(--danger)">${t('syStop')}</button></div>` : ''}
    <p class="note" style="margin-bottom:14px">${t('tip')}</p>
    <div class="row" style="margin-bottom:8px"><button class="btn" id="sHelp">📖 ${t('help')}</button></div>
    <div class="row"><button class="btn" data-support>☕ ${t('support')}</button><a class="btn" href="https://github.com/Plus98ir/SSH-Servers" target="_blank" rel="noopener" style="text-decoration:none">GitHub ↗</a></div>
    <div class="row" style="margin-top:8px"><button class="btn" id="sClose">${t('close')}</button></div>`, 'settings');
  $('#sLang').onclick = e => { const b = e.target.closest('button'); if (!b) return; lang = b.dataset.l; store.set('lang', lang); render(); settings(); };
  $('#sClose').onclick = closeSheet;
  $('#sHelp').onclick = guide;
  if (sc) {
    $('#sNow').onclick = () => runSync(true);
    $('#sCopy').onclick = async () => { if (await copy($('#sLink').value)) toast(t('copied')); };
    $('#sStop').onclick = () => {
      sheet(`<h3>${t('syStop')}</h3><p>${t('syStopQ')}</p><div class="row"><button class="btn danger" id="oGo">${t('syStop')}</button><button class="btn" id="oNo">${t('cancel')}</button></div>`);
      $('#oGo').onclick = () => { sc = null; store.set('sync', null); state.err = ''; closeSheet(); render(); };
      $('#oNo').onclick = settings;
    };
  }
}

// ---------------------------------------------------------------- events
$('#grid').onclick = e => {
  const m = e.target.closest('[data-more]'); if (m) { e.stopPropagation(); menu(m.dataset.more); return; }
  if (e.target.closest('#addCard')) { editor(null); return; }
  const c = e.target.closest('.card[data-id]'); if (!c) return;
  const s = doc.servers.find(x => x.id === c.dataset.id); if (!s) return;
  const url = 'ssh://' + (s.user ? encodeURIComponent(s.user) + '@' : '') + s.host + ':' + (s.port || 22);
  let left = false; const onHide = () => { left = true; };
  document.addEventListener('visibilitychange', onHide, { once: true });
  location.href = url;
  setTimeout(async () => { document.removeEventListener('visibilitychange', onHide); if (!left && !document.hidden) { await copy(cmd(s)); toast(t('openFail'), '', 5000); } }, 1600);
};
$('#grid').addEventListener('contextmenu', e => { const c = e.target.closest('.card[data-id]'); if (c) { e.preventDefault(); menu(c.dataset.id); } });
$('#syncBtn').onclick = () => {
  if (sc) { runSync(true); return; }
  const box = $('#setup .setup'); if (!box) return;
  box.scrollIntoView({ behavior: 'smooth', block: 'start' });
  box.animate([{ boxShadow: '0 0 0 0 rgba(34,195,142,.0)' }, { boxShadow: '0 0 0 4px rgba(34,195,142,.55)' }, { boxShadow: '0 0 0 0 rgba(34,195,142,.0)' }], { duration: 1200 });
};
setInterval(() => { if (sc && !document.hidden && !state.busy) runSync(); }, 60000);
$('#setBtn').onclick = settings;
document.addEventListener('visibilitychange', () => { if (!document.hidden) syncSoon(200); });
window.addEventListener('online', () => syncSoon(200));

// ---------------------------------------------------------------- start
(async () => {
  if (location.hash.length > 1) {
    const p = S.parseLink(location.href);
    history.replaceState(null, '', location.pathname + location.search);
    if (p) { sc = p; store.set('sync', sc); }
  }
  render();
  if (sc) runSync();
  if ('serviceWorker' in navigator) try { navigator.serviceWorker.register('sw.js'); } catch (e) {}
})();
})();
