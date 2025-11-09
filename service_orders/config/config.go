package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config содержит конфигурацию приложения
type Config struct {
	DB     DBConfig
	Server ServerConfig
	JWT    JWTConfig
	Users  UsersServiceConfig
}

// DBConfig содержит конфигурацию базы данных
type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

// ServerConfig содержит конфигурацию сервера
type ServerConfig struct {
	Port string
}

// JWTConfig содержит конфигурацию JWT
type JWTConfig struct {
	Secret string
}

// UsersServiceConfig содержит конфигурацию для взаимодействия с сервисом пользователей
type UsersServiceConfig struct {
	URL string
}

// Load загружает конфигурацию из переменных окружения
func Load() (*Config, error) {
	config := &Config{}

	// Конфигурация БД
	config.DB.Host = getEnv("DB_HOST", "localhost")

	port, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %v", err)
	}
	config.DB.Port = port

	config.DB.Name = getEnv("DB_NAME", "system_control")
	config.DB.User = getEnv("DB_USER", "postgres")
	config.DB.Password = getEnv("DB_PASSWORD", "postgres")

	// Конфигурация сервера
	config.Server.Port = getEnv("SERVER_PORT", "8082")

	// Конфигурация JWT
	config.JWT.Secret = getEnv("JWT_SECRET", "your_secret_key")

	// Конфигурация сервиса пользователей
	config.Users.URL = getEnv("USERS_SERVICE_URL", "http://localhost:8081")

	return config, nil
}

// DSN возвращает строку подключения к PostgreSQL
func (db *DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.User, db.Password, db.Name)
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
