# StaffAttendance — Backend

MVP del backend de StaffAttendance, multi-negocio. Ver `../SPEC.md` para el spec completo.

## Estado

Implementado:

- Multi-negocio (sección 8): tabla `negocios`, `POST /negocios` da de alta un negocio nuevo junto con su primer usuario admin, y todo (`usuarios`, `turnos`, `empleados`, `registros_asistencia`) queda aislado por `negocio_id` — viene en el JWT y se filtra en cada consulta.
- Autenticación JWT (`POST /login`, `POST /usuarios/registro` — este último ya autenticado, solo agrega usuarios al negocio del token).
- CRUD de turnos (`POST/GET /turnos`) y de empleados (`POST/GET /empleados`, `PATCH /empleados/:id/estado`), enrolamiento facial (`POST /empleados/:id/enrolar`).
- Un negocio, una colección de Rekognition (`empleados-asistencia-{negocio_id}`), igual que GuarderiaBiometric por guardería.
- Marcaje de entrada/salida (sección 6.2): `POST /asistencia/marcar` reconoce el rostro, decide entrada/salida según el último registro del día y calcula puntualidad/retardo contra el turno asignado.
- Respaldo manual (sección 8): `POST /asistencia/marcar-manual` — sin cámara si falla Rekognition, o para sincronizar un marcaje que la PWA guardó localmente sin internet (acepta `capturado_en` con la hora real de captura, para que la puntualidad se calcule con esa hora y no con la de sincronización).
- Vista en tiempo real (sección 6.3): `GET /asistencia/hoy`.
- Reportes básicos (sección 6.4): `GET /reportes/asistencia?inicio=&fin=&empleado_id=` — días trabajados, retardos, faltas, horas trabajadas/extra y % de puntualidad por empleado. La exportación a CSV se arma en el frontend a partir de este JSON.

Pendiente:

- Notificaciones (sección 6.5, v2).
- Exportación a Excel/PDF (hoy solo CSV, generado en el frontend).
- Integración con nómina (sección 8: a futuro).

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

El servidor arranca en `http://localhost:8100` (configurable con `PORT`). Si las credenciales de AWS todavía no están listas, arranca igual (solo avisa en el log) y la colección de un negocio se reintenta sola al enrolar su primer empleado — únicamente `/asistencia/marcar` y `/empleados/:id/enrolar` necesitan Rekognition funcionando.

## Dar de alta un negocio

```bash
curl -X POST http://localhost:8100/negocios \
  -H 'Content-Type: application/json' \
  -d '{"negocio_nombre":"Taquería El Buen Sabor","username":"admin","password":"..."}'
```

Devuelve un token para usar de inmediato; el resto de los endpoints (`/empleados`, `/turnos`, `/asistencia/*`, `/reportes/*`) quedan aislados a ese negocio automáticamente.

## Pruebas

`internal/handlers` tiene pruebas de integración contra un Postgres real (arman el router completo con `httptest`, autenticación real incluida, y cubren el flujo de marcaje, puntualidad/retardo, "hoy", reportes, el aislamiento entre negocios y la sincronización offline con `capturado_en`). Si no hay una base disponible en `postgres://postgres:postgres@localhost:5432/staffattendance_test?sslmode=disable`, se saltan solas; para apuntarlas a otra base usa `TEST_DATABASE_URL`:

```bash
createdb staffattendance_test
TEST_DATABASE_URL="postgres://usuario:pass@localhost:5432/staffattendance_test?sslmode=disable" go test ./...
```
