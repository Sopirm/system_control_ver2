package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"service_users/config"
	"service_users/models"
	"service_users/repository"
	"service_users/utils"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// UserHandler обработчик для пользователей
type UserHandler struct {
	userRepo repository.UserRepository
	config   *config.Config
}

// NewUserHandler создает новый обработчик пользователей
func NewUserHandler(userRepo repository.UserRepository, config *config.Config) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		config:   config,
	}
}

// RegisterUser обрабатывает регистрацию нового пользователя
func (h *UserHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, "Некорректный JSON")
		return
	}

	// Валидация входных данных
	if err := utils.ValidateStruct(req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, err.Error())
		return
	}

	// Проверка существования email
	exists, err := h.userRepo.EmailExists(req.Email)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка проверки email")
		return
	}
	if exists {
		h.sendErrorResponse(w, http.StatusConflict, models.ErrorCodeConflict, "Пользователь с таким email уже существует")
		return
	}

	// Хеширование пароля
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка обработки пароля")
		return
	}

	// Создание пользователя
	user := &models.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Password:  hashedPassword,
		Name:      req.Name,
		Roles:     pq.StringArray{"user"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.userRepo.Create(user); err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка создания пользователя")
		return
	}

	// Очищаем пароль перед отправкой
	user.Password = ""
	h.sendSuccessResponse(w, http.StatusCreated, user)
}

// LoginUser обрабатывает вход пользователя
func (h *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, "Некорректный JSON")
		return
	}

	// Валидация входных данных
	if err := utils.ValidateStruct(req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, err.Error())
		return
	}

	// Поиск пользователя по email
	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		h.sendErrorResponse(w, http.StatusUnauthorized, models.ErrorCodeUnauthorized, "Неверный email или пароль")
		return
	}

	// Проверка пароля
	if !utils.CheckPassword(req.Password, user.Password) {
		h.sendErrorResponse(w, http.StatusUnauthorized, models.ErrorCodeUnauthorized, "Неверный email или пароль")
		return
	}

	// Генерация JWT токена
	token, err := utils.GenerateJWT(user, h.config.JWT.Secret)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка генерации токена")
		return
	}

	// Очищаем пароль перед отправкой
	user.Password = ""
	
	response := models.LoginResponse{
		Token: token,
		User:  *user,
	}

	h.sendSuccessResponse(w, http.StatusOK, response)
}

// GetUserProfile возвращает профиль текущего пользователя
func (h *UserHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserIDFromContext(r)
	if err != nil {
		h.sendErrorResponse(w, http.StatusUnauthorized, models.ErrorCodeUnauthorized, "Не удалось получить ID пользователя")
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Пользователь не найден")
		return
	}

	// Очищаем пароль перед отправкой
	user.Password = ""
	h.sendSuccessResponse(w, http.StatusOK, user)
}

// UpdateUserProfile обновляет профиль пользователя
func (h *UserHandler) UpdateUserProfile(w http.ResponseWriter, r *http.Request) {
	userID, err := h.getUserIDFromContext(r)
	if err != nil {
		h.sendErrorResponse(w, http.StatusUnauthorized, models.ErrorCodeUnauthorized, "Не удалось получить ID пользователя")
		return
	}

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, "Некорректный JSON")
		return
	}

	// Валидация входных данных
	if err := utils.ValidateStruct(req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, err.Error())
		return
	}

	// Получение текущего пользователя
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		h.sendErrorResponse(w, http.StatusNotFound, models.ErrorCodeNotFound, "Пользователь не найден")
		return
	}

	// Проверка уникальности email (если изменился)
	if user.Email != req.Email {
		exists, err := h.userRepo.EmailExists(req.Email)
		if err != nil {
			h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка проверки email")
			return
		}
		if exists {
			h.sendErrorResponse(w, http.StatusConflict, models.ErrorCodeConflict, "Пользователь с таким email уже существует")
			return
		}
	}

	// Обновление данных
	user.Email = req.Email
	user.Name = req.Name

	if err := h.userRepo.Update(user); err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка обновления профиля")
		return
	}

	// Очищаем пароль перед отправкой
	user.Password = ""
	h.sendSuccessResponse(w, http.StatusOK, user)
}

// ListUsers возвращает список пользователей (только для администраторов)
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Проверка роли администратора
	if !h.isAdmin(r) {
		h.sendErrorResponse(w, http.StatusForbidden, models.ErrorCodeForbidden, "Недостаточно прав доступа")
		return
	}

	// Парсинг параметров запроса
	req := &models.ListUsersRequest{
		Limit:  10, // значение по умолчанию
		Offset: 0,
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

	req.Email = r.URL.Query().Get("email")
	req.Name = r.URL.Query().Get("name")
	req.Role = r.URL.Query().Get("role")

	// Валидация параметров
	if err := utils.ValidateStruct(req); err != nil {
		h.sendErrorResponse(w, http.StatusBadRequest, models.ErrorCodeValidation, err.Error())
		return
	}

	// Получение списка пользователей
	response, err := h.userRepo.List(req)
	if err != nil {
		h.sendErrorResponse(w, http.StatusInternalServerError, models.ErrorCodeInternalServer, "Ошибка получения списка пользователей")
		return
	}

	h.sendSuccessResponse(w, http.StatusOK, response)
}

// getUserIDFromContext извлекает ID пользователя из заголовка (переданного от API Gateway)
func (h *UserHandler) getUserIDFromContext(r *http.Request) (uuid.UUID, error) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return uuid.Nil, fmt.Errorf("отсутствует заголовок X-User-ID")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("некорректный формат X-User-ID: %v", err)
	}

	return userID, nil
}

// isAdmin проверяет, является ли пользователь администратором
func (h *UserHandler) isAdmin(r *http.Request) bool {
	rolesStr := r.Header.Get("X-User-Roles")
	if rolesStr == "" {
		return false
	}

	roles := strings.Split(rolesStr, ",")
	for _, role := range roles {
		if strings.TrimSpace(role) == "admin" {
			return true
		}
	}

	return false
}

// sendSuccessResponse отправляет успешный ответ
func (h *UserHandler) sendSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := models.NewSuccessResponse(data)
	json.NewEncoder(w).Encode(response)
}

// sendErrorResponse отправляет ответ с ошибкой
func (h *UserHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	response := models.NewErrorResponse(code, message)
	json.NewEncoder(w).Encode(response)
}
