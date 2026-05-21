/* ============================================================
   EmlakPro — Service Worker (PWA)
   ============================================================ */

const CACHE_NAME = 'emlakpro-v202605210951';
const STATIC_ASSETS = [
  '/',
  '/static/css/app.css',
  '/static/js/api.js',
  '/static/js/app.js',
  '/static/img/icon-192.png',
  '/static/manifest.json',
];

// Kurulum — statik dosyaları cache'e al
self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => cache.addAll(STATIC_ASSETS))
      .then(() => self.skipWaiting())
  );
});

// Aktivasyon — eski cache'leri temizle
self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys
        .filter(key => key !== CACHE_NAME)
        .map(key => caches.delete(key))
      )
    ).then(() => self.clients.claim())
  );
});

// Fetch — API istekleri network-first, statik dosyalar cache-first
self.addEventListener('fetch', event => {
  const url = new URL(event.request.url);

  // API istekleri her zaman network'ten git
  if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/uploads/')) {
    event.respondWith(
      fetch(event.request).catch(() =>
        new Response(JSON.stringify({ success: false, error: 'Çevrimdışısınız' }), {
          headers: { 'Content-Type': 'application/json' }
        })
      )
    );
    return;
  }

  // Statik dosyalar: cache-first, yoksa network
  event.respondWith(
    caches.match(event.request).then(cached => {
      if (cached) return cached;
      return fetch(event.request).then(response => {
        if (response.ok) {
          const clone = response.clone();
          caches.open(CACHE_NAME).then(cache => cache.put(event.request, clone));
        }
        return response;
      });
    })
  );
});
