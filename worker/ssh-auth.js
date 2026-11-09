// SSH Servers — Cloudflare Worker: CORS bridge for "Sign in with GitHub" (device flow) on the phone web app.
// Browsers can't call github.com/login/* directly (no CORS), so the page posts here instead.
// Route: corenetbase.com/ssh-auth/*   (same origin as the web app)
// Locked to the SSH Servers OAuth app and the gist scope, so it is not an open proxy.
const CLIENT_ID = 'Ov23lixhLgGnIlMRameI';
const ORIGINS = ['https://corenetbase.com', 'https://www.corenetbase.com'];

function cors(req) {
  const o = req.headers.get('Origin') || '';
  return {
    'Access-Control-Allow-Origin': ORIGINS.includes(o) ? o : ORIGINS[0],
    'Access-Control-Allow-Methods': 'POST, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type',
    'Access-Control-Max-Age': '86400',
    'Vary': 'Origin',
    'Cache-Control': 'no-store',
  };
}

export default {
  async fetch(req) {
    const h = cors(req);
    if (req.method === 'OPTIONS') return new Response(null, { status: 204, headers: h });
    if (req.method !== 'POST') return new Response('POST only', { status: 405, headers: h });
    const path = new URL(req.url).pathname.replace(/\/+$/, '');
    let body = {};
    try { body = await req.json(); } catch (e) {}
    let url, form;
    if (path.endsWith('/device')) {
      url = 'https://github.com/login/device/code';
      form = { client_id: CLIENT_ID, scope: 'gist' };
    } else if (path.endsWith('/token')) {
      if (typeof body.device_code !== 'string' || !body.device_code || body.device_code.length > 200)
        return new Response('bad request', { status: 400, headers: h });
      url = 'https://github.com/login/oauth/access_token';
      form = { client_id: CLIENT_ID, device_code: body.device_code, grant_type: 'urn:ietf:params:oauth:grant-type:device_code' };
    } else return new Response('not found', { status: 404, headers: h });
    const r = await fetch(url, {
      method: 'POST',
      headers: { Accept: 'application/json', 'Content-Type': 'application/x-www-form-urlencoded', 'User-Agent': 'SSH-Servers' },
      body: new URLSearchParams(form),
    });
    return new Response(await r.text(), { status: r.status, headers: { ...h, 'Content-Type': 'application/json' } });
  },
};
