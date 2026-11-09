// Sync core shared by the phone web app (same format and merge rules as sync.go).
// Gist file ssh-servers.json = {app,v,iv,ct}: AES-256-GCM over the JSON doc
// {v, servers:[{id,name,user,host,port,color,legacy,u}], deleted:{id:ms}, orderU}.
(function (root) {
  const FILE = 'ssh-servers.json';
  const TOMB_LIFE = 180 * 24 * 3600 * 1000;
  const subtle = (root.crypto || require('crypto').webcrypto).subtle;
  const cryptoObj = root.crypto || require('crypto').webcrypto;

  const b64 = {
    enc: u8 => { let s = ''; u8.forEach(c => s += String.fromCharCode(c)); return btoa(s); },
    dec: s => Uint8Array.from(atob(s), c => c.charCodeAt(0)),
    urlDec: s => { s = s.replace(/-/g, '+').replace(/_/g, '/'); while (s.length % 4) s += '='; return Uint8Array.from(atob(s), c => c.charCodeAt(0)); },
  };

  function emptyDoc() { return { v: 1, servers: [], deleted: {}, orderU: 0 }; }

  // Newest edit of each server wins, deletions beat older edits,
  // order comes from the side that reordered last. Ties go to a.
  function merge(a, b) {
    const del = {};
    for (const m of [a.deleted || {}, b.deleted || {}])
      for (const k in m) if (!(k in del) || m[k] > del[k]) del[k] = m[k];
    const win = {};
    for (const list of [a.servers || [], b.servers || []])
      for (const s of list) if (!win[s.id] || (s.u || 0) > (win[s.id].u || 0)) win[s.id] = s;
    const [first, second] = (b.orderU || 0) > (a.orderU || 0) ? [b, a] : [a, b];
    const out = [], seen = {};
    for (const list of [first.servers || [], second.servers || []])
      for (const s of list) {
        if (seen[s.id]) continue;
        seen[s.id] = 1;
        const w = win[s.id];
        if (s.id in del && del[s.id] >= (w.u || 0)) continue;
        out.push(w);
      }
    const cut = Date.now() - TOMB_LIFE;
    for (const k in del) if (del[k] < cut) delete del[k];
    return { v: 1, servers: out, deleted: del, orderU: Math.max(a.orderU || 0, b.orderU || 0) };
  }

  // canonical form so two docs compare equal no matter where they came from
  function canon(d) {
    const keys = Object.keys(d.deleted || {}).sort();
    const del = {}; keys.forEach(k => del[k] = d.deleted[k]);
    return JSON.stringify({
      v: 1,
      servers: (d.servers || []).map(s => ({ id: s.id, name: s.name, user: s.user, host: s.host, port: s.port, color: s.color, legacy: !!s.legacy, u: s.u || 0 })),
      deleted: del, orderU: d.orderU || 0,
    });
  }
  const same = (a, b) => canon(a) === canon(b);

  async function importKey(k) {
    const raw = b64.urlDec(k);
    if (raw.length !== 32) throw new Error('badkey');
    return subtle.importKey('raw', raw, 'AES-GCM', false, ['encrypt', 'decrypt']);
  }

  async function seal(k, doc) {
    const key = await importKey(k);
    const iv = cryptoObj.getRandomValues(new Uint8Array(12));
    const plain = new TextEncoder().encode(JSON.stringify(doc));
    const ct = new Uint8Array(await subtle.encrypt({ name: 'AES-GCM', iv }, key, plain));
    return JSON.stringify({ app: 'ssh-servers', v: 1, iv: b64.enc(iv), ct: b64.enc(ct) }, null, 1);
  }

  async function unseal(k, content) {
    let env;
    try { env = JSON.parse(content); } catch (e) { throw new Error('badkey'); }
    if (!env || env.app !== 'ssh-servers') throw new Error('badkey');
    const key = await importKey(k);
    let plain;
    try { plain = await subtle.decrypt({ name: 'AES-GCM', iv: b64.dec(env.iv) }, key, b64.dec(env.ct)); }
    catch (e) { throw new Error('badkey'); }
    const d = JSON.parse(new TextDecoder().decode(plain));
    d.servers = d.servers || []; d.deleted = d.deleted || {};
    return d;
  }

  async function gh(method, path, token, body) {
    let res;
    try {
      res = await fetch('https://api.github.com' + path, {
        method, cache: 'no-store',
        headers: { Authorization: 'Bearer ' + token, Accept: 'application/vnd.github+json', ...(body ? { 'Content-Type': 'application/json' } : {}) },
        body: body ? JSON.stringify(body) : undefined,
      });
    } catch (e) { throw new Error('net'); }
    if (res.status === 401 || res.status === 403) throw new Error('auth');
    if (res.status === 404) throw new Error('notfound');
    if (!res.ok) throw new Error('GitHub ' + res.status);
    return method === 'PATCH' ? null : res.json();
  }

  async function pull(sc) {
    const g = await gh('GET', '/gists/' + encodeURIComponent(sc.g), sc.t);
    const f = g.files && g.files[FILE];
    if (!f) throw new Error('notfound');
    let content = f.content;
    if (f.truncated) content = await (await fetch(f.raw_url, { cache: 'no-store' })).text();
    return unseal(sc.k, content);
  }

  async function push(sc, doc) {
    await gh('PATCH', '/gists/' + encodeURIComponent(sc.g), sc.t, { files: { [FILE]: { content: await seal(sc.k, doc) } } });
  }

  // pull -> merge -> push if needed. Returns {doc, changed}
  async function syncDoc(sc, local) {
    const remote = await pull(sc);
    const merged = merge(local, remote);
    if (!same(merged, remote)) await push(sc, merged);
    return { doc: merged, changed: !same(merged, local) };
  }

  // start from a bare GitHub token: returns {exists:true, g} or creates a new encrypted gist -> {sc}
  async function fromToken(token, doc, replace) {
    const list = await gh('GET', '/gists?per_page=100', token);
    const found = (list || []).find(g => g.files && g.files[FILE]);
    if (found && !replace) return { exists: true };
    const raw = cryptoObj.getRandomValues(new Uint8Array(32));
    const k = b64.enc(raw).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
    const content = await seal(k, doc);
    let g;
    if (found) { g = found.id; await gh('PATCH', '/gists/' + g, token, { files: { [FILE]: { content } } }); }
    else g = (await gh('POST', '/gists', token, { description: 'SSH Servers sync (encrypted)', public: false, files: { [FILE]: { content } } })).id;
    return { sc: { g, k, t: token } };
  }

  function linkOf(sc, base) { return base + '#g=' + encodeURIComponent(sc.g) + '&k=' + encodeURIComponent(sc.k) + '&t=' + encodeURIComponent(sc.t); }

  function parseLink(s) {
    const i = s.indexOf('#'); if (i < 0) return null;
    const q = new URLSearchParams(s.slice(i + 1));
    const sc = { g: q.get('g'), k: q.get('k'), t: q.get('t') };
    if (!sc.g || !sc.k || !sc.t) return null;
    try { if (b64.urlDec(sc.k).length !== 32) return null; } catch (e) { return null; }
    return sc;
  }

  const api = { fromToken, linkOf, merge, same, seal, unseal, pull, push, syncDoc, parseLink, emptyDoc };
  if (typeof module !== 'undefined') module.exports = api; else root.SSHSync = api;
})(typeof window !== 'undefined' ? window : globalThis);
