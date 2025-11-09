package events

import (
	"context"
	"fmt"
	"net/http"

	"service_orders/models"

	"github.com/google/uuid"
)

// EventService сервис для работы с доменными событиями
type EventService struct {
	publisher EventPublisher
}

// NewEventService создает новый сервис событий
func NewEventService(publisher EventPublisher) *EventService {
	service := &EventService{
		publisher: publisher,
	}
	
	// Регистрируем стандартные обработчики
	service.registerDefaultHandlers()
	
	return service
}

// registerDefaultHandlers регистрирует стандартные обработчики событий
func (s *EventService) registerDefaultHandlers() {
	// Регистрируем базовые обработчики для логирования
	for eventType, handler := range DefaultEventHandlers {
		if err := s.publisher.Subscribe(eventType, handler); err != nil {
			fmt.Printf("Ошибка регистрации базового обработчика для %s: %v\n", eventType, err)
		}
	}
	
	// Регистрируем дополнительные обработчики
	handlers := []struct {
		name    string
		handler EventHandler
	}{
		{"analytics", AnalyticsEventHandler},
		{"notifications", NotificationEventHandler},
		{"audit", AuditEventHandler},
	}
	
	for _, h := range handlers {
		// Подписываемся на все типы событий
		for _, eventType := range []EventType{OrderCreatedEvent, OrderStatusUpdatedEvent} {
			if err := s.publisher.Subscribe(eventType, h.handler); err != nil {
				fmt.Printf("Ошибка регистрации %s обработчика для %s: %v\n", h.name, eventType, err)
			}
		}
	}
	
	fmt.Println("Все обработчики событий зарегистрированы: базовые, аналитика, уведомления, аудит")
}

// PublishOrderCreated публикует событие создания заказа
func (s *EventService) PublishOrderCreated(ctx context.Context, order *models.Order, r *http.Request) error {
	metadata := s.extractMetadata(r, "order.create")
	event := NewOrderCreatedEvent(order, metadata)
	
	return s.publisher.Publish(ctx, event)
}

// PublishOrderStatusUpdated публикует событие обновления статуса заказа
func (s *EventService) PublishOrderStatusUpdated(ctx context.Context, orderID, userID, updatedBy uuid.UUID, 
	oldStatus, newStatus models.OrderStatus, r *http.Request) error {
	
	metadata := s.extractMetadata(r, "order.status.update")
	event := NewOrderStatusUpdatedEvent(orderID, userID, updatedBy, oldStatus, newStatus, metadata)
	
	return s.publisher.Publish(ctx, event)
}

// PublishOrderCancelled публикует событие отмены заказа (специальный случай обновления статуса)
func (s *EventService) PublishOrderCancelled(ctx context.Context, orderID, userID, cancelledBy uuid.UUID, 
	oldStatus models.OrderStatus, r *http.Request) error {
	
	return s.PublishOrderStatusUpdated(ctx, orderID, userID, cancelledBy, oldStatus, models.OrderStatusCancelled, r)
}

// extractMetadata извлекает метаданные из HTTP запроса
func (s *EventService) extractMetadata(r *http.Request, operation string) Metadata {
	if r == nil {
		return Metadata{
			Source: "service_orders",
		}
	}
	
	return Metadata{
		RequestID:     r.Header.Get("X-Request-ID"),
		UserAgent:     r.UserAgent(),
		IPAddress:     r.RemoteAddr,
		Source:        "service_orders",
		CorrelationID: generateCorrelationID(r, operation),
	}
}

// generateCorrelationID генерирует ID корреляции для связи событий
func generateCorrelationID(r *http.Request, operation string) string {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		return operation
	}
	return fmt.Sprintf("%s-%s", requestID, operation)
}

// AddCustomHandler добавляет кастомный обработчик событий
func (s *EventService) AddCustomHandler(eventType EventType, handler EventHandler) error {
	return s.publisher.Subscribe(eventType, handler)
}

// Close закрывает сервис событий
func (s *EventService) Close() error {
	return s.publisher.Close()
}

// GetStats возвращает статистику событий
func (s *EventService) GetStats() map[string]int64 {
	return GetEventStats()
}
