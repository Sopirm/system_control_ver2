package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"service_orders/config"
	"service_orders/events"
	"service_orders/logger"
	"service_orders/models"
	"service_orders/repository"
	"service_orders/utils"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// OrderHandler обработчик для заказов
type OrderHandler struct {
	orderRepo    repository.OrderRepository
	config       *config.Config
	eventService *events.EventService
}

// NewOrderHandler создает новый обработчик заказов
func NewOrderHandler(orderRepo repository.OrderRepository, config *config.Config, eventService *events.EventService) *OrderHandler {
	return &OrderHandler{
		orderRepo:    orderRepo,
		config:       config,
		eventService: eventService,
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
		logger.LogOrderAction(r, "create_order", order.ID.String(), err.Error(), false)
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка создания заказа")
		return
	}

	// Логируем успешное создание заказа
	details := fmt.Sprintf("items_count=%d, total_sum=%.2f", len(order.Items), order.TotalSum)
	logger.LogOrderAction(r, "create_order", order.ID.String(), details, true)
	logger.LogBusinessEvent(r, "order_created", order.ID.String(), "order", details)

	// Публикуем событие создания заказа
	ctx := context.Background()
	if err := h.eventService.PublishOrderCreated(ctx, order, r); err != nil {
		// Логируем ошибку, но не прерываем обработку - заказ уже создан
		logger.LogOrderAction(r, "publish_event", order.ID.String(), "OrderCreatedEvent failed: "+err.Error(), false)
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
		logger.LogOrderAction(r, "get_order", orderID.String(), "Order not found", false)
		h.sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Заказ не найден")
		return
	}

	// Проверка прав доступа
	if err := userCtx.ValidateOrderOwnership(order.UserID); err != nil {
		logger.LogOrderAction(r, "get_order", orderID.String(), "Access denied: "+err.Error(), false)
		h.sendErrorResponse(w, http.StatusForbidden, models.ErrorCodeForbidden, err.Error())
		return
	}

	logger.LogOrderAction(r, "get_order", orderID.String(), fmt.Sprintf("status=%s", order.Status), true)
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
		logger.LogOrderAction(r, "list_orders", userCtx.UserID.String(), err.Error(), false)
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка получения списка заказов")
		return
	}

	// Логируем успешное получение списка заказов
	listDetails := fmt.Sprintf("found=%d, limit=%d, offset=%d", len(response.Orders), req.Limit, req.Offset)
	logger.LogOrderAction(r, "list_orders", userCtx.UserID.String(), listDetails, true)

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

	// Сохраняем старый статус для события
	oldStatus := order.Status

	// Обновление статуса
	if err := h.orderRepo.UpdateStatus(orderID, req.Status); err != nil {
		logger.LogOrderAction(r, "update_status", orderID.String(), err.Error(), false)
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка обновления статуса заказа")
		return
	}

	// Логируем успешное обновление статуса
	statusDetails := fmt.Sprintf("%s -> %s", oldStatus, req.Status)
	logger.LogOrderAction(r, "update_status", orderID.String(), statusDetails, true)
	logger.LogBusinessEvent(r, "order_status_updated", orderID.String(), "order", statusDetails)

	// Публикуем событие обновления статуса
	ctx := context.Background()
	if err := h.eventService.PublishOrderStatusUpdated(ctx, orderID, order.UserID, userCtx.UserID, oldStatus, req.Status, r); err != nil {
		// Логируем ошибку, но не прерываем обработку - статус уже обновлен
		logger.LogOrderAction(r, "publish_event", orderID.String(), "OrderStatusUpdatedEvent failed: "+err.Error(), false)
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

	// Сохраняем старый статус для события
	oldStatus := order.Status

	// Отмена заказа
	if err := h.orderRepo.Cancel(orderID); err != nil {
		logger.LogOrderAction(r, "cancel_order", orderID.String(), err.Error(), false)
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка отмены заказа")
		return
	}

	// Логируем успешную отмену заказа
	cancelDetails := fmt.Sprintf("cancelled from status: %s", oldStatus)
	logger.LogOrderAction(r, "cancel_order", orderID.String(), cancelDetails, true)
	logger.LogBusinessEvent(r, "order_cancelled", orderID.String(), "order", cancelDetails)

	// Публикуем событие отмены заказа
	ctx := context.Background()
	if err := h.eventService.PublishOrderCancelled(ctx, orderID, order.UserID, userCtx.UserID, oldStatus, r); err != nil {
		// Логируем ошибку, но не прерываем обработку - заказ уже отменен
		logger.LogOrderAction(r, "publish_event", orderID.String(), "OrderCancelledEvent failed: "+err.Error(), false)
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
	// Логируем ошибки с уровнем ERROR если код >= 500, иначе WARN
	zapLogger := logger.GetLogger()
	if statusCode >= 500 {
		zapLogger.Error("HTTP Error Response",
			zap.Int("status_code", statusCode),
			zap.String("error_code", code),
			zap.String("error_message", message),
			zap.String("service", "service_orders"),
		)
	} else {
		zapLogger.Warn("HTTP Error Response",
			zap.Int("status_code", statusCode),
			zap.String("error_code", code),
			zap.String("error_message", message),
			zap.String("service", "service_orders"),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := models.NewErrorResponse(code, message)
	json.NewEncoder(w).Encode(response)
}
