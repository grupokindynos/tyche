package microservices

//TycheBalance is the model for the GetBalance endpoint
type TycheBalance struct {
	Balance float64 `json:"balance"`
}

//TycheRate is the model for storing rates in the cache
type TycheRate struct {
	Rate      float64 `json:"rate"`
	Timestamp int64   `json:"timestamp"`
	FromCoin  string  `json:"fromcoin"`
	ToCoin    string  `json:"tocoin"`
	Amount    string  `json:"amount"`
}

//TychePrepare is the model for the response of the prepare endpoint
type TychePrepare struct {
	Token string    `json:"token"`
	Rate  TycheRate `json:"rate"`
}
