package seed

type UserSeed struct {
	ID       int64
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
	UserID    int64
	Asset     string
	Available int64
	Locked    int64
}

type PositionSeed struct {
	UserID   int64
	Symbol   string
	Quantity int64
}

var Users = []UserSeed{
	{
		ID:       1,
		Email:    "test1@velocity.dev",
		Password: "dummy-password-hash",
	},
	{
		ID:       2,
		Email:    "test2@velocity.dev",
		Password: "dummy-password-hash",
	},
	{
		ID:       3,
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
		UserID:    1,
		Asset:     "BTC",
		Available: 5,
		Locked:    0,
	},
	{
		UserID:    1,
		Asset:     "USDT",
		Available: 100000000,
		Locked:    0,
	},

	// ---------- test2 ----------
	{
		UserID:    2,
		Asset:     "BTC",
		Available: 5,
		Locked:    0,
	},
	{
		UserID:    2,
		Asset:     "USDT",
		Available: 100000000,
		Locked:    0,
	},

	// ---------- market maker ----------
	{
		UserID:    3,
		Asset:     "BTC",
		Available: 100,
		Locked:    0,
	},
	{
		UserID:    3,
		Asset:     "USDT",
		Available: 1000000000,
		Locked:    0,
	},
}

var Positions = []PositionSeed{

	{
		UserID:   1,
		Symbol:   "BTCUSDT",
		Quantity: 0,
	},

	{
		UserID:   2,
		Symbol:   "BTCUSDT",
		Quantity: 0,
	},

	{
		UserID:   3,
		Symbol:   "BTCUSDT",
		Quantity: 0,
	},
}
