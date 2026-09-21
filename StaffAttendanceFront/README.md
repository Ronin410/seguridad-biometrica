# StaffAttendance — Frontend

MVP del panel administrativo (React + Vite + Tailwind). Ver `../SPEC.md`.

## Estado

Implementado: login, gestión de empleados (alta, baja/reactivación, enrolamiento facial vía webcam), gestión de turnos, kiosco de marcaje (cámara + registro manual de respaldo) y reportes con exportación a CSV.

Pendiente: notificaciones (sección 6.5, v2) y exportación a Excel/PDF (hoy solo CSV).

## Ejecutar

```bash
cp .env.example .env
npm install
npm run dev
```

Por defecto apunta a `http://localhost:8100` (configurable con `VITE_API_URL`).
