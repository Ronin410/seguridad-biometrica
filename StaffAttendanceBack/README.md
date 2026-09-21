# StaffAttendance — Backend

MVP del backend de StaffAttendance. Ver `../SPEC.md` para el spec completo.

## Estado

Implementado:

- Migraciones del modelo de datos (`usuarios`, `turnos`, `empleados`, `registros_asistencia`).
- Autenticación JWT (`POST /login`, `POST /usuarios/registro`).
- CRUD de turnos (`POST/GET /turnos`).
- CRUD de empleados y enrolamiento facial (`POST/GET /empleados`, `PATCH /empleados/:id/estado`, `POST /empleados/:id/enrolar`).
- Cliente de AWS Rekognition contra una colección dedicada a empleados.
- Marcaje de entrada/salida (sección 6.2): `POST /asistencia/marcar` reconoce el rostro, decide entrada/salida según el último registro del día y calcula puntualidad/retardo contra el turno asignado. `POST /asistencia/marcar-manual` es el respaldo sin cámara (sección 8) para cuando falla la conexión a AWS.
- Vista en tiempo real (sección 6.3): `GET /asistencia/hoy`.
- Reportes básicos (sección 6.4): `GET /reportes/asistencia?inicio=&fin=&empleado_id=` — días trabajados, retardos, faltas, horas trabajadas/extra y % de puntualidad por empleado. La exportación a CSV se arma en el frontend a partir de este JSON.

Pendiente:

- Notificaciones (sección 6.5, v2).
- Exportación a Excel/PDF (hoy solo CSV, generado en el frontend).

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

El servidor arranca en `http://localhost:8100` (configurable con `PORT`). Si las credenciales de AWS todavía no están listas, el servidor arranca igual (solo avisa en el log) — únicamente `/asistencia/marcar` y `/empleados/:id/enrolar` necesitan Rekognition funcionando.

## Pruebas

`internal/handlers` tiene pruebas de integración contra un Postgres real (arman el router completo con `httptest` y cubren el flujo de marcaje, puntualidad/retardo, "hoy" y reportes). Si no hay una base disponible en `postgres://postgres:postgres@localhost:5432/staffattendance_test?sslmode=disable`, se saltan solas; para apuntarlas a otra base usa `TEST_DATABASE_URL`:

```bash
createdb staffattendance_test
TEST_DATABASE_URL="postgres://usuario:pass@localhost:5432/staffattendance_test?sslmode=disable" go test ./...
```
