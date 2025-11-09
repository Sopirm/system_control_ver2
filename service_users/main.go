package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"service_users/config"
	"service_users/handlers"
	"service_users/repository"

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

	fmt.Printf("Service Users запущен на порту :%s\n", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, router))
}

// loggingMiddleware middleware для логирования запросов
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
