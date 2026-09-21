# StaffAttendance — Frontend

Scaffold inicial del panel administrativo (React + Vite + Tailwind). Ver `../SPEC.md`.

## Estado

Implementado: login, gestión de empleados (alta, baja/reactivación, enrolamiento facial vía webcam) y gestión de turnos.

Pendiente: kiosco de marcaje en tiempo real y reportes — ambos muestran una pantalla de "pendiente" hasta que el backend implemente esas rutas (ver `StaffAttendanceBack/README.md`).

## Ejecutar

```bash
cp .env.example .env
npm install
npm run dev
```

Por defecto apunta a `http://localhost:8100` (configurable con `VITE_API_URL`).
