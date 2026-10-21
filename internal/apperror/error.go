package apperror

import (
	"errors"
	"net/http"
)

type Error struct {
	Code    string
	Message string
	Status  int
	Details map[string]any
	Cause   error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func New(status int, code, message string, details map[string]any) *Error {
	return &Error{Status: status, Code: code, Message: message, Details: details}
}

func Wrap(status int, code, message string, cause error) *Error {
	return &Error{Status: status, Code: code, Message: message, Cause: cause}
}

func As(err error) (*Error, bool) {
	var appErr *Error
	ok := errors.As(err, &appErr)
	return appErr, ok
}

func Validation(details map[string]any) *Error {
	return New(http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", details)
}

func AccessDenied() *Error {
	return New(http.StatusForbidden, "ACCESS_DENIED", "You do not have permission to perform this operation", nil)
}

func TokenInvalid() *Error {
	return New(http.StatusUnauthorized, "TOKEN_INVALID", "Access token is invalid", nil)
}

func TokenExpired() *Error {
	return New(http.StatusUnauthorized, "TOKEN_EXPIRED", "Access token has expired", nil)
}

func RefreshTokenInvalid() *Error {
	return New(http.StatusUnauthorized, "REFRESH_TOKEN_INVALID", "Refresh token is invalid or expired", nil)
}

func ProductNotFound(id int64) *Error {
	return New(http.StatusNotFound, "PRODUCT_NOT_FOUND", "Product was not found", map[string]any{"product_id": id})
}

func ProductInactive(id int64) *Error {
	return New(http.StatusConflict, "PRODUCT_INACTIVE", "Product is not active", map[string]any{"product_id": id})
}

func OrderNotFound(id int64) *Error {
	return New(http.StatusNotFound, "ORDER_NOT_FOUND", "Order was not found", map[string]any{"order_id": id})
}

func OrderOwnershipViolation() *Error {
	return New(http.StatusForbidden, "ORDER_OWNERSHIP_VIOLATION", "Order belongs to another user", nil)
}

func OrderLimitExceeded(retryAfterSeconds int64) *Error {
	return New(http.StatusTooManyRequests, "ORDER_LIMIT_EXCEEDED", "Order operation is being performed too often", map[string]any{"retry_after_seconds": retryAfterSeconds})
}

func OrderHasActive() *Error {
	return New(http.StatusConflict, "ORDER_HAS_ACTIVE", "User already has an active order", nil)
}

func InvalidStateTransition(current string) *Error {
	return New(http.StatusConflict, "INVALID_STATE_TRANSITION", "Operation is not allowed for the current order status", map[string]any{"current_status": current})
}

func InsufficientStock(items []map[string]any) *Error {
	return New(http.StatusConflict, "INSUFFICIENT_STOCK", "There is not enough stock for one or more products", map[string]any{"products": items})
}

func PromoInvalid() *Error {
	return New(http.StatusUnprocessableEntity, "PROMO_CODE_INVALID", "Promo code is unavailable", nil)
}

func PromoMinAmount(required, actual float64) *Error {
	return New(http.StatusUnprocessableEntity, "PROMO_CODE_MIN_AMOUNT", "Order amount is below the promo code minimum", map[string]any{
		"required_amount": required,
		"actual_amount":   actual,
	})
}

func Internal(cause error) *Error {
	return Wrap(http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", cause)
}
