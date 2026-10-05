# StaffAttendance — Frontend

MVP del panel administrativo, PWA (React + Vite + Tailwind). Ver `../SPEC.md`.

## Estado

Implementado: alta de negocio nuevo y login (multi-negocio, sección 8), gestión de empleados (alta, baja/reactivación, enrolamiento facial vía webcam), gestión de turnos, kiosco de marcaje (cámara + registro manual de respaldo), reportes con exportación a CSV, y gestión de usuarios/perfiles.

**Perfiles (sección 4):** una cuenta `admin` ve las cinco pestañas (Kiosco, Empleados, Turnos, Reportes, Usuarios); una cuenta `kiosco` — la de la tablet del mostrador — solo ve Kiosco, sin ni siquiera el selector de pestañas. Esto es una comodidad de interfaz: el límite real está en el backend (`middleware.RequireRol`), así que un `kiosco` tampoco puede llegar a esas pantallas llamando a la API directamente. Un admin crea la cuenta de cada tablet desde la pestaña **Usuarios**.

**PWA:** manifest + service worker (`public/manifest.json`, `public/sw.js`) para poder instalarse en cualquier tablet. El service worker solo cachea el shell de la app (nunca las llamadas a la API) para que el kiosco siga abriendo sin internet.

**Modo sin internet (sección 8):** si el registro manual (`Kiosco.jsx`) no logra llegar al servidor, el marcaje se guarda en el dispositivo (`src/offlineQueue.js`, con la hora real de captura) y se sincroniza solo en cuanto vuelve la conexión — no bloquea el registro de asistencia.

Pendiente: notificaciones (sección 6.5, v2), exportación a Excel/PDF (hoy solo CSV), e integración con nómina (a futuro).

## Ejecutar

```bash
cp .env.example .env
npm install
npm run dev
```

Por defecto apunta a `http://localhost:8100` (configurable con `VITE_API_URL`).

## Primer uso

En la pantalla de login, "¿Negocio nuevo? Crear cuenta" da de alta un negocio y su usuario admin (llama a `POST /negocios`). Cada negocio ve únicamente sus propios empleados, turnos y reportes.
