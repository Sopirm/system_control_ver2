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

	"api_gateway/logger"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

var (
    // URL сервисов берём из переменных окружения, чтобы избежать ошибок проксирования
    usersServiceURL  = getEnv("USERS_SERVICE_URL", "http://service_users:8081")
    ordersServiceURL = getEnv("ORDERS_SERVICE_URL", "http://service_orders:8082")
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
	// Инициализация логгера
	env := getEnv("ENVIRONMENT", "development")
	if err := logger.Init(env); err != nil {
		log.Fatalf("Ошибка инициализации логгера: %v", err)
	}
	defer logger.Sync()

	zapLogger := logger.GetLogger()
	zapLogger.Info("Запуск API Gateway", zap.String("environment", env))
	// Логируем целевые сервисы для диагностики
	zapLogger.Info("Конфигурация upstream сервисов",
		zap.String("users_service_url", usersServiceURL),
		zap.String("orders_service_url", ordersServiceURL),
	)

	router := mux.NewRouter()

	// CORS Middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Разрешить все источники для простоты, в реальном приложении указать конкретные
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300, // 5 минут
	})

	// Middleware для X-Request-ID (должен быть первым)
	router.Use(requestIDMiddleware)

	// Middleware для логирования
	router.Use(loggingMiddleware)

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

	zapLogger.Info("API Gateway запущен на порту :8080")

	if err := http.ListenAndServe(":8080", handledRouter); err != nil {
		zapLogger.Fatal("Ошибка запуска HTTP сервера", zap.Error(err))
	}
}

// proxyToUsersService проксирует запросы к service_users
func proxyToUsersService(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")

	logger.LogServiceCall(requestID, "api_gateway", "service_users", r.URL.Path, true, nil)

	userProxy.ServeHTTP(w, r)
}

// proxyToOrdersService проксирует запросы к service_orders
func proxyToOrdersService(w http.ResponseWriter, r *http.Request) {
	requestID := r.Header.Get("X-Request-ID")

	logger.LogServiceCall(requestID, "api_gateway", "service_orders", r.URL.Path, true, nil)

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

			// Структурированное логирование аутентификации
			log := logger.GetLogger()
			requestID := r.Header.Get("X-Request-ID")
			if requestID != "" {
				log = logger.WithRequestID(log, requestID)
			}
			log = logger.WithUserContext(log, claims.UserID.String(), claims.Email, claims.Roles)
			log.Info("Пользователь успешно аутентифицирован",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)

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
			// Логируем превышение лимита с контекстом
			log := logger.GetLogger()
			if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
				log = logger.WithRequestID(log, requestID)
			}
			log.Warn("Rate limit exceeded",
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)

			respondWithError(w, http.StatusTooManyRequests, "Слишком много запросов")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware middleware для логирования запросов
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Создаем wrapper для захвата статус кода
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		start := time.Now()
		next.ServeHTTP(wrapper, r)
		duration := time.Since(start)

		// Используем новый структурированный логгер
		logger.LogHTTPRequest(r, wrapper.statusCode, r.Method, r.URL.Path, "api_gateway")

		// Дополнительные метрики
		log := logger.GetLogger()
		if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
			log = logger.WithRequestID(log, requestID)
		}

		log.Info("Request completed",
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
	// Логируем ошибки с уровнем ERROR если код >= 500, иначе WARN
	log := logger.GetLogger()
	if code >= 500 {
		log.Error("HTTP Error Response",
			zap.Int("status_code", code),
			zap.String("error_message", message),
		)
	} else {
		log.Warn("HTTP Error Response",
			zap.Int("status_code", code),
			zap.String("error_message", message),
		)
	}

	respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON отправляет JSON-ответ
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log := logger.GetLogger()
		log.Error("Failed to marshal JSON response", zap.Error(err))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
		return
	}

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
