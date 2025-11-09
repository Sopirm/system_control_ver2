package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"service_orders/models"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// OrderRepository интерфейс для работы с заказами
type OrderRepository interface {
	Create(order *models.Order) error
	GetByID(id uuid.UUID) (*models.Order, error)
	GetByUserID(userID uuid.UUID, req *models.ListOrdersRequest) (*models.ListOrdersResponse, error)
	Update(order *models.Order) error
	UpdateStatus(id uuid.UUID, status models.OrderStatus) error
	Cancel(id uuid.UUID) error
	UserExists(userID uuid.UUID) (bool, error)
}

// orderRepository реализация OrderRepository
type orderRepository struct {
	db *sql.DB
}

// NewOrderRepository создает новый экземпляр OrderRepository
func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{db: db}
}

// Create создает новый заказ
func (r *orderRepository) Create(order *models.Order) error {
	// Сериализуем items в JSONB
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return fmt.Errorf("ошибка сериализации items: %v", err)
	}

	query := `
		INSERT INTO orders (id, user_id, items, status, total_sum, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err = r.db.Exec(query,
		order.ID,
		order.UserID,
		itemsJSON,
		string(order.Status),
		order.TotalSum,
		order.CreatedAt,
		order.UpdatedAt,
	)
	
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return fmt.Errorf("пользователь с ID %s не существует", order.UserID)
		}
		return fmt.Errorf("ошибка создания заказа: %v", err)
	}
	
	return nil
}

// GetByID получает заказ по ID
func (r *orderRepository) GetByID(id uuid.UUID) (*models.Order, error) {
	query := `
		SELECT id, user_id, items, status, total_sum, created_at, updated_at
		FROM orders
		WHERE id = $1
	`
	
	order := &models.Order{}
	var itemsJSON []byte
	var status string
	
	err := r.db.QueryRow(query, id).Scan(
		&order.ID,
		&order.UserID,
		&itemsJSON,
		&status,
		&order.TotalSum,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("заказ с ID %s не найден", id)
		}
		return nil, fmt.Errorf("ошибка получения заказа: %v", err)
	}
	
	// Десериализуем items из JSONB
	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, fmt.Errorf("ошибка десериализации items: %v", err)
	}
	
	order.Status = models.OrderStatus(status)
	
	return order, nil
}

// GetByUserID получает заказы пользователя с фильтрацией и пагинацией
func (r *orderRepository) GetByUserID(userID uuid.UUID, req *models.ListOrdersRequest) (*models.ListOrdersResponse, error) {
	// Построение WHERE условий
	conditions := []string{"user_id = $1"}
	args := []interface{}{userID}
	argIndex := 2

	if req.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, string(req.Status))
		argIndex++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Построение ORDER BY
	sortField := "created_at"
	if req.Sort != "" {
		sortField = req.Sort
	}
	
	sortOrder := "DESC"
	if req.Order == "asc" {
		sortOrder = "ASC"
	}
	
	orderClause := fmt.Sprintf("ORDER BY %s %s", sortField, sortOrder)

	// Получение общего количества
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders %s", whereClause)
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("ошибка подсчета заказов: %v", err)
	}

	// Получение списка заказов
	query := fmt.Sprintf(`
		SELECT id, user_id, items, status, total_sum, created_at, updated_at
		FROM orders
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, argIndex, argIndex+1)

	args = append(args, req.Limit, req.Offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка заказов: %v", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		var itemsJSON []byte
		var status string
		
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&itemsJSON,
			&status,
			&order.TotalSum,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования заказа: %v", err)
		}
		
		// Десериализуем items из JSONB
		if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
			return nil, fmt.Errorf("ошибка десериализации items: %v", err)
		}
		
		order.Status = models.OrderStatus(status)
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по строкам: %v", err)
	}

	return &models.ListOrdersResponse{
		Orders: orders,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, nil
}

// Update обновляет данные заказа
func (r *orderRepository) Update(order *models.Order) error {
	// Сериализуем items в JSONB
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return fmt.Errorf("ошибка сериализации items: %v", err)
	}

	query := `
		UPDATE orders
		SET items = $2, status = $3, total_sum = $4, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.Exec(query,
		order.ID,
		itemsJSON,
		string(order.Status),
		order.TotalSum,
	)
	
	if err != nil {
		return fmt.Errorf("ошибка обновления заказа: %v", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %v", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("заказ с ID %s не найден", order.ID)
	}
	
	return nil
}

// UpdateStatus обновляет статус заказа
func (r *orderRepository) UpdateStatus(id uuid.UUID, status models.OrderStatus) error {
	query := `
		UPDATE orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`
	
	result, err := r.db.Exec(query, id, string(status))
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса заказа: %v", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества обновленных строк: %v", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("заказ с ID %s не найден", id)
	}
	
	return nil
}

// Cancel отменяет заказ
func (r *orderRepository) Cancel(id uuid.UUID) error {
	return r.UpdateStatus(id, models.OrderStatusCancelled)
}

// UserExists проверяет существование пользователя
func (r *orderRepository) UserExists(userID uuid.UUID) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)"
	
	var exists bool
	err := r.db.QueryRow(query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки существования пользователя: %v", err)
	}
	
	return exists, nil
}
