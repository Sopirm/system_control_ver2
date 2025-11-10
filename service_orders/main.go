package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"service_orders/config"
	"service_orders/events"
	"service_orders/handlers"
	"service_orders/logger"
	"service_orders/repository"

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
	zapLogger.Info("Запуск Service Orders", zap.String("environment", env))

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

	// Инициализация системы событий
	eventPublisher := events.NewInMemoryEventPublisher()
	eventService := events.NewEventService(eventPublisher)
	
	// Настройка graceful shutdown для корректного закрытия системы событий
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Получен сигнал завершения, закрываем сервис...")
		
		if err := eventService.Close(); err != nil {
			log.Printf("Ошибка закрытия сервиса событий: %v", err)
		}
		
		if err := db.Close(); err != nil {
			log.Printf("Ошибка закрытия БД: %v", err)
		}
		
		log.Println("Сервис корректно завершен")
		os.Exit(0)
	}()

	log.Println("Система событий инициализирована")

	// Инициализация репозитория и обработчиков
	orderRepo := repository.NewOrderRepository(db)
	orderHandler := handlers.NewOrderHandler(orderRepo, cfg, eventService)

	// Настройка маршрутов
	router := mux.NewRouter()

	// Маршруты для сервиса заказов
	router.HandleFunc("/v1/orders", orderHandler.CreateOrder).Methods("POST")
	router.HandleFunc("/v1/orders/{id}", orderHandler.GetOrder).Methods("GET")
	router.HandleFunc("/v1/orders", orderHandler.ListOrders).Methods("GET")
	router.HandleFunc("/v1/orders/{id}/status", orderHandler.UpdateOrderStatus).Methods("PUT")
	router.HandleFunc("/v1/orders/{id}/cancel", orderHandler.CancelOrder).Methods("PUT")
	// Совместимость с тестами: поддерживаем также POST для отмены заказа
	router.HandleFunc("/v1/orders/{id}/cancel", orderHandler.CancelOrder).Methods("POST")

	// Дополнительный endpoint для статистики событий (для мониторинга)
	router.HandleFunc("/v1/events/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := eventService.GetStats()
		
		response := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"statistics":    stats,
				"service":       "service_orders",
				"timestamp":     time.Now().Format(time.RFC3339),
				"description":   "Статистика доменных событий",
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// Middleware для логирования
	router.Use(loggingMiddleware)

	zapLogger.Info("Service Orders с системой событий запущен", zap.String("port", cfg.Server.Port))
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
		logger.LogHTTPRequest(r, wrapper.statusCode, r.Method, r.URL.Path, "service_orders")
		
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
