package models

// APIResponse представляет стандартный формат ответа API
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError представляет структуру ошибки
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewSuccessResponse создает успешный ответ
func NewSuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse создает ответ с ошибкой
func NewErrorResponse(code, message string) APIResponse {
	return APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
}

// Константы для кодов ошибок
const (
	ErrorCodeValidation     = "VALIDATION_ERROR"
	ErrorCodeNotFound       = "NOT_FOUND"
	ErrorCodeUnauthorized   = "UNAUTHORIZED"
	ErrorCodeForbidden      = "FORBIDDEN"
	ErrorCodeConflict       = "CONFLICT"
	ErrorCodeInternalServer = "INTERNAL_SERVER_ERROR"
)
