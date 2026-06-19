package errors

import "net/http"

type CodeError struct {
	Code       int
	Message    string
	StatusCode int
}

func (e *CodeError) Error() string {
	return e.Message
}

func New(code int, message string, statusCode int) *CodeError {
	return &CodeError{Code: code, Message: message, StatusCode: statusCode}
}

func BadRequest(message string) *CodeError {
	return New(40000, message, http.StatusBadRequest)
}

func Unauthorized(message string) *CodeError {
	return New(40100, message, http.StatusUnauthorized)
}

func Forbidden(message string) *CodeError {
	return New(40300, message, http.StatusForbidden)
}

func NotFound(message string) *CodeError {
	return New(40400, message, http.StatusNotFound)
}

func Conflict(message string) *CodeError {
	return New(40900, message, http.StatusConflict)
}

func TooManyRequests(message string) *CodeError {
	return New(42900, message, http.StatusTooManyRequests)
}

func ServiceUnavailable(message string) *CodeError {
	return New(50300, message, http.StatusServiceUnavailable)
}

func Internal(message string) *CodeError {
	return New(50000, message, http.StatusInternalServerError)
}
