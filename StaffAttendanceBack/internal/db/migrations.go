package db

import (
	"database/sql"
	"fmt"
	"log"
)

// RunMigrations crea el esquema descrito en SPEC.md sección 5. El sistema es
// multi-negocio desde el MVP (sección 8): todo cuelga de `negocios`, y cada
// negocio tiene su propia colección de Rekognition.
func RunMigrations(conn *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS negocios (
			id SERIAL PRIMARY KEY,
			nombre VARCHAR(150) NOT NULL,
			slug VARCHAR(60) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS usuarios (
			id SERIAL PRIMARY KEY,
			negocio_id INTEGER NOT NULL REFERENCES negocios(id) ON DELETE CASCADE,
			username VARCHAR(50) UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			rol VARCHAR(20) NOT NULL DEFAULT 'admin' CHECK (rol IN ('admin', 'rh')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS turnos (
			id SERIAL PRIMARY KEY,
			negocio_id INTEGER NOT NULL REFERENCES negocios(id) ON DELETE CASCADE,
			nombre VARCHAR(100) NOT NULL,
			hora_entrada TIME NOT NULL,
			hora_salida TIME NOT NULL,
			tolerancia_retardo_minutos INTEGER NOT NULL DEFAULT 10,
			dias_aplicables SMALLINT[] NOT NULL DEFAULT '{1,2,3,4,5}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS empleados (
			id SERIAL PRIMARY KEY,
			negocio_id INTEGER NOT NULL REFERENCES negocios(id) ON DELETE CASCADE,
			nombre_completo VARCHAR(150) NOT NULL,
			puesto VARCHAR(100),
			departamento VARCHAR(100),
			turno_id INTEGER REFERENCES turnos(id) ON DELETE SET NULL,
			estado VARCHAR(20) NOT NULL DEFAULT 'activo' CHECK (estado IN ('activo', 'inactivo')),
			rekognition_face_id VARCHAR(255) UNIQUE,
			fecha_alta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS registros_asistencia (
			id SERIAL PRIMARY KEY,
			negocio_id INTEGER NOT NULL REFERENCES negocios(id) ON DELETE CASCADE,
			empleado_id INTEGER NOT NULL REFERENCES empleados(id) ON DELETE CASCADE,
			fecha DATE NOT NULL DEFAULT CURRENT_DATE,
			hora_real TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			tipo VARCHAR(20) NOT NULL CHECK (tipo IN ('entrada', 'salida')),
			estado VARCHAR(20) CHECK (estado IN ('puntual', 'retardo', 'falta', 'salida_anticipada')),
			metodo VARCHAR(20) NOT NULL DEFAULT 'facial' CHECK (metodo IN ('facial', 'manual'))
		);`,

		`CREATE INDEX IF NOT EXISTS idx_registros_asistencia_empleado_fecha ON registros_asistencia (empleado_id, fecha);`,
		`CREATE INDEX IF NOT EXISTS idx_empleados_negocio ON empleados (negocio_id);`,
		`CREATE INDEX IF NOT EXISTS idx_turnos_negocio ON turnos (negocio_id);`,
	}

	for _, query := range queries {
		if _, err := conn.Exec(query); err != nil {
			return fmt.Errorf("ejecutando migración: %w", err)
		}
	}

	log.Println("Migraciones de StaffAttendance completadas")
	return nil
}
