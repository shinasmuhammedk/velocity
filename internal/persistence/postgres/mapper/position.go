package mapper

import (
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/transport/http/dto/response"
)

func ToPositionResponse(
	p generated.Position,
) response.PositionResponse {

	return response.PositionResponse{
		ID:        p.ID.String(),
		UserID:    p.UserID.String(),
		Symbol:    p.Symbol,
		Quantity:  p.Quantity,
		UpdatedAt: p.UpdatedAt,
	}
}
