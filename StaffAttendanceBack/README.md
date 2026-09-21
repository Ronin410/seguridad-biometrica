# StaffAttendance — Backend

Scaffold inicial del backend de StaffAttendance. Ver `../SPEC.md` para el spec completo.

## Estado

Implementado:

- Migraciones del modelo de datos (`usuarios`, `turnos`, `empleados`, `registros_asistencia`).
- Autenticación JWT (`POST /login`, `POST /usuarios/registro`).
- CRUD de turnos (`POST/GET /turnos`).
- CRUD de empleados y enrolamiento facial (`POST/GET /empleados`, `PATCH /empleados/:id/estado`, `POST /empleados/:id/enrolar`).
- Cliente de AWS Rekognition contra una colección dedicada a empleados.

Pendiente (ver SPEC.md secciones 6.2, 6.4, 6.5 y 8):

- Marcaje de entrada/salida en tiempo real y cálculo de puntualidad/retardo.
- Reportes y exportación (Excel/PDF/CSV).
- Notificaciones.

## Requisitos

- Go 1.24+
- PostgreSQL
- Credenciales de AWS con permisos de Rekognition

## Configuración

```bash
cp .env.example .env
# edita .env con tus credenciales
```

## Ejecutar

```bash
export $(cat .env | xargs)
go run .
```

El servidor arranca en `http://localhost:8100` (configurable con `PORT`).
