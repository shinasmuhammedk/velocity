package errors

type Code string

const (

	// Generic
	CodeInternal     Code = "INTERNAL_ERROR"
	CodeValidation   Code = "VALIDATION_ERROR"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeForbidden    Code = "FORBIDDEN"
	CodeNotFound     Code = "NOT_FOUND"
	CodeConflict     Code = "CONFLICT"

	// Database
	CodeDatabase Code = "DATABASE_ERROR"

	// Orders
	CodeInvalidOrder   Code = "INVALID_ORDER"
	CodeOrderNotFound  Code = "ORDER_NOT_FOUND"
	CodeOrderCancelled Code = "ORDER_CANCELLED"
	CodeOrderFilled    Code = "ORDER_FILLED"

	// Engine
	CodeEngine Code = "ENGINE_ERROR"
    
    // Stop Orders
	CodeStopOrderNotFound Code = "STOP_ORDER_NOT_FOUND"
)