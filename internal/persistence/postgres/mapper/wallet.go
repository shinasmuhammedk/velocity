package mapper

import (
	"strconv"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/transport/http/dto/response"
)

func ToWalletResponse(
	w generated.Wallet,
) response.WalletResponse {

	return response.WalletResponse{
		ID:        w.ID.String(),
		UserID:    strconv.FormatInt(w.UserID, 10),
		Asset:     w.Asset,
		Available: w.Available,
		Locked:    w.Locked,
		Total:     w.Available + w.Locked,
	}
}



func ToWalletTransactionResponse(
	t generated.WalletTransaction,
) response.WalletTransactionResponse {

	var tradeID *int64

	if t.TradeID.Valid {
		value := t.TradeID.Int64
		tradeID = &value
	}

	return response.WalletTransactionResponse{
		ID:        t.ID.String(),
		UserID:    strconv.FormatInt(t.UserID, 10),
		Asset:     t.Asset,
		Amount:    t.Amount,
		Type:      t.Type,
		TradeID:   tradeID,
		CreatedAt: t.CreatedAt,
	}
}