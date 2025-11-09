package events

import (
	"encoding/json"
	"time"

	"service_orders/models"

	"github.com/google/uuid"
)

// EventType представляет тип доменного события
type EventType string

const (
	// OrderCreatedEvent событие создания заказа
	OrderCreatedEvent EventType = "order.created"
	// OrderStatusUpdatedEvent событие обновления статуса заказа
	OrderStatusUpdatedEvent EventType = "order.status.updated"
)

// DomainEvent представляет базовую структуру доменного события
type DomainEvent struct {
	ID          uuid.UUID   `json:"id"`
	Type        EventType   `json:"type"`
	AggregateID uuid.UUID   `json:"aggregate_id"` // ID заказа
	UserID      uuid.UUID   `json:"user_id"`      // ID пользователя
	Timestamp   time.Time   `json:"timestamp"`
	Version     int         `json:"version"`      // Версия события для совместимости
	Data        interface{} `json:"data"`
	Metadata    Metadata    `json:"metadata"`
}

// Metadata содержит дополнительную информацию о событии
type Metadata struct {
	RequestID   string `json:"request_id,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
	IPAddress   string `json:"ip_address,omitempty"`
	Source      string `json:"source"`      // Источник события (service_orders)
	CorrelationID string `json:"correlation_id,omitempty"` // Для трассировки связанных событий
}

// OrderCreatedEventData данные события создания заказа
type OrderCreatedEventData struct {
	OrderID   uuid.UUID          `json:"order_id"`
	UserID    uuid.UUID          `json:"user_id"`
	Items     []models.OrderItem `json:"items"`
	TotalSum  float64            `json:"total_sum"`
	Status    models.OrderStatus `json:"status"`
	CreatedAt time.Time          `json:"created_at"`
}

// OrderStatusUpdatedEventData данные события обновления статуса заказа
type OrderStatusUpdatedEventData struct {
	OrderID     uuid.UUID          `json:"order_id"`
	UserID      uuid.UUID          `json:"user_id"`
	OldStatus   models.OrderStatus `json:"old_status"`
	NewStatus   models.OrderStatus `json:"new_status"`
	UpdatedAt   time.Time          `json:"updated_at"`
	UpdatedBy   uuid.UUID          `json:"updated_by"` // Кто обновил (может отличаться от владельца)
}

// NewOrderCreatedEvent создает новое событие создания заказа
func NewOrderCreatedEvent(order *models.Order, metadata Metadata) *DomainEvent {
	return &DomainEvent{
		ID:          uuid.New(),
		Type:        OrderCreatedEvent,
		AggregateID: order.ID,
		UserID:      order.UserID,
		Timestamp:   time.Now(),
		Version:     1,
		Data: OrderCreatedEventData{
			OrderID:   order.ID,
			UserID:    order.UserID,
			Items:     order.Items,
			TotalSum:  order.TotalSum,
			Status:    order.Status,
			CreatedAt: order.CreatedAt,
		},
		Metadata: metadata,
	}
}

// NewOrderStatusUpdatedEvent создает новое событие обновления статуса заказа
func NewOrderStatusUpdatedEvent(orderID, userID, updatedBy uuid.UUID, oldStatus, newStatus models.OrderStatus, metadata Metadata) *DomainEvent {
	return &DomainEvent{
		ID:          uuid.New(),
		Type:        OrderStatusUpdatedEvent,
		AggregateID: orderID,
		UserID:      userID,
		Timestamp:   time.Now(),
		Version:     1,
		Data: OrderStatusUpdatedEventData{
			OrderID:   orderID,
			UserID:    userID,
			OldStatus: oldStatus,
			NewStatus: newStatus,
			UpdatedAt: time.Now(),
			UpdatedBy: updatedBy,
		},
		Metadata: metadata,
	}
}

// ToJSON сериализует событие в JSON
func (e *DomainEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON десериализует событие из JSON
func FromJSON(data []byte) (*DomainEvent, error) {
	var event DomainEvent
	err := json.Unmarshal(data, &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// GetEventName возвращает человекочитаемое имя события
func (t EventType) GetEventName() string {
	switch t {
	case OrderCreatedEvent:
		return "Заказ создан"
	case OrderStatusUpdatedEvent:
		return "Статус заказа обновлен"
	default:
		return "Неизвестное событие"
	}
}
