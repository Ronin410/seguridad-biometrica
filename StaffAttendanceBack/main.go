package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"staffattendance/internal/config"
	"staffattendance/internal/db"
	"staffattendance/internal/handlers"
	"staffattendance/internal/middleware"
	"staffattendance/internal/rekognition"
)

func main() {
	cfg := config.Load()

	conn, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("No se pudo conectar a la base de datos: %v", err)
	}
	defer conn.Close()

	if err := db.RunMigrations(conn); err != nil {
		log.Fatalf("No se pudieron ejecutar las migraciones: %v", err)
	}

	rekClient, err := rekognition.NewClient(context.Background(), cfg.AWSRegion, cfg.RekognitionCollection)
	if err != nil {
		log.Fatalf("No se pudo inicializar el cliente de Rekognition: %v", err)
	}
	if err := rekClient.EnsureCollection(context.Background()); err != nil {
		log.Fatalf("No se pudo preparar la colección de Rekognition: %v", err)
	}

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	jwtSecret := []byte(cfg.JWTSecret)

	authHandler := handlers.NewAuthHandler(conn, jwtSecret)
	authHandler.Register(router)

	protected := router.Group("/")
	protected.Use(middleware.Auth(jwtSecret))

	handlers.NewEmpleadosHandler(conn, rekClient).Register(protected)
	handlers.NewTurnosHandler(conn).Register(protected)
	handlers.NewAsistenciaHandler(conn, rekClient).Register(protected)

	log.Printf("StaffAttendance escuchando en el puerto %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
