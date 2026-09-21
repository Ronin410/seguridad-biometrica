# Sistema de Control de Asistencia por Reconocimiento Facial (StaffAttendance)

> Convertido desde el documento original `Spec_Control_de_Asistencia_por_Reconocimiento_Facial.docx`.

## 1. Objetivo del Proyecto

Desarrollar un sistema independiente que utilice reconocimiento facial para registrar la entrada y salida de trabajadores, generando reportes de asistencia, puntualidad y horas trabajadas. El proyecto reutiliza la capa de detección/reconocimiento facial ya construida en GuarderiaBiometric, adaptándola de "padres/tutores de niños" a "empleados".

## 2. Alcance

**Incluye:**

- Registro biométrico facial de empleados (enrolamiento).
- Detección facial en tiempo real para marcar entrada/salida.
- Generación de reportes de asistencia (diarios, semanales, mensuales).
- Cálculo de retardos, faltas y horas trabajadas.
- Panel administrativo para gestión de empleados y turnos.

**No incluye (fuera de alcance v1):**

- Control de acceso físico (apertura de puertas/torniquetes).
- Nómina/cálculo de pago (aunque el reporte puede servir como insumo — ver sección 8).

**Multi-negocio:** confirmado en sección 8 — el sistema es multi-tenant desde el MVP (tabla `negocios`, un negocio por tablet/cuenta).

## 3. Componentes Reutilizables de GuarderiaBiometric

| Componente | Reutilización | Ajustes necesarios |
|---|---|---|
| Integración con AWS Rekognition (IndexFaces / SearchFacesByImage) | Directo | Ninguno — nueva Collection de Rekognition para empleados |
| Backend Go (API, lógica de negocio) | Adaptar | Nuevo dominio `attendance` junto al de `daycare`; se reutiliza el esqueleto de framework (Gin/Echo/Fiber) |
| Frontend React (panel admin) | Adaptar | Reutilizar componentes de tablas, formularios y gráficas; nuevas pantallas para empleados/turnos |
| Proceso de enrolamiento | Adaptar | Cambiar entidad "niño/padre" → "empleado"; misma llamada a IndexFaces |
| Motor de eventos de entrada/salida | Adaptar | Renombrar lógica de "check-in/check-out de niño" a "check-in/check-out de turno" |
| Generador de reportes | Adaptar | Nuevas métricas: retardos, horas extra, ausencias |
| Base de datos / modelos | Adaptar | Nuevo esquema de entidades (ver sección 5); mismo motor (Postgres/MySQL) que ya usa GuarderiaBiometric |

Confirmado: el reconocimiento facial usa AWS Rekognition (colecciones de rostros vía IndexFaces/SearchFacesByImage), con backend en Go y frontend en React — mismo stack que GuarderiaBiometric. Cada negocio tiene su propia colección de Rekognition (`empleados-asistencia-{negocio_id}`), igual que GuarderiaBiometric usa una colección por guardería (`guarderia-{id}`) — así nunca se mezcla biometría entre negocios.

## 4. Actores del Sistema

- **Administrador/RH** — gestiona empleados, turnos, revisa y exporta reportes.
- **Empleado** — solo interactúa con la cámara para marcar entrada/salida (no requiere login).
- **Sistema** — procesa el reconocimiento y genera los registros automáticamente.

## 5. Modelo de Datos (propuesto)

**Negocio** (multi-negocio, sección 8)
- id
- nombre, slug
- rekognition_collection_id (derivado de su id: `empleados-asistencia-{id}`)

**Usuario** (admin/RH que opera el panel)
- id
- negocio_id
- username, password_hash, rol

**Empleado**
- id
- negocio_id
- nombre completo
- puesto / departamento
- horario asignado (hora entrada, hora salida, días laborables)
- estado (activo/inactivo)
- rekognition_face_id (ID devuelto por AWS Rekognition al enrolar, vía IndexFaces)
- fecha de alta

**Turno / Horario**
- id
- negocio_id
- nombre (ej. "Matutino", "Vespertino")
- hora_entrada, hora_salida
- tolerancia_retardo (minutos)
- días aplicables

**RegistroAsistencia**
- id
- negocio_id
- empleado_id
- fecha
- hora_entrada_real
- hora_salida_real
- tipo (entrada/salida)
- estado (puntual/retardo/falta/salida_anticipada)
- método (facial vía AWS Rekognition / manual — respaldo si no hay internet)

**Reporte** (generado, no necesariamente tabla física)
- rango de fechas
- por empleado o global
- totales: días trabajados, retardos, faltas, horas trabajadas, horas extra

## 6. Funcionalidades Principales

### 6.1 Enrolamiento de Empleado
- Capturar rostro del nuevo empleado (varias tomas/ángulos, igual que en GuarderiaBiometric).
- Asociar a datos del empleado (nombre, puesto, horario).
- Confirmar calidad de captura antes de guardar.

### 6.2 Registro de Entrada/Salida
- Cámara en modo continuo o activada por proximidad/botón.
- Reconoce el rostro y determina automáticamente si es entrada o salida (según el último registro del día).
- Muestra retroalimentación visual/sonora (ej. "Bienvenido, Juan — 08:03 am").
- Compara hora real contra horario asignado para marcar puntualidad/retardo.
- Manejo de casos: rostro no reconocido, doble marcaje accidental, empleado sin horario asignado.

### 6.3 Panel Administrativo
- Alta/baja/edición de empleados y horarios.
- Vista de asistencia del día en tiempo real.
- Corrección manual de registros (con motivo/auditoría).

