package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func Connect(databaseURL string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("abriendo conexión a la base de datos: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("conectando a la base de datos: %w", err)
	}

	return conn, nil
}
