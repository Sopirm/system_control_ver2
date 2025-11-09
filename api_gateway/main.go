package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"golang.org/x/time/rate"
)

const (
	usersServiceURL  = "http://service_users:8081"
	ordersServiceURL = "http://service_orders:8082"
)

var jwtSecret = getEnv("JWT_SECRET", "your_secret_key")

// JWTClaims представляет claims для JWT токена
type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Roles  []string  `json:"roles"`
	jwt.RegisteredClaims
}

var ( // Использование глобальных переменных для примера, в реальном приложении лучше использовать DI
	userProxy  *httputil.ReverseProxy
	orderProxy *httputil.ReverseProxy

	rateLimiter *rate.Limiter
)

func init() {
	// Инициализация прокси-серверов
	userURL, _ := url.Parse(usersServiceURL)
	userProxy = httputil.NewSingleHostReverseProxy(userURL)

	orderURL, _ := url.Parse(ordersServiceURL)
	orderProxy = httputil.NewSingleHostReverseProxy(orderURL)

	// Инициализация ограничителя частоты запросов: 1 запрос в секунду с "burst" в 5 запросов
	rateLimiter = rate.NewLimiter(rate.Every(time.Second), 5)
}

func main() {
	router := mux.NewRouter()

	// CORS Middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Разрешить все источники для простоты, в реальном приложении указать конкретные
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300, // 5 минут
	})

	// Middleware для логирования и X-Request-ID
	router.Use(loggingMiddleware)
	router.Use(requestIDMiddleware)

	// Middleware для ограничения частоты запросов
	router.Use(rateLimitMiddleware)

	// Публичные маршруты (регистрация и вход)
	router.HandleFunc("/v1/users/register", proxyToUsersService).Methods("POST")
	router.HandleFunc("/v1/users/login", proxyToUsersService).Methods("POST")

	// Защищенные маршруты
	subrouter := router.PathPrefix("/v1").Subrouter()
	subrouter.Use(jwtAuthMiddleware) // JWT аутентификация для защищенных маршрутов

	// Маршруты для сервиса пользователей (защищенные)
	subrouter.PathPrefix("/users").Handler(http.HandlerFunc(proxyToUsersService))

	// Маршруты для сервиса заказов (защищенные)
	subrouter.PathPrefix("/orders").Handler(http.HandlerFunc(proxyToOrdersService))

	handledRouter := c.Handler(router)

	fmt.Println("API Gateway запущен на порту :8080")
	log.Fatal(http.ListenAndServe(":8080", handledRouter))
}

// proxyToUsersService проксирует запросы к service_users
func proxyToUsersService(w http.ResponseWriter, r *http.Request) {
	log.Printf("Проксирование запроса к Users Service: %s %s", r.Method, r.URL.Path)
	userProxy.ServeHTTP(w, r)
}

// proxyToOrdersService проксирует запросы к service_orders
func proxyToOrdersService(w http.ResponseWriter, r *http.Request) {
	log.Printf("Проксирование запроса к Orders Service: %s %s", r.Method, r.URL.Path)
	orderProxy.ServeHTTP(w, r)
}

// jwtAuthMiddleware middleware для проверки JWT токена и передачи пользовательского контекста
func jwtAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondWithError(w, http.StatusUnauthorized, "Требуется токен авторизации")
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Неожиданный метод подписи: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			respondWithError(w, http.StatusUnauthorized, fmt.Sprintf("Недействительный токен: %v", err))
			return
		}

		if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
			// Добавляем пользовательский контекст в заголовки для микросервисов
			r.Header.Set("X-User-ID", claims.UserID.String())
			r.Header.Set("X-User-Email", claims.Email)
			r.Header.Set("X-User-Roles", strings.Join(claims.Roles, ","))
			
			log.Printf("Пользователь аутентифицирован: ID=%s, Email=%s, Roles=%v", 
				claims.UserID, claims.Email, claims.Roles)
			
			next.ServeHTTP(w, r)
			return
		}
		respondWithError(w, http.StatusUnauthorized, "Недействительный токен")
	})
}

// rateLimitMiddleware middleware для ограничения частоты запросов
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rateLimiter.Allow() {
			respondWithError(w, http.StatusTooManyRequests, "Слишком много запросов")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware middleware для логирования запросов
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s %s", time.Now().Format("2006-01-02 15:04:05"), r.RemoteAddr, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// requestIDMiddleware middleware для обработки X-Request-ID
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID() // Генерируем новый ID, если отсутствует
			// Опционально: добавить сгенерированный ID в заголовок ответа
			w.Header().Set("X-Request-ID", requestID)
		}
		// Прокидываем X-Request-ID во все исходящие запросы к микросервисам
		r.Header.Set("X-Request-ID", requestID)
		// Можно также сохранить requestID в контекст для использования в последующих обработчиках
		ctx := context.WithValue(r.Context(), "requestID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateRequestID() string {
	// Простая реализация генерации ID. В реальном приложении использовать более надежный метод
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// respondWithError отправляет JSON-ответ с ошибкой
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON отправляет JSON-ответ
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
