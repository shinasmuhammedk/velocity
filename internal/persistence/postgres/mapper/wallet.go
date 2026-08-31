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
