package errors

var (

	// Generic

	ErrInternal = New(
		CodeInternal,
		"internal server error",
	)

	ErrValidation = New(
		CodeValidation,
		"validation failed",
	)

	ErrUnauthorized = New(
		CodeUnauthorized,
		"unauthorized",
	)

	ErrForbidden = New(
		CodeForbidden,
		"forbidden",
	)

	ErrNotFound = New(
		CodeNotFound,
		"resource not found",
	)

	ErrConflict = New(
		CodeConflict,
		"resource conflict",
	)

	// Database

	ErrDatabase = New(
		CodeDatabase,
		"database error",
	)

	// Matching Engine

	ErrEngine = New(
		CodeEngine,
		"matching engine error",
	)

	// Orders

	ErrInvalidOrder = New(
		CodeInvalidOrder,
		"invalid order",
	)

	ErrOrderNotFound = New(
		CodeOrderNotFound,
		"order not found",
	)

	ErrOrderFilled = New(
		CodeOrderFilled,
		"order already filled",
	)

	ErrOrderCancelled = New(
		CodeOrderCancelled,
		"order already cancelled",
	)

	// Stop Orders

	ErrStopOrderNotFound = New(
		CodeStopOrderNotFound,
		"stop order not found",
	)
)
