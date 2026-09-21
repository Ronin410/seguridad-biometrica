# seguridad-biometrica

## StaffAttendance — Control de Asistencia por Reconocimiento Facial

Spec completo en [`SPEC.md`](./SPEC.md).

Reutiliza la capa de reconocimiento facial (AWS Rekognition) y el stack Go + React de [GuarderiaBiometric](https://github.com/Ronin410/GuarderiaBiometric), adaptado de "padres/niños" a "empleados".

- [`StaffAttendanceBack/`](./StaffAttendanceBack) — backend Go (Gin + Rekognition + Postgres).
- [`StaffAttendanceFront/`](./StaffAttendanceFront) — frontend React (Vite + Tailwind).

Cada carpeta tiene su propio README con instrucciones de configuración y ejecución.
