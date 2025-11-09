package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"

	"service_orders/models"
)

// EventStats для отслеживания статистики событий
var eventStats struct {
	OrdersCreated       int64
	StatusUpdates       int64
	OrdersCancelled     int64
	EventsPublished     int64
	EventProcessingErrors int64
}

// AnalyticsEventHandler обработчик событий для аналитики
func AnalyticsEventHandler(ctx context.Context, event *DomainEvent) error {
	atomic.AddInt64(&eventStats.EventsPublished, 1)
	
	switch event.Type {
	case OrderCreatedEvent:
		atomic.AddInt64(&eventStats.OrdersCreated, 1)
		return handleOrderCreatedAnalytics(event)
	case OrderStatusUpdatedEvent:
		atomic.AddInt64(&eventStats.StatusUpdates, 1)
		return handleOrderStatusAnalytics(event)
	default:
		log.Printf("Неизвестный тип события для аналитики: %s", event.Type)
	}
	return nil
}

// NotificationEventHandler обработчик событий для уведомлений
func NotificationEventHandler(ctx context.Context, event *DomainEvent) error {
	switch event.Type {
	case OrderCreatedEvent:
		return handleOrderCreatedNotification(event)
	case OrderStatusUpdatedEvent:
		return handleOrderStatusNotification(event)
	default:
		log.Printf("Неизвестный тип события для уведомлений: %s", event.Type)
	}
	return nil
}

// AuditEventHandler обработчик событий для аудита (логирование в БД/файл)
func AuditEventHandler(ctx context.Context, event *DomainEvent) error {
	auditLog := map[string]interface{}{
		"event_id":     event.ID,
		"event_type":   event.Type,
		"aggregate_id": event.AggregateID,
		"user_id":      event.UserID,
		"timestamp":    event.Timestamp,
		"metadata":     event.Metadata,
		"data":         event.Data,
	}
	
	auditJSON, err := json.MarshalIndent(auditLog, "", "  ")
	if err != nil {
		atomic.AddInt64(&eventStats.EventProcessingErrors, 1)
		return fmt.Errorf("ошибка сериализации события для аудита: %v", err)
	}
	
	// В реальном приложении здесь была бы запись в базу аудита или в файл
	log.Printf("AUDIT EVENT: %s", auditJSON)
	
	return nil
}

// handleOrderCreatedAnalytics обрабатывает аналитику создания заказа
func handleOrderCreatedAnalytics(event *DomainEvent) error {
	data, ok := event.Data.(OrderCreatedEventData)
	if !ok {
		// Попробуем десериализовать из map[string]interface{} (может быть после JSON unmarshaling)
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			dataJSON, _ := json.Marshal(dataMap)
			if err := json.Unmarshal(dataJSON, &data); err != nil {
				atomic.AddInt64(&eventStats.EventProcessingErrors, 1)
				return fmt.Errorf("невозможно десериализовать данные OrderCreatedEvent: %v", err)
			}
		} else {
			atomic.AddInt64(&eventStats.EventProcessingErrors, 1)
			return fmt.Errorf("неверный тип данных для OrderCreatedEvent")
		}
	}
	
	// Здесь можно добавить логику для:
	// - Обновления метрик продаж
	// - Анализа популярных товаров
	// - Расчета конверсии
	log.Printf("📊 АНАЛИТИКА: Новый заказ на сумму %.2f руб. (%d товаров)", 
		data.TotalSum, len(data.Items))
	
	return nil
}

// handleOrderStatusAnalytics обрабатывает аналитику изменения статуса
func handleOrderStatusAnalytics(event *DomainEvent) error {
	data, ok := event.Data.(OrderStatusUpdatedEventData)
	if !ok {
		// Попробуем десериализовать из map[string]interface{}
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			dataJSON, _ := json.Marshal(dataMap)
			if err := json.Unmarshal(dataJSON, &data); err != nil {
				atomic.AddInt64(&eventStats.EventProcessingErrors, 1)
				return fmt.Errorf("невозможно десериализовать данные OrderStatusUpdatedEvent: %v", err)
			}
		} else {
			atomic.AddInt64(&eventStats.EventProcessingErrors, 1)
			return fmt.Errorf("неверный тип данных для OrderStatusUpdatedEvent")
		}
	}
	
	// Подсчитываем отмены
	if data.NewStatus == models.OrderStatusCancelled {
		atomic.AddInt64(&eventStats.OrdersCancelled, 1)
	}
	
	// Здесь можно добавить логику для:
	// - Отслеживания времени выполнения заказов
	// - Анализа причин отмен
	// - Метрик производительности
	log.Printf("📊 АНАЛИТИКА: Заказ %s изменил статус: %s → %s", 
		data.OrderID, data.OldStatus, data.NewStatus)
	
	return nil
}

// handleOrderCreatedNotification отправляет уведомление о создании заказа
func handleOrderCreatedNotification(event *DomainEvent) error {
	data, ok := event.Data.(OrderCreatedEventData)
	if !ok {
		// Попробуем десериализовать из map[string]interface{}
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			dataJSON, _ := json.Marshal(dataMap)
			if err := json.Unmarshal(dataJSON, &data); err != nil {
				atomic.AddInt64(&eventStats.EventProcessingErrors, 1)
				return fmt.Errorf("невозможно десериализовать данные для уведомления: %v", err)
			}
		} else {
			atomic.AddInt64(&eventStats.EventProcessingErrors, 1)
			return fmt.Errorf("неверный тип данных для уведомления о создании заказа")
		}
	}
	
	// Здесь можно добавить логику для:
	// - Отправки email уведомлений
	// - Push-уведомлений в мобильном приложении
	// - SMS уведомлений
	log.Printf("📧 УВЕДОМЛЕНИЕ: Пользователю %s отправлено уведомление о создании заказа %s", 
		data.UserID, data.OrderID)
	
	return nil
}

// handleOrderStatusNotification отправляет уведомление об изменении статуса
func handleOrderStatusNotification(event *DomainEvent) error {
	data, ok := event.Data.(OrderStatusUpdatedEventData)
	if !ok {
		// Попробуем десериализовать из map[string]interface{}
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			dataJSON, _ := json.Marshal(dataMap)
			if err := json.Unmarshal(dataJSON, &data); err != nil {
				atomic.AddInt64(&eventStats.EventProcessingErrors, 1)
				return fmt.Errorf("невозможно десериализовать данные для уведомления: %v", err)
			}
		} else {
			atomic.AddInt64(&eventStats.EventProcessingErrors, 1)
			return fmt.Errorf("неверный тип данных для уведомления об изменении статуса")
		}
	}
	
	// Отправляем уведомления только для определенных статусов
	if data.NewStatus == models.OrderStatusCompleted || data.NewStatus == models.OrderStatusCancelled {
		log.Printf("📧 УВЕДОМЛЕНИЕ: Пользователю %s отправлено уведомление об изменении статуса заказа %s на '%s'", 
			data.UserID, data.OrderID, data.NewStatus)
	}
	
	return nil
}

// GetEventStats возвращает статистику событий
func GetEventStats() map[string]int64 {
	return map[string]int64{
		"orders_created":         atomic.LoadInt64(&eventStats.OrdersCreated),
		"status_updates":         atomic.LoadInt64(&eventStats.StatusUpdates),
		"orders_cancelled":       atomic.LoadInt64(&eventStats.OrdersCancelled),
		"events_published":       atomic.LoadInt64(&eventStats.EventsPublished),
		"event_processing_errors": atomic.LoadInt64(&eventStats.EventProcessingErrors),
	}
}
