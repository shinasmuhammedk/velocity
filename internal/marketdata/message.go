package marketdata

type Message struct {
	Type   string      `json:"type"`
	Symbol string      `json:"symbol"`
	Data   interface{} `json:"data"`
}
