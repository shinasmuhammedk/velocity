package response

type SymbolResponse struct {
	Symbol      string `json:"symbol"`
	DisplayName string `json:"display_name"`
	BaseAsset   string `json:"base_asset"`
	QuoteAsset  string `json:"quote_asset"`
	TickSize    int64  `json:"tick_size"`
	LotSize     int64  `json:"lot_size"`
	IsActive    bool   `json:"is_active"`
}