package models

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus представляет статус заказа
type OrderStatus string

const (
	OrderStatusCreated   OrderStatus = "создан"
	OrderStatusInWork    OrderStatus = "в работе"
	OrderStatusCompleted OrderStatus = "выполнен"
	OrderStatusCancelled OrderStatus = "отменён"
)

// OrderItem представляет позицию в заказе
type OrderItem struct {
	Product  string  `json:"product" validate:"required"`
	Quantity int     `json:"quantity" validate:"required,min=1"`
	Price    float64 `json:"price" validate:"required,min=0"`
}

// Order представляет модель заказа
type Order struct {
	ID        uuid.UUID   `json:"id" db:"id"`
	UserID    uuid.UUID   `json:"user_id" db:"user_id"`
	Items     []OrderItem `json:"items" db:"items"`
	Status    OrderStatus `json:"status" db:"status"`
	TotalSum  float64     `json:"total_sum" db:"total_sum"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt time.Time   `json:"updated_at" db:"updated_at"`
}

// CreateOrderRequest представляет запрос на создание заказа
type CreateOrderRequest struct {
	Items []OrderItem `json:"items" validate:"required,min=1,dive"`
}

// UpdateOrderStatusRequest представляет запрос на обновление статуса заказа
type UpdateOrderStatusRequest struct {
	Status OrderStatus `json:"status" validate:"required,oneof=создан 'в работе' выполнен отменён"`
}

// ListOrdersRequest представляет параметры для получения списка заказов
type ListOrdersRequest struct {
	Limit  int         `json:"limit" validate:"min=1,max=100"`
	Offset int         `json:"offset" validate:"min=0"`
	Status OrderStatus `json:"status"`
	Sort   string      `json:"sort" validate:"omitempty,oneof=created_at updated_at total_sum"`
	Order  string      `json:"order" validate:"omitempty,oneof=asc desc"`
}

// ListOrdersResponse представляет ответ со списком заказов
type ListOrdersResponse struct {
	Orders []Order `json:"orders"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// CalculateTotal вычисляет общую стоимость заказа
func (o *Order) CalculateTotal() {
	total := 0.0
	for _, item := range o.Items {
		total += item.Price * float64(item.Quantity)
	}
	o.TotalSum = total
}

// CanBeUpdated проверяет, можно ли обновить заказ
func (o *Order) CanBeUpdated() bool {
	return o.Status == OrderStatusCreated || o.Status == OrderStatusInWork
}

// CanBeCancelled проверяет, можно ли отменить заказ
func (o *Order) CanBeCancelled() bool {
	return o.Status == OrderStatusCreated || o.Status == OrderStatusInWork
}

// ValidateStatus проверяет корректность статуса
func (s OrderStatus) IsValid() bool {
	return s == OrderStatusCreated || s == OrderStatusInWork ||
		s == OrderStatusCompleted || s == OrderStatusCancelled
}
