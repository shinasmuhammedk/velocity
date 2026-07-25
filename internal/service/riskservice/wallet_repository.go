package riskservice

import "context"

type WalletRepository interface {
	GetAvailableBalance(ctx context.Context, userID string, asset string) (int64, error)
}
