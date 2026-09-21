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
- Nómina/cálculo de pago (aunque el reporte puede servir como insumo).
- Multi-sucursal (se deja como consideración futura en el modelo de datos).

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

Confirmado: el reconocimiento facial usa AWS Rekognition (colecciones de rostros vía IndexFaces/SearchFacesByImage), con backend en Go y frontend en React — mismo stack que GuarderiaBiometric. Se creará una colección de Rekognition separada (ej. `empleados-asistencia`) para no mezclar biometría de empleados con la de padres/niños.

## 4. Actores del Sistema

- **Administrador/RH** — gestiona empleados, turnos, revisa y exporta reportes.
- **Empleado** — solo interactúa con la cámara para marcar entrada/salida (no requiere login).
- **Sistema** — procesa el reconocimiento y genera los registros automáticamente.

## 5. Modelo de Datos (propuesto)

**Empleado**
- id
- nombre completo
- puesto / departamento
- horario asignado (hora entrada, hora salida, días laborables)
- estado (activo/inactivo)
- rekognition_face_id (ID devuelto por AWS Rekognition al enrolar, vía IndexFaces)
- fecha de alta

**Turno / Horario**
- id
- nombre (ej. "Matutino", "Vespertino")
- hora_entrada, hora_salida
- tolerancia_retardo (minutos)
- días aplicables

**RegistroAsistencia**
- id
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
- **Disponibilidad:** requiere conexión a internet para el reconocimiento (a diferencia de un modelo local); definir un flujo de respaldo (registro manual) para cortes de conexión.
- **Costo:** Rekognition cobra por llamada (IndexFaces, SearchFacesByImage) — estimar volumen mensual de marcajes para proyectar costo, relevante si se planea multi-negocio.
- **Escalabilidad:** pensado inicialmente para un negocio pequeño/mediano (ej. la taquería), pero con modelo de datos y colecciones de Rekognition que permiten crecer a multi-negocio.

## 8. Pendientes a Confirmar Antes de Iniciar Desarrollo

> Resueltos como decisiones de trabajo para poder avanzar con el MVP. Son supuestos razonables, no respuestas confirmadas del negocio — revisar y ajustar si no aplican.

1. **¿Mismo dispositivo o hardware nuevo?** → Se asume el mismo tipo de dispositivo (tablet/kiosco) con conexión a internet, igual que GuarderiaBiometric. El frontend no depende de hardware específico (cámara vía navegador con `react-webcam`), así que no bloquea el desarrollo.
2. **¿Multi-negocio?** → Solo uso interno por ahora (v1, una sola taquería). El modelo de datos ya deja `negocio_id` en `empleados`, `turnos` y `registros_asistencia` (default `1`) para no tener que migrar si más adelante se vende a otros negocios.
3. **¿Integración con nómina?** → No en v1. El reporte se expone vía API y exportación CSV para que pueda usarse como insumo manual de un sistema de nómina externo; no hay integración directa.
4. **¿Qué hacer sin internet?** → Registro manual de respaldo que se reconcilia después (no se bloquea el sistema). Se agrega `POST /asistencia/marcar-manual` para que un administrador registre entrada/salida sin pasar por Rekognition, marcado con `metodo = 'manual'` en `registros_asistencia`.

## 9. Siguientes Pasos

1. Confirmar pendientes de la sección 8.
2. Extraer y documentar el módulo facial reutilizable de GuarderiaBiometric.
3. Definir stack tecnológico final (reutilizando el mayor % posible del proyecto original).
4. Diseñar esquema de base de datos definitivo.
5. Construir MVP: enrolamiento + registro de entrada/salida + reporte básico.

## 10. Estado de este repositorio

Este repositorio contiene el **MVP** de StaffAttendance descrito en la sección 9 (ver `StaffAttendanceBack/` y `StaffAttendanceFront/`):

- Estructura de proyecto (backend Go + frontend React), calcada de GuarderiaBiometric.
- Modelo de datos de la sección 5 implementado como migraciones (`usuarios`, `empleados`, `turnos`, `registros_asistencia`).
- Cliente de AWS Rekognition, autenticación JWT y CRUD de empleados/turnos.
- Enrolamiento facial de empleados (sección 6.1).
- Marcaje de entrada/salida con cálculo automático de puntualidad/retardo (sección 6.2), más el respaldo manual sin cámara de la sección 8.
- Vista de asistencia del día en tiempo real (sección 6.3).
- Reportes por empleado/periodo con exportación a CSV (sección 6.4).
- Pruebas de integración contra Postgres real cubriendo el flujo completo (turnos → empleado → marcaje → hoy → reporte).

**Aún no implementado:**

- Notificaciones (sección 6.5, v2).
- Exportación a Excel/PDF (hoy solo CSV).
- Integración con nómina y soporte multi-negocio más allá del campo `negocio_id` en el modelo de datos (ver decisiones de la sección 8).

Ver `StaffAttendanceBack/README.md` y `StaffAttendanceFront/README.md` para instrucciones de ejecución y pruebas.
