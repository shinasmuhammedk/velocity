package mapper

import (
	"strconv"

	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/transport/http/dto/response"
)

func ToPositionResponse(
	p generated.Position,
) response.PositionResponse {

	return response.PositionResponse{
		ID:        p.ID.String(),
		UserID:    strconv.FormatInt(p.UserID, 10),
		Symbol:    p.Symbol,
		Quantity:  p.Quantity,
		UpdatedAt: p.UpdatedAt,
	}
}