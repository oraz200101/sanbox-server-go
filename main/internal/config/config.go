package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server struct {
		Port int
	}

	DBPostgres struct {
		Host     string
		Port     int
		User     string
		Password string
		Name     string
	}

	DBMongo struct {
		Host string
		Port int
		Name string
	}

	Elasticsearch struct {
		Host string
		Port int
	}

	Kafka struct {
		Host    string
		Port    int
		Topics  []string
		GroupID string
	}

	Environment string
}

func Load() (*Config, error) {
	err := godotenv.Load()

	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Server: struct{ Port int }{
			Port: getEnvInt("API_PORT", 8080),
		},
		DBPostgres: struct {
			Host     string
			Port     int
			User     string
			Password string
			Name     string
		}{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 12218),
			User:     getEnv("DB_USER", "postgres"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     getEnv("DB_NAME", "sandbox"),
		},
		DBMongo: struct {
			Host string
			Port int
			Name string
		}{
			Host: getEnv("MONGO_HOST", "localhost"),
			Port: getEnvInt("MONGO_PORT", 12217),
			Name: getEnv("MONGO_DB", "sandbox"),
		},

		Kafka: struct {
			Host    string
			Port    int
			Topics  []string
			GroupID string
		}{
			Host:    getEnv("KAFKA_HOST", "localhost"),
			Port:    getEnvInt("KAFKA_PORT", 12211),
			Topics:  make([]string, 0),
			GroupID: getEnv("KAFKA_GROUP_ID", "exam-service"),
		},

		Elasticsearch: struct {
			Host string
			Port int
		}{
			Host: getEnv("ELASTICSEARCH_HOST", "localhost"),
			Port: getEnvInt("ELASTICSEARCH_PORT", 12216),
		},

		Environment: getEnv("ENVIRONMENT", "development"),
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	i, _ := strconv.Atoi(val)
	return i
}
