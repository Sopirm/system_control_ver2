package utils

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// UserContext представляет контекст пользователя из заголовков
type UserContext struct {
	UserID uuid.UUID
	Email  string
	Roles  []string
}

// GetUserContextFromHeaders извлекает пользовательский контекст из заголовков HTTP
func GetUserContextFromHeaders(r *http.Request) (*UserContext, error) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return nil, fmt.Errorf("отсутствует заголовок X-User-ID")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("некорректный формат X-User-ID: %v", err)
	}

	email := r.Header.Get("X-User-Email")
	if email == "" {
		return nil, fmt.Errorf("отсутствует заголовок X-User-Email")
	}

	rolesStr := r.Header.Get("X-User-Roles")
	var roles []string
	if rolesStr != "" {
		roles = strings.Split(rolesStr, ",")
		// Убираем лишние пробелы
		for i, role := range roles {
			roles[i] = strings.TrimSpace(role)
		}
	}

	return &UserContext{
		UserID: userID,
		Email:  email,
		Roles:  roles,
	}, nil
}

// HasRole проверяет, есть ли у пользователя указанная роль
func (uc *UserContext) HasRole(role string) bool {
	for _, r := range uc.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin проверяет, является ли пользователь администратором
func (uc *UserContext) IsAdmin() bool {
	return uc.HasRole("admin")
}

// ValidateOrderOwnership проверяет, может ли пользователь работать с заказом
func (uc *UserContext) ValidateOrderOwnership(orderUserID uuid.UUID) error {
	// Администратор может работать со всеми заказами
	if uc.IsAdmin() {
		return nil
	}
	
	// Обычный пользователь может работать только со своими заказами
	if uc.UserID != orderUserID {
		return fmt.Errorf("недостаточно прав для доступа к заказу")
	}
	
	return nil
}
