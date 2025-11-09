package logger

import (
	"net/http"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger *zap.Logger

// Init инициализирует глобальный логгер
func Init(env string) error {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	} else {
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Кастомный формат времени
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var err error
	globalLogger, err = config.Build()
	if err != nil {
		return err
	}

	return nil
}

// GetLogger возвращает глобальный логгер
func GetLogger() *zap.Logger {
	if globalLogger == nil {
		// Fallback к простому логгеру если не инициализирован
		globalLogger, _ = zap.NewDevelopment()
	}
	return globalLogger
}

// WithRequestID добавляет Request ID к логгеру
func WithRequestID(logger *zap.Logger, requestID string) *zap.Logger {
	if requestID == "" {
		return logger
	}
	return logger.With(zap.String("request_id", requestID))
}

// WithUserContext добавляет пользовательский контекст к логгеру
func WithUserContext(logger *zap.Logger, userID, email string, roles []string) *zap.Logger {
	fields := make([]zap.Field, 0, 3)
	
	if userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}
	if email != "" {
		fields = append(fields, zap.String("user_email", email))
	}
	if len(roles) > 0 {
		fields = append(fields, zap.Strings("user_roles", roles))
	}
	
	if len(fields) > 0 {
		return logger.With(fields...)
	}
	return logger
}

// LogHTTPRequest логирует HTTP запрос с контекстом
func LogHTTPRequest(r *http.Request, statusCode int, method, path, service string) {
	logger := GetLogger()
	
	// Добавляем Request ID если есть
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		logger = WithRequestID(logger, requestID)
	}
	
	// Добавляем пользовательский контекст если есть
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		email := r.Header.Get("X-User-Email")
		var roles []string
		if rolesStr := r.Header.Get("X-User-Roles"); rolesStr != "" {
			// Разбираем роли из строки, разделенной запятыми
			roles = strings.Split(rolesStr, ",")
			for i := range roles {
				roles[i] = strings.TrimSpace(roles[i])
			}
		}
		logger = WithUserContext(logger, userID, email, roles)
	}
	
	logger.Info("HTTP Request",
		zap.String("method", method),
		zap.String("path", path),
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("user_agent", r.UserAgent()),
		zap.String("service", service),
		zap.Int("status_code", statusCode),
	)
}

// LogOrderAction логирует действия с заказами с дополнительным контекстом
func LogOrderAction(r *http.Request, action, orderID, details string, success bool) {
	logger := GetLogger()
	
	// Добавляем Request ID если есть
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		logger = WithRequestID(logger, requestID)
	}
	
	// Добавляем пользовательский контекст если есть
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		email := r.Header.Get("X-User-Email")
		var roles []string
		if rolesStr := r.Header.Get("X-User-Roles"); rolesStr != "" {
			roles = strings.Split(rolesStr, ",")
			for i := range roles {
				roles[i] = strings.TrimSpace(roles[i])
			}
		}
		logger = WithUserContext(logger, userID, email, roles)
	}
	
	fields := []zap.Field{
		zap.String("action", action),
		zap.String("order_id", orderID),
		zap.Bool("success", success),
	}
	
	if details != "" {
		fields = append(fields, zap.String("details", details))
	}
	
	if success {
		logger.Info("Order Action", fields...)
	} else {
		logger.Warn("Order Action Failed", fields...)
	}
}

// LogBusinessEvent логирует бизнес-события (создание заказов, смена статусов и т.д.)
func LogBusinessEvent(r *http.Request, event, entityID, entityType, details string) {
	logger := GetLogger()
	
	// Добавляем Request ID если есть
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		logger = WithRequestID(logger, requestID)
	}
	
	// Добавляем пользовательский контекст если есть
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		email := r.Header.Get("X-User-Email")
		var roles []string
		if rolesStr := r.Header.Get("X-User-Roles"); rolesStr != "" {
			roles = strings.Split(rolesStr, ",")
			for i := range roles {
				roles[i] = strings.TrimSpace(roles[i])
			}
		}
		logger = WithUserContext(logger, userID, email, roles)
	}
	
	fields := []zap.Field{
		zap.String("business_event", event),
		zap.String("entity_id", entityID),
		zap.String("entity_type", entityType),
	}
	
	if details != "" {
		fields = append(fields, zap.String("details", details))
	}
	
	logger.Info("Business Event", fields...)
}

// Sync синхронизирует логгер (должно вызываться при завершении приложения)
func Sync() {
	if globalLogger != nil {
		globalLogger.Sync()
	}
}
