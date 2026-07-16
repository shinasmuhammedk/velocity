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

	ErrInvalidStopPrice = New(
		CodeInvalidStopPrice,
		"invalid stop price",
	)

	ErrPostOnlyMustBeLimit = New(
		CodePostOnlyViolation,
		"post only orders must be limit orders",
	)

	ErrBuyStopBelowMarket = New(
		CodeInvalidStopTrigger,
		"buy stop must be above market price",
	)

	ErrSellStopAboveMarket = New(
		CodeInvalidStopTrigger,
		"sell stop must be below market price",
	)

	ErrUserNotFound = New(
		CodeUserNotFound,
		"user not found",
	)

	// Symbols

	ErrSymbolNotFound = New(
		CodeSymbolNotFound,
		"symbol not found",
	)

	ErrSymbolInactive = New(
		CodeSymbolInactive,
		"symbol inactive",
	)

	ErrEngineUnavailable = New(
		CodeEngineUnavailable,
		"symbol engine unavailable",
	)

	ErrOrderNotCancelable = New(
		CodeOrderNotCancelable,
		"order cannot be cancelled",
	)

	ErrOrderModificationNotAllowed = New(
		CodeOrderModificationNotAllowed,
		"only open orders can be modified",
	)

	ErrQuantityTooLow = New(
		CodeQuantityTooLow,
		"quantity cannot be less than filled quantity",
	)

	ErrConfigInvalid = New(
		CodeConfigInvalid,
		"invalid configuration",
	)

	ErrConfigMissing = New(
		CodeConfigMissing,
		"required configuration missing",
	)
)
