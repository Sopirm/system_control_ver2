package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// User представляет модель пользователя
type User struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	Email     string         `json:"email" db:"email"`
	Password  string         `json:"-" db:"password_hash"` // хэш пароля, не возвращается в JSON
	Name      string         `json:"name" db:"name"`
	Roles     pq.StringArray `json:"roles" db:"roles"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// RegisterRequest представляет запрос на регистрацию пользователя
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required,min=2"`
}

// LoginRequest представляет запрос на вход пользователя
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse представляет ответ при успешном входе
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// UpdateProfileRequest представляет запрос на обновление профиля
type UpdateProfileRequest struct {
	Name  string `json:"name" validate:"required,min=2"`
	Email string `json:"email" validate:"required,email"`
}

// ListUsersRequest представляет параметры для получения списка пользователей
type ListUsersRequest struct {
	Limit  int    `json:"limit" validate:"min=1,max=100"`
	Offset int    `json:"offset" validate:"min=0"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

// ListUsersResponse представляет ответ со списком пользователей
type ListUsersResponse struct {
	Users  []User `json:"users"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// HasRole проверяет, есть ли у пользователя указанная роль
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin проверяет, является ли пользователь администратором
func (u *User) IsAdmin() bool {
	return u.HasRole("admin")
}
