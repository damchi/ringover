package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort    string
	DbHost     string
	DbPort     string
	DbUser     string
	DbPassword string
	DbName     string
	DbParams   string
}

func LoadConfig() *Config {
	_ = godotenv.Load(".env")

	return &Config{
		AppPort:    getEnv("APP_PORT", "8080"),
		DbHost:     getEnv("MYSQL_HOST", "db"),
		DbPort:     getEnv("MYSQL_PORT", "3306"),
		DbUser:     getEnv("MYSQL_USER", "ringover"),
		DbPassword: getEnv("MYSQL_PASSWORD", "ringover"),
		DbName:     getEnv("MYSQL_DATABASE", "ringover"),
		DbParams:   getEnv("MYSQL_PARAMS", "parseTime=true&multiStatements=true"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