### 6.4 Reportes
- Reporte individual por empleado (rango de fechas).
- Reporte general por día/semana/mes.
- Exportación a Excel/PDF/CSV.
- Indicadores: % puntualidad, total de retardos, total de faltas, horas trabajadas vs. horas programadas.

### 6.5 Notificaciones (opcional v2)
- Alertas de retardo o falta al administrador.
- Resumen semanal automático.

## 7. Requerimientos No Funcionales

- **Precisión del reconocimiento:** delegada a AWS Rekognition; usar el umbral de confianza (`SimilarityThreshold`) recomendado por AWS (ej. ≥95%) para minimizar falsos positivos/negativos.
- **Velocidad:** depende de la latencia a AWS Rekognition (~1 seg típico); considerar red y ancho de banda del sitio.
- **Privacidad/legal:** consentimiento del empleado para uso de datos biométricos (Ley Federal de Protección de Datos Personales en Posesión de los Particulares); datos biométricos procesados y almacenados en la nube de AWS (revisar región y política de retención de Rekognition).
- **Disponibilidad:** requiere conexión a internet para el reconocimiento (a diferencia de un modelo local). La PWA guarda localmente los marcajes manuales cuando no hay conexión y los sincroniza al reconectar (sección 8).
- **Costo:** Rekognition cobra por llamada (IndexFaces, SearchFacesByImage) — con multi-negocio, estimar volumen mensual de marcajes por negocio para proyectar costo.
- **Escalabilidad:** multi-negocio desde el MVP (sección 8) — cada negocio con su propia colección de Rekognition y sus datos aislados por `negocio_id`.

## 8. Pendientes a Confirmar Antes de Iniciar Desarrollo

> Respondidas por el negocio.

1. **¿Mismo dispositivo o hardware nuevo?** → Es una **PWA** que correrá en **distintas tablets** (no un dispositivo único). Necesita internet para llegar al backend y a AWS Rekognition; se instala como app (manifest + service worker) para poder abrirse en modo kiosco en cualquier tablet.
2. **¿Multi-negocio?** → **Sí.** El sistema es multi-negocio desde ya: tabla `negocios`, cada `usuario` pertenece a un negocio (`negocio_id` en el JWT), todas las consultas de empleados/turnos/asistencia/reportes se filtran por ese negocio, y cada negocio tiene su propia colección de Rekognition (`empleados-asistencia-{negocio_id}`) para no mezclar biometría entre negocios — mismo patrón que GuarderiaBiometric usa por guardería.
3. **¿Integración con nómina?** → No por ahora, queda **a futuro**. El reporte se expone vía API y exportación CSV como insumo manual.
4. **¿Qué hacer sin internet?** → Dos escenarios distintos:
   - Falla Rekognition pero la tablet sí llega al backend: ya cubierto con `POST /asistencia/marcar-manual` (un admin registra el marcaje sin cámara).
   - Se cae el internet por completo (la tablet no llega ni al backend): la PWA guarda el marcaje manual **localmente en el dispositivo** (con la hora real de captura) y lo sincroniza solo cuando vuelve la conexión, mostrando "guardado localmente, pendiente de sincronizar" mientras tanto. No se bloquea el registro de asistencia por un corte de internet.

## 9. Siguientes Pasos

1. Confirmar pendientes de la sección 8.
2. Extraer y documentar el módulo facial reutilizable de GuarderiaBiometric.
3. Definir stack tecnológico final (reutilizando el mayor % posible del proyecto original).
4. Diseñar esquema de base de datos definitivo.
5. Construir MVP: enrolamiento + registro de entrada/salida + reporte básico.

## 10. Estado de este repositorio

Este repositorio contiene el **MVP** de StaffAttendance descrito en la sección 9, ya multi-negocio (ver `StaffAttendanceBack/` y `StaffAttendanceFront/`):

- Estructura de proyecto (backend Go + frontend React/PWA), calcada de GuarderiaBiometric.
- **Multi-negocio** (sección 8): tabla `negocios`, alta de negocio + admin en un solo paso (`POST /negocios` / pantalla "Crear cuenta" del login), `negocio_id` en el JWT, y todos los datos (empleados, turnos, asistencia, reportes) aislados por negocio — con su propia colección de Rekognition cada uno.
- Modelo de datos de la sección 5 implementado como migraciones.
- Autenticación JWT, CRUD de empleados/turnos, enrolamiento facial (sección 6.1).
- Marcaje de entrada/salida con cálculo automático de puntualidad/retardo (sección 6.2).
- Respaldo manual (sección 8): sin cámara si falla Rekognition, o para sincronizar marcajes que la PWA guardó localmente cuando la tablet se quedó sin internet — con la hora real de captura, no la de sincronización.
- PWA instalable (manifest + service worker) para correr en distintas tablets, con caché del shell para que el kiosco abra sin conexión.
- Vista de asistencia del día en tiempo real (sección 6.3).
- Reportes por empleado/periodo con exportación a CSV (sección 6.4).
- Pruebas de integración contra Postgres real: flujo completo, puntualidad/retardo, aislamiento entre negocios, y sincronización offline.

**Aún no implementado:**

- Notificaciones (sección 6.5, v2).
- Exportación a Excel/PDF (hoy solo CSV).
- Integración con nómina (sección 8: a futuro).

Ver `StaffAttendanceBack/README.md` y `StaffAttendanceFront/README.md` para instrucciones de ejecución y pruebas.
