// Kill switch: an old version of the web app lived at /SSH-Servers/ and registered a service worker here.
// This replacement removes itself and its caches so the site shows the current pages.
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', e => {
  e.waitUntil((async () => {
    for (const k of await caches.keys()) if (!k.startsWith('ssh-servers-v3')) await caches.delete(k);
    await self.registration.unregister();
    for (const c of await self.clients.matchAll({ type: 'window' })) c.navigate(c.url);
  })());
});
