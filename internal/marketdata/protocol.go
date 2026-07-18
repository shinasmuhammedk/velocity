package marketdata

type ClientRequest struct {
	Action string `json:"action"`
	Symbol string `json:"symbol,omitempty"`
}

type ServerResponse struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}