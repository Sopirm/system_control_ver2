package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"service_users/config"
	"service_users/handlers"
	"service_users/logger"
	"service_users/repository"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	// Инициализация логгера
	env := getEnv("ENVIRONMENT", "development")
	if err := logger.Init(env); err != nil {
		log.Fatalf("Ошибка инициализации логгера: %v", err)
	}
	defer logger.Sync()

	zapLogger := logger.GetLogger()
	zapLogger.Info("Запуск Service Users", zap.String("environment", env))

	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		zapLogger.Fatal("Ошибка загрузки конфигурации", zap.Error(err))
	}

	// Подключение к базе данных
	db, err := sql.Open("postgres", cfg.DB.DSN())
	if err != nil {
		zapLogger.Fatal("Ошибка подключения к базе данных", zap.Error(err))
	}
	defer db.Close()

	// Проверка подключения к БД
	if err := db.Ping(); err != nil {
		zapLogger.Fatal("Ошибка проверки подключения к БД", zap.Error(err))
	}

	zapLogger.Info("Успешное подключение к базе данных")

	// Инициализация репозитория и обработчиков
	userRepo := repository.NewUserRepository(db)
	userHandler := handlers.NewUserHandler(userRepo, cfg)

	// Настройка маршрутов
	router := mux.NewRouter()

	// Публичные маршруты
	router.HandleFunc("/v1/users/register", userHandler.RegisterUser).Methods("POST")
	router.HandleFunc("/v1/users/login", userHandler.LoginUser).Methods("POST")

	// Защищенные маршруты
	router.HandleFunc("/v1/users/profile", userHandler.GetUserProfile).Methods("GET")
	router.HandleFunc("/v1/users/profile", userHandler.UpdateUserProfile).Methods("PUT")
	router.HandleFunc("/v1/users", userHandler.ListUsers).Methods("GET")

	// Middleware для логирования
	router.Use(loggingMiddleware)

	zapLogger.Info("Service Users запущен", zap.String("port", cfg.Server.Port))
	log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, router))
}

// loggingMiddleware middleware для логирования запросов
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Создаем wrapper для захвата статус кода
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		start := time.Now()
		next.ServeHTTP(wrapper, r)
		duration := time.Since(start)

		// Используем структурированный логгер
		logger.LogHTTPRequest(r, wrapper.statusCode, r.Method, r.URL.Path, "service_users")

		// Дополнительные метрики
		zapLogger := logger.GetLogger()
		if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
			zapLogger = logger.WithRequestID(zapLogger, requestID)
		}

		zapLogger.Info("Request completed",
			zap.Duration("duration", duration),
			zap.Int64("content_length", r.ContentLength),
		)
	})
}

// responseWrapper для захвата HTTP статус кода
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
