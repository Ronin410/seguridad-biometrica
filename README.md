# seguridad-biometrica

## StaffAttendance — Control de Asistencia por Reconocimiento Facial

Spec completo en [`SPEC.md`](./SPEC.md).

Reutiliza la capa de reconocimiento facial (AWS Rekognition) y el stack Go + React de [GuarderiaBiometric](https://github.com/Ronin410/GuarderiaBiometric), adaptado de "padres/niños" a "empleados".

- [`StaffAttendanceBack/`](./StaffAttendanceBack) — backend Go (Gin + Rekognition + Postgres).
- [`StaffAttendanceFront/`](./StaffAttendanceFront) — frontend React (Vite + Tailwind).

Cada carpeta tiene su propio README con instrucciones de configuración y ejecución sin Docker.

## Levantarlo localmente con Docker

Requiere [Docker](https://docs.docker.com/get-docker/) (con Docker Compose, incluido desde Docker Desktop / Docker Engine recientes).

```bash
git clone https://github.com/Ronin410/seguridad-biometrica
cd seguridad-biometrica
git checkout claude/spec-generation-wteekh

cp .env.example .env   # opcional: agrega tus credenciales de AWS

docker compose up --build
```

Esto levanta tres contenedores: Postgres, el backend (`http://localhost:8100`) y el frontend (`http://localhost:5174`). El backend corre sus migraciones solo al arrancar. Abre **http://localhost:5174** y, en el login, usa "¿Negocio nuevo? Crear cuenta" para dar de alta tu primer negocio.

Sin credenciales de AWS en `.env`, todo funciona excepto el enrolamiento facial y el marcaje por cámara (que responden 503) — usa "Sin conexión: registrar manualmente" en el Kiosco mientras tanto.

Para parar todo: `docker compose down` (agrega `-v` si además quieres borrar los datos de Postgres).
