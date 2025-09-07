package models

import (
	"fmt"
	"net/http"
)

// APIError はAPIエラーを表します
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Type    string `json:"type"`
}

// Error はerrorインターフェースを実装します
func (e *APIError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// HTTPStatus はHTTPステータスコードを返します
func (e *APIError) HTTPStatus() int {
	return e.Code
}

// NewValidationError はバリデーションエラーを作成します
func NewValidationError(field, message string) *APIError {
	return &APIError{
		Code:    http.StatusBadRequest,
		Message: message,
		Field:   field,
		Type:    "validation_error",
	}
}

// NewNotFoundError は404エラーを作成します
func NewNotFoundError(resource string) *APIError {
	return &APIError{
		Code:    http.StatusNotFound,
		Message: fmt.Sprintf("%sが見つかりません", resource),
		Type:    "not_found_error",
	}
}

// NewInternalError は500エラーを作成します
func NewInternalError(message string) *APIError {
	return &APIError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Type:    "internal_error",
	}
}

// NewConflictError は409エラーを作成します
func NewConflictError(message string) *APIError {
	return &APIError{
		Code:    http.StatusConflict,
		Message: message,
		Type:    "conflict_error",
	}
}

// ErrorResponse は統一されたエラーレスポンス形式です
type ErrorResponse struct {
	Error *APIError `json:"error"`
}
