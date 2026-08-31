package mapper

import (
	"velocity/internal/persistence/postgres/generated"
	"velocity/internal/transport/http/dto/response"
)

func ToSymbolResponse(
	s generated.Symbol,
) response.SymbolResponse {

	return response.SymbolResponse{
		Symbol:      s.Symbol,
		DisplayName: s.DisplayName,
		BaseAsset:   s.BaseAsset,
		QuoteAsset:  s.QuoteAsset,
		TickSize:    s.TickSize,
		LotSize:     s.LotSize,
		IsActive:    s.IsActive,
	}
}
