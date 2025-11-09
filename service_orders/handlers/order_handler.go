package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"service_orders/config"
	"service_orders/models"
	"service_orders/repository"
	"service_orders/utils"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// OrderHandler обработчик для заказов
type OrderHandler struct {
	orderRepo repository.OrderRepository
	config    *config.Config
}

// NewOrderHandler создает новый обработчик заказов
func NewOrderHandler(orderRepo repository.OrderRepository, config *config.Config) *OrderHandler {
	return &OrderHandler{
		orderRepo: orderRepo,
		config:    config,
	}
}

// CreateOrder обрабатывает создание нового заказа
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// Получение пользовательского контекста
	userCtx, err := utils.GetUserContextFromHeaders(r)
	if err != nil {
		h.sendErrorResponse(w, http.StatusUnauthorized, models.ErrorCodeUnauthorized, err.Error())
		return
	}

	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, "Некорректный JSON")
		return
	}

	// Валидация входных данных
	if err := utils.ValidateStruct(req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, err.Error())
		return
	}

	// Проверка существования пользователя
	exists, err := h.orderRepo.UserExists(userCtx.UserID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка проверки пользователя")
		return
	}
	if !exists {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, "Пользователь не существует")
		return
	}

	// Создание заказа
	order := &models.Order{
		ID:        uuid.New(),
		UserID:    userCtx.UserID,
		Items:     req.Items,
		Status:    models.OrderStatusCreated,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Вычисление общей стоимости
	order.CalculateTotal()

	if err := h.orderRepo.Create(order); err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка создания заказа")
		return
	}

	h.sendSuccessResponse(w, http.StatusCreated, order)
}

// GetOrder возвращает заказ по идентификатору
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	// Получение пользовательского контекста
	userCtx, err := utils.GetUserContextFromHeaders(r)
	if err != nil {
		h.sendErrorResponse(w, http.StatusUnauthorized, models.ErrorCodeUnauthorized, err.Error())
		return
	}

	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, "Некорректный ID заказа")
		return
	}

	order, err := h.orderRepo.GetByID(orderID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Заказ не найден")
		return
	}

	// Проверка прав доступа
	if err := userCtx.ValidateOrderOwnership(order.UserID); err != nil {
		h.sendErrorResponse(w, http.StatusForbidden, models.ErrorCodeForbidden, err.Error())
		return
	}

	h.sendSuccessResponse(w, http.StatusOK, order)
}

// ListOrders возвращает список заказов текущего пользователя
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	// Получение пользовательского контекста
	userCtx, err := utils.GetUserContextFromHeaders(r)
	if err != nil {
		h.sendErrorResponse(w, http.StatusUnauthorized, models.ErrorCodeUnauthorized, err.Error())
		return
	}

	// Парсинг параметров запроса
	req := &models.ListOrdersRequest{
		Limit:  10, // значение по умолчанию
		Offset: 0,
		Sort:   "created_at",
		Order:  "desc",
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			req.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			req.Offset = offset
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		req.Status = models.OrderStatus(status)
	}

	if sort := r.URL.Query().Get("sort"); sort != "" {
		req.Sort = sort
	}

	if order := r.URL.Query().Get("order"); order != "" {
		req.Order = order
	}

	// Валидация параметров
	if err := utils.ValidateStruct(req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, err.Error())
		return
	}

	// Получение списка заказов
	response, err := h.orderRepo.GetByUserID(userCtx.UserID, req)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка получения списка заказов")
		return
	}

	h.sendSuccessResponse(w, http.StatusOK, response)
}

// UpdateOrderStatus обновляет статус заказа
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	// Получение пользовательского контекста
	userCtx, err := utils.GetUserContextFromHeaders(r)
	if err != nil {
		h.sendErrorResponse(w, http.StatusUnauthorized, models.ErrorCodeUnauthorized, err.Error())
		return
	}

	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, "Некорректный ID заказа")
		return
	}

	var req models.UpdateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, "Некорректный JSON")
		return
	}

	// Валидация входных данных
	if err := utils.ValidateStruct(req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, err.Error())
		return
	}

	// Получение текущего заказа
	order, err := h.orderRepo.GetByID(orderID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Заказ не найден")
		return
	}

	// Проверка прав доступа
	if err := userCtx.ValidateOrderOwnership(order.UserID); err != nil {
		h.sendErrorResponse(w, http.StatusForbidden, models.ErrorCodeForbidden, err.Error())
		return
	}

	// Проверка возможности обновления
	if !order.CanBeUpdated() {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, 
			fmt.Sprintf("Нельзя обновить заказ со статусом '%s'", order.Status))
		return
	}

	// Обновление статуса
	if err := h.orderRepo.UpdateStatus(orderID, req.Status); err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка обновления статуса заказа")
		return
	}

	// Получение обновленного заказа
	updatedOrder, err := h.orderRepo.GetByID(orderID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка получения обновленного заказа")
		return
	}

	h.sendSuccessResponse(w, http.StatusOK, updatedOrder)
}

// CancelOrder отменяет заказ
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	// Получение пользовательского контекста
	userCtx, err := utils.GetUserContextFromHeaders(r)
	if err != nil {
		h.sendErrorResponse(w, http.StatusUnauthorized, models.ErrorCodeUnauthorized, err.Error())
		return
	}

	vars := mux.Vars(r)
	orderID, err := uuid.Parse(vars["id"])
	if err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, "Некорректный ID заказа")
		return
	}

	// Получение текущего заказа
	order, err := h.orderRepo.GetByID(orderID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Заказ не найден")
		return
	}

	// Проверка прав доступа
	if err := userCtx.ValidateOrderOwnership(order.UserID); err != nil {
		h.sendErrorResponse(w, http.StatusForbidden, models.ErrorCodeForbidden, err.Error())
		return
	}

	// Проверка возможности отмены
	if !order.CanBeCancelled() {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, 
			fmt.Sprintf("Нельзя отменить заказ со статусом '%s'", order.Status))
		return
	}

	// Отмена заказа
	if err := h.orderRepo.Cancel(orderID); err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка отмены заказа")
		return
	}

	// Получение обновленного заказа
	cancelledOrder, err := h.orderRepo.GetByID(orderID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка получения отмененного заказа")
		return
	}

	h.sendSuccessResponse(w, http.StatusOK, cancelledOrder)
}

// sendSuccessResponse отправляет успешный ответ
func (h *OrderHandler) sendSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := models.NewSuccessResponse(data)
	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse отправляет ответ с ошибкой
func (h *OrderHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := models.NewErrorResponse(code, message)
	json.NewEncoder(w).Encode(response)
}
