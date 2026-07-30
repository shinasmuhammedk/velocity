package seed

type UserSeed struct {
	ID       string
	Email    string
	Password string
}

type SymbolSeed struct {
	Symbol      string
	DisplayName string
	BaseAsset   string
	QuoteAsset  string
	TickSize    int64
	LotSize     int64
	IsActive    bool
}

type WalletSeed struct {
	UserID    string
	Asset     string
	Available int64
	Locked    int64
}

type PositionSeed struct {
	UserID   string
	Symbol   string
	Quantity int64
}

var Users = []UserSeed{
	{
		ID:       "a0d34cbb-524f-45bd-bfc7-921b7f9ffda0",
		Email:    "test1@velocity.dev",
		Password: "dummy-password-hash",
	},
	{
		ID:       "5b31d252-99ad-4ebe-a842-e593050414c4",
		Email:    "test2@velocity.dev",
		Password: "dummy-password-hash",
	},
	{
		ID:       "367bc8a3-e320-4cc6-adfd-55248b9947ab",
		Email:    "marketmaker@velocity.dev",
		Password: "dummy-password-hash",
	},
}

var Symbols = []SymbolSeed{
	{
		Symbol:      "BTCUSDT",
		DisplayName: "Bitcoin / Tether",
		BaseAsset:   "BTC",
		QuoteAsset:  "USDT",
		TickSize:    1,
		LotSize:     1,
		IsActive:    true,
	},
}

var Wallets = []WalletSeed{
	// ---------- test1 ----------
	{
		UserID:    "a0d34cbb-524f-45bd-bfc7-921b7f9ffda0",
		Asset:     "BTC",
		Available: 5,
		Locked:    0,
	},
	{
		UserID:    "a0d34cbb-524f-45bd-bfc7-921b7f9ffda0",
		Asset:     "USDT",
		Available: 100000000,
		Locked:    0,
	},

	// ---------- test2 ----------
	{
		UserID:    "5b31d252-99ad-4ebe-a842-e593050414c4",
		Asset:     "BTC",
		Available: 5,
		Locked:    0,
	},
	{
		UserID:    "5b31d252-99ad-4ebe-a842-e593050414c4",
		Asset:     "USDT",
		Available: 100000000,
		Locked:    0,
	},

	// ---------- market maker ----------
	{
		UserID:    "367bc8a3-e320-4cc6-adfd-55248b9947ab",
		Asset:     "BTC",
		Available: 100,
		Locked:    0,
	},
	{
		UserID:    "367bc8a3-e320-4cc6-adfd-55248b9947ab",
		Asset:     "USDT",
		Available: 1000000000,
		Locked:    0,
	},
}

var Positions = []PositionSeed{
	{
		UserID:   "a0d34cbb-524f-45bd-bfc7-921b7f9ffda0",
		Symbol:   "BTCUSDT",
		Quantity: 0,
	},
	{
		UserID:   "5b31d252-99ad-4ebe-a842-e593050414c4",
		Symbol:   "BTCUSDT",
		Quantity: 0,
	},
	{
		UserID:   "367bc8a3-e320-4cc6-adfd-55248b9947ab",
		Symbol:   "BTCUSDT",
		Quantity: 0,
	},
}