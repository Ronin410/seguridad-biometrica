package config

import "os"

type Config struct {
	Port                  string
	DatabaseURL           string
	AWSRegion             string
	RekognitionCollection string
	JWTSecret             string
}

func Load() Config {
	return Config{
		Port:                  getEnv("PORT", "8100"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		AWSRegion:             os.Getenv("AWS_REGION"),
		RekognitionCollection: getEnv("REKOGNITION_COLLECTION_ID", "empleados-asistencia"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
