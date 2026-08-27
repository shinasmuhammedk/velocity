package seed

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SellerProductSeed struct {
	SellerID int64
	Name     string
	Symbol   string
	Status   string
}

var SellerProducts = []SellerProductSeed{
	// Seller 3
	{SellerID: 3, Name: "Bitcoin", Symbol: "BTC", Status: "Active"},
	{SellerID: 3, Name: "Ethereum", Symbol: "ETH", Status: "Active"},
	{SellerID: 3, Name: "Solana", Symbol: "SOL", Status: "Active"},
	{SellerID: 3, Name: "Cardano", Symbol: "ADA", Status: "Active"},
	{SellerID: 3, Name: "Polkadot", Symbol: "DOT", Status: "Active"},
	{SellerID: 3, Name: "Dogecoin", Symbol: "DOGE", Status: "Active"},

	// Seller 4
	{SellerID: 4, Name: "Apple Inc.", Symbol: "AAPL", Status: "Active"},
	{SellerID: 4, Name: "Tesla Inc.", Symbol: "TSLA", Status: "Active"},
	{SellerID: 4, Name: "Amazon.com Inc.", Symbol: "AMZN", Status: "Active"},
	{SellerID: 4, Name: "Netflix Inc.", Symbol: "NFLX", Status: "Active"},
	{SellerID: 4, Name: "Intel Corp.", Symbol: "INTC", Status: "Active"},
	{SellerID: 4, Name: "Microsoft Corp.", Symbol: "MSFT", Status: "Active"},

	// Seller 15
	{SellerID: 15, Name: "Alphabet Inc.", Symbol: "GOOGL", Status: "Active"},
	{SellerID: 15, Name: "Meta Platforms Inc.", Symbol: "META", Status: "Active"},
	{SellerID: 15, Name: "Nvidia Corp.", Symbol: "NVDA", Status: "Active"},
	{SellerID: 15, Name: "Advanced Micro Devices Inc.", Symbol: "AMD", Status: "Active"},
	{SellerID: 15, Name: "Walt Disney Co.", Symbol: "DIS", Status: "Active"},
	{SellerID: 15, Name: "Starbucks Corp.", Symbol: "SBUX", Status: "Active"},

	// Seller 16
	{SellerID: 16, Name: "Ripple", Symbol: "XRP", Status: "Active"},
	{SellerID: 16, Name: "Chainlink", Symbol: "LINK", Status: "Active"},
	{SellerID: 16, Name: "Polygon", Symbol: "MATIC", Status: "Active"},
	{SellerID: 16, Name: "Uniswap", Symbol: "UNI", Status: "Active"},
	{SellerID: 16, Name: "Avalanche", Symbol: "AVAX", Status: "Active"},
	{SellerID: 16, Name: "Litecoin", Symbol: "LTC", Status: "Active"},
}

func SeedSellerProducts(ctx context.Context, pool *pgxpool.Pool) error {
	for _, p := range SellerProducts {
		_, err := pool.Exec(ctx, `
			INSERT INTO seller_products (seller_id, name, symbol, status)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (seller_id, symbol) DO NOTHING
		`, p.SellerID, p.Name, p.Symbol, p.Status)

		if err != nil {
			return err
		}
	}
	return nil
}
