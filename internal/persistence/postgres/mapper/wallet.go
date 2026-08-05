package mapper

import (
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/transport/http/dto/response"
)

func ToWalletResponse(
	w generated.Wallet,
) response.WalletResponse {

	return response.WalletResponse{
		ID:        w.ID.String(),
		Asset:     w.Asset,
		Available: w.Available,
		Locked:    w.Locked,
		Total:     w.Available + w.Locked,
	}
}
