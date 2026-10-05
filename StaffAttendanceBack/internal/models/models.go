package models

import "time"

// Empleado corresponde al modelo de datos de SPEC.md sección 5.
type Empleado struct {
	ID                int       `json:"id"`
	NombreCompleto    string    `json:"nombre_completo"`
	Puesto            string    `json:"puesto"`
	Departamento      string    `json:"departamento"`
	TurnoID           *int      `json:"turno_id"`
	Estado            string    `json:"estado"`
	RekognitionFaceID string    `json:"rekognition_face_id,omitempty"`
	FechaAlta         time.Time `json:"fecha_alta"`
}

// Turno / horario asignable a uno o varios empleados.
type Turno struct {
	ID                       int    `json:"id"`
	Nombre                   string `json:"nombre"`
	HoraEntrada              string `json:"hora_entrada"`
	HoraSalida               string `json:"hora_salida"`
	ToleranciaRetardoMinutos int    `json:"tolerancia_retardo_minutos"`
	DiasAplicables           []int  `json:"dias_aplicables"`
}

// RegistroAsistencia representa un marcaje de entrada o salida.
type RegistroAsistencia struct {
	ID         int       `json:"id"`
	EmpleadoID int       `json:"empleado_id"`
	Fecha      string    `json:"fecha"`
	HoraReal   time.Time `json:"hora_real"`
	Tipo       string    `json:"tipo"`
	Estado     string    `json:"estado"`
	Metodo     string    `json:"metodo"`
}
