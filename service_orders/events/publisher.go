package events

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// EventPublisher интерфейс для публикации доменных событий
type EventPublisher interface {
	Publish(ctx context.Context, event *DomainEvent) error
	Subscribe(eventType EventType, handler EventHandler) error
	Close() error
}

// EventHandler функция-обработчик события
type EventHandler func(ctx context.Context, event *DomainEvent) error

// InMemoryEventPublisher простая реализация для разработки и тестирования
// В будущем будет заменена на Kafka/RabbitMQ
type InMemoryEventPublisher struct {
	subscribers map[EventType][]EventHandler
	mutex       sync.RWMutex
	events      chan *DomainEvent
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewInMemoryEventPublisher создает новый in-memory publisher
func NewInMemoryEventPublisher() *InMemoryEventPublisher {
	ctx, cancel := context.WithCancel(context.Background())
	
	publisher := &InMemoryEventPublisher{
		subscribers: make(map[EventType][]EventHandler),
		events:      make(chan *DomainEvent, 100), // Буфер для 100 событий
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// Запускаем горутину для обработки событий
	publisher.wg.Add(1)
	go publisher.processEvents()
	
	return publisher
}

// Publish публикует событие
func (p *InMemoryEventPublisher) Publish(ctx context.Context, event *DomainEvent) error {
	select {
	case p.events <- event:
		log.Printf("Событие опубликовано: %s (ID: %s, AggregateID: %s)", 
			event.Type, event.ID, event.AggregateID)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-p.ctx.Done():
		return fmt.Errorf("publisher закрыт")
	default:
		return fmt.Errorf("очередь событий переполнена")
	}
}

// Subscribe подписывается на события определенного типа
func (p *InMemoryEventPublisher) Subscribe(eventType EventType, handler EventHandler) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	p.subscribers[eventType] = append(p.subscribers[eventType], handler)
	log.Printf("Добавлен обработчик для событий типа: %s", eventType)
	
	return nil
}

// processEvents обрабатывает события в отдельной горутине
func (p *InMemoryEventPublisher) processEvents() {
	defer p.wg.Done()
	
	for {
		select {
		case event := <-p.events:
			p.handleEvent(event)
		case <-p.ctx.Done():
			// Обрабатываем оставшиеся события перед закрытием
			for {
				select {
				case event := <-p.events:
					p.handleEvent(event)
				default:
					return
				}
			}
		}
	}
}

// handleEvent обрабатывает одно событие
func (p *InMemoryEventPublisher) handleEvent(event *DomainEvent) {
	p.mutex.RLock()
	handlers := p.subscribers[event.Type]
	p.mutex.RUnlock()
	
	if len(handlers) == 0 {
		log.Printf("Нет обработчиков для события: %s", event.Type)
		return
	}
	
	// Обрабатываем событие всеми подписчиками
	for _, handler := range handlers {
		go func(h EventHandler) {
			ctx := context.Background()
			if err := h(ctx, event); err != nil {
				log.Printf("Ошибка обработки события %s: %v", event.Type, err)
			}
		}(handler)
	}
}

// Close закрывает publisher
func (p *InMemoryEventPublisher) Close() error {
	p.cancel()
	p.wg.Wait()
	close(p.events)
	
	log.Println("EventPublisher закрыт")
	return nil
}

// KafkaEventPublisher заготовка для Kafka (для будущих итераций)
type KafkaEventPublisher struct {
	brokers []string
	topic   string
	// producer kafka.Producer // Будет добавлен при интеграции с Kafka
}

// NewKafkaEventPublisher создает новый Kafka publisher (заготовка)
func NewKafkaEventPublisher(brokers []string, topic string) (*KafkaEventPublisher, error) {
	// TODO: Реализовать при интеграции с Kafka
	return &KafkaEventPublisher{
		brokers: brokers,
		topic:   topic,
	}, fmt.Errorf("Kafka publisher еще не реализован - используйте InMemoryEventPublisher")
}

// Publish публикует событие в Kafka (заготовка)
func (p *KafkaEventPublisher) Publish(ctx context.Context, event *DomainEvent) error {
	// TODO: Реализовать отправку в Kafka
	return fmt.Errorf("Kafka publisher еще не реализован")
}

// Subscribe подписывается на события из Kafka (заготовка)
func (p *KafkaEventPublisher) Subscribe(eventType EventType, handler EventHandler) error {
	// TODO: Реализовать подписку на Kafka топики
	return fmt.Errorf("Kafka publisher еще не реализован")
}

// Close закрывает Kafka подключение (заготовка)
func (p *KafkaEventPublisher) Close() error {
	// TODO: Закрыть Kafka producer/consumer
	return nil
}

// DefaultEventHandlers стандартные обработчики событий для логирования
var DefaultEventHandlers = map[EventType]EventHandler{
	OrderCreatedEvent: func(ctx context.Context, event *DomainEvent) error {
		data, ok := event.Data.(OrderCreatedEventData)
		if !ok {
			return fmt.Errorf("неверный тип данных для OrderCreatedEvent")
		}
		
		log.Printf("🎉 СОЗДАН НОВЫЙ ЗАКАЗ: ID=%s, Пользователь=%s, Сумма=%.2f, Товаров=%d",
			data.OrderID, data.UserID, data.TotalSum, len(data.Items))
		
		// Здесь можно добавить отправку уведомлений, обновление аналитики и т.д.
		return nil
	},
	
	OrderStatusUpdatedEvent: func(ctx context.Context, event *DomainEvent) error {
		data, ok := event.Data.(OrderStatusUpdatedEventData)
		if !ok {
			return fmt.Errorf("неверный тип данных для OrderStatusUpdatedEvent")
		}
		
		log.Printf("📊 ОБНОВЛЕН СТАТУС ЗАКАЗА: ID=%s, %s → %s, Пользователь=%s",
			data.OrderID, data.OldStatus, data.NewStatus, data.UserID)
		
		// Здесь можно добавить бизнес-логику в зависимости от статуса:
		// - Отправка email уведомлений
		// - Обновление инвентаря
		// - Запуск процесса доставки
		// - Обновление аналитики
		
		return nil
	},
}
