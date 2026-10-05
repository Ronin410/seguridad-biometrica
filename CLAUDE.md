# StaffAttendance (seguridad-biometrica)

Control de asistencia de empleados por reconocimiento facial (AWS Rekognition), multi-negocio, con panel admin y kiosco PWA para tablets. El spec de producto completo está en [`SPEC.md`](./SPEC.md).

Este proyecto usa *spec-driven development* con el plugin `agentes` (agentes `arquitecto`, `desarrollador` y `qa-seguridad`). Las specs viven en `specs/NNN-slug/`; la plantilla es `specs/_plantilla.md`. `SPEC.md` es el documento de producto original, no una spec del flujo.

## Stack

- Backend (`StaffAttendanceBack/`): Go 1.24, Gin, JWT, `lib/pq` sobre PostgreSQL, AWS SDK v2 (Rekognition). Módulo `staffattendance`.
- Frontend (`StaffAttendanceFront/`): React 19 + Vite 7 + Tailwind 4, axios, react-router-dom, react-webcam. PWA (`public/manifest.json`, `public/sw.js`). JavaScript (JSX), sin TypeScript.
- Base de datos: PostgreSQL 16. Las migraciones están en `StaffAttendanceBack/internal/db/migrations.go` y se aplican solas al arrancar el backend.
- Local completo: `docker-compose.yml` en la raíz (db + backend :8100 + frontend :5174).

## Comandos

Backend: ejecutar dentro de `StaffAttendanceBack/`. Frontend: ejecutar dentro de `StaffAttendanceFront/`.

| Acción | Comando |
|---|---|
| Instalar dependencias | Backend: `go mod download` · Frontend: `npm ci` |
| Tests | Backend: `go vet ./... && go test -count=1 ./...` (requiere Postgres, ver abajo) · Frontend: no hay tests |
| Lint | Backend: `go vet ./...` y `gofmt -l .` (debe salir vacío) · Frontend: `npm run lint` |
| Build | Backend: `go build ./...` · Frontend: `npm run build` |
| Correr en local | Todo: `docker compose up --build` en la raíz · Backend: `export $(cat .env \| xargs) && go run .` · Frontend: `npm run dev` |

Los agentes usan **exactamente** estos comandos. Mantenlos actualizados.

### Postgres para los tests (importante)

Las pruebas de `internal/handlers` son de integración contra un Postgres real y **se saltan en silencio** si no hay base disponible: un `ok` en ~0.01 s significa que **no corrieron**. Usa `go test -count=1 -v ./...` y confirma que aparecen líneas `--- PASS`, no `--- SKIP`.

- URL por defecto: `postgres://postgres:postgres@localhost:5432/staffattendance_test?sslmode=disable` (se cambia con `TEST_DATABASE_URL`).
- Con Docker: `docker compose up -d db` y luego `docker compose exec db createdb -U postgres staffattendance_test`.
- Sin Docker (p. ej. sesión en la nube con Postgres instalado): arrancar el servicio (`pg_ctlcluster <versión> main start`), poner contraseña `postgres` al usuario `postgres` y `createdb staffattendance_test`.

Ninguna prueba llama a AWS: `rekognition.FaceRecognizer` es una interfaz y las pruebas usan `fakeRekognition` (`internal/handlers/fake_rekognition_test.go`). Las pruebas nuevas que toquen reconocimiento facial deben usar ese doble, nunca el SDK real.

## Convenciones

- Estructura del backend: `main.go` (arranque y armado del router), `internal/config`, `internal/db` (conexión y migraciones), `internal/handlers` (un archivo por dominio: asistencia, auth, empleados, negocios, reportes, turnos; cada uno registra sus propias rutas y su middleware de rol), `internal/middleware` (JWT y `RequireRol`), `internal/models`, `internal/rekognition`.
- Estructura del frontend: `src/pages/` (una pantalla por archivo), `src/axiosConfig.js` (cliente HTTP con el token), `src/offlineQueue.js` (cola local de marcajes sin internet).
- Estilo: `gofmt` en Go; ESLint (`eslint.config.js`) en el frontend. Código, nombres de dominio, mensajes y documentación en **español**, como el código existente.
- Nombres de ramas: `spec/NNN-slug` para specs. Nunca push directo a `main`.
- Commits: `spec NNN: <tarea>`.
- Tests: backend en `internal/handlers/handlers_test.go` (estilo `TestXxx` con el router completo vía `httptest` y autenticación real). Toda funcionalidad nueva de la API lleva prueba de integración ahí (o en un `*_test.go` del mismo paquete).
- Cuando cambie el estado del producto, actualiza la sección "Estado" de `SPEC.md` y de los README de `StaffAttendanceBack/` y `StaffAttendanceFront/`.

## Despliegue

- Plataforma: Render.
- `main` → producción / ambiente principal.
- Cada PR genera un **preview** en Render (rama de preview: la rama del PR, `spec/NNN-slug`).
- Variables de entorno (`DATABASE_URL`, `JWT_SECRET`, `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `TZ_NEGOCIO`, `SIMILARITY_THRESHOLD`, `PORT`, `VITE_API_URL`): se configuran en el panel de Render, **nunca** en el repo. Solo se versionan los `.env.example`.

## Reglas de seguridad específicas

- **Datos biométricos.** Los rostros son datos personales sensibles (LFPDPPP). No guardar imágenes de rostros en la base, en disco ni en logs; solo el `rekognition_face_id`. No loguear imágenes, tokens ni contraseñas.
- **Aislamiento multi-negocio.** Toda consulta a `usuarios`, `empleados`, `turnos` y `registros_asistencia` debe filtrar por el `negocio_id` del JWT, nunca por uno que mande el cliente. Cada negocio usa su propia colección de Rekognition (`empleados-asistencia-{negocio_id}`). Cualquier endpoint nuevo necesita una prueba de que un negocio no ve ni modifica datos de otro (ver `TestAislamientoEntreNegocios`).
- **Perfiles.** `admin` puede todo; `kiosco` solo `/asistencia/*` y `GET /empleados`. El límite se aplica en el backend con `middleware.RequireRol("admin")` en cada ruta administrativa; ocultarlo en la interfaz no basta. Rutas nuevas son solo admin salvo que la spec diga lo contrario, con prueba en el estilo de `TestPerfilKioscoSoloAsistencia`.
- **Endpoints públicos:** solo `POST /login` y `POST /negocios`. Todo lo demás requiere JWT.
- **Reconocimiento:** respetar `SIMILARITY_THRESHOLD` (95 por defecto); no bajarlo en código. Si Rekognition falla, responder `503` (no `404` "no reconocido").
- **Contraseñas** con hash (bcrypt); `JWT_SECRET` solo desde el entorno. El valor de ejemplo de `.env.example` no debe usarse en producción.
- **Service worker:** cachea solo el shell de la app, nunca respuestas de la API.
- No versionar secretos; usar variables de entorno.
