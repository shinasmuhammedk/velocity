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
	// Orders
	CodeInvalidOrder       Code = "INVALID_ORDER"
	CodeOrderNotFound      Code = "ORDER_NOT_FOUND"
	CodeOrderCancelled     Code = "ORDER_CANCELLED"
	CodeOrderFilled        Code = "ORDER_FILLED"
	CodeInvalidStopPrice   Code = "INVALID_STOP_PRICE"
	CodePostOnlyViolation  Code = "POST_ONLY_VIOLATION"
	CodeInvalidStopTrigger Code = "INVALID_STOP_TRIGGER"

	// Engine
	CodeEngine Code = "ENGINE_ERROR"

	// Stop Orders
	CodeStopOrderNotFound Code = "STOP_ORDER_NOT_FOUND"

	// Users
	CodeUserNotFound Code = "USER_NOT_FOUND"

	// Symbols
	CodeSymbolNotFound Code = "SYMBOL_NOT_FOUND"
	CodeSymbolInactive Code = "SYMBOL_INACTIVE"

	// Engine
	CodeEngineUnavailable Code = "ENGINE_UNAVAILABLE"

	CodeOrderNotCancelable Code = "ORDER_NOT_CANCELABLE"
	// Orders
	CodeOrderModificationNotAllowed Code = "ORDER_MODIFICATION_NOT_ALLOWED"
	CodeQuantityTooLow              Code = "QUANTITY_TOO_LOW"
    
    
    // Configuration
CodeConfigInvalid Code = "CONFIG_INVALID"
CodeConfigMissing Code = "CONFIG_MISSING"
)
