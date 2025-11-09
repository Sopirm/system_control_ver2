package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"service_orders/config"
	"service_orders/handlers"
	"service_orders/repository"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Подключение к базе данных
	db, err := sql.Open("postgres", cfg.DB.DSN())
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Проверка подключения к БД
	if err := db.Ping(); err != nil {
		log.Fatalf("Ошибка проверки подключения к БД: %v", err)
	}

	log.Println("Успешное подключение к базе данных")

	// Инициализация репозитория и обработчиков
	orderRepo := repository.NewOrderRepository(db)
	orderHandler := handlers.NewOrderHandler(orderRepo, cfg)

	// Настройка маршрутов
	router := mux.NewRouter()

	// Маршруты для сервиса заказов
	router.HandleFunc("/v1/orders", orderHandler.CreateOrder).Methods("POST")
	router.HandleFunc("/v1/orders/{id}", orderHandler.GetOrder).Methods("GET")
	router.HandleFunc("/v1/orders", orderHandler.ListOrders).Methods("GET")
	router.HandleFunc("/v1/orders/{id}/status", orderHandler.UpdateOrderStatus).Methods("PUT")
	router.HandleFunc("/v1/orders/{id}/cancel", orderHandler.CancelOrder).Methods("PUT")

	// Middleware для логирования
	router.Use(loggingMiddleware)

	fmt.Printf("Service Orders запущен на порту :%s\n", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, router))
}

// loggingMiddleware middleware для логирования запросов
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		userID := r.Header.Get("X-User-ID")
		
		log.Printf("[%s] %s %s | User: %s | Request-ID: %s", 
			r.Method, r.URL.Path, r.RemoteAddr, userID, requestID)
		
		next.ServeHTTP(w, r)
	})
}
