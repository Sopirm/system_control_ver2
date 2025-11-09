package utils

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator глобальный экземпляр валидатора
var Validator *validator.Validate

func init() {
	Validator = validator.New()
}

// ValidateStruct валидирует структуру и возвращает читаемые ошибки
func ValidateStruct(s interface{}) error {
	err := Validator.Struct(s)
	if err != nil {
		var errors []string
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, getErrorMessage(err))
		}
		return fmt.Errorf(strings.Join(errors, "; "))
	}
	return nil
}

// getErrorMessage возвращает читаемое сообщение об ошибке валидации
func getErrorMessage(fe validator.FieldError) string {
	field := strings.ToLower(fe.Field())
	
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("поле '%s' обязательно для заполнения", field)
	case "email":
		return fmt.Sprintf("поле '%s' должно содержать корректный email адрес", field)
	case "min":
		return fmt.Sprintf("поле '%s' должно содержать минимум %s символов", field, fe.Param())
	case "max":
		return fmt.Sprintf("поле '%s' должно содержать максимум %s символов", field, fe.Param())
	default:
		return fmt.Sprintf("поле '%s' содержит некорректное значение", field)
	}
}
