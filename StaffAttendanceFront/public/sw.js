// Service worker mínimo de la PWA (SPEC.md sección 8): sirve la app desde
// caché cuando la tablet pierde internet, para que el kiosco siga
// abriendo y pueda usar la cola de marcajes offline. No cachea llamadas a
// la API (/login, /empleados, /asistencia/*, etc.) — esas siempre van a
// la red; si fallan, las maneja la cola offline del propio frontend.
const CACHE_NAME = 'staffattendance-shell-v1';

self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key)))
    )
  );
  self.clients.claim();
});

function esLlamadaApi(url) {
  return url.pathname.startsWith('/login') ||
    url.pathname.startsWith('/negocios') ||
    url.pathname.startsWith('/usuarios') ||
    url.pathname.startsWith('/empleados') ||
    url.pathname.startsWith('/turnos') ||
    url.pathname.startsWith('/asistencia') ||
    url.pathname.startsWith('/reportes');
}

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);
  if (event.request.method !== 'GET' || url.origin !== self.location.origin || esLlamadaApi(url)) {
    return;
  }

  event.respondWith(
    caches.open(CACHE_NAME).then(async (cache) => {
      try {
        const response = await fetch(event.request);
        cache.put(event.request, response.clone());
        return response;
      } catch (err) {
        const cached = await cache.match(event.request);
        if (cached) return cached;
        throw err;
      }
    })
  );
});
