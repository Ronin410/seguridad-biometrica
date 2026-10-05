package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	AWSRegion           string
	JWTSecret           string
	Timezone            string
	SimilarityThreshold float32
}

func Load() Config {
	return Config{
		Port:                getEnv("PORT", "8100"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		AWSRegion:           os.Getenv("AWS_REGION"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		Timezone:            getEnv("TZ_NEGOCIO", "America/Mazatlan"),
		SimilarityThreshold: getEnvFloat("SIMILARITY_THRESHOLD", 95.0),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvFloat(key string, fallback float32) float32 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return fallback
	}
	return float32(parsed)
}
