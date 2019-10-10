package microservices

//TycheBalance is the model for the GetBalance endpoint
type TycheBalance struct {
	Balance float64 `json:"balance"`
}

//TycheRate is the model for storing rates in the cache
type TycheRate struct {
	Rate     float64 `json:"rate"`
	FromCoin string  `json:"fromcoin"`
	ToCoin   string  `json:"tocoin"`
	Amount   int64   `json:"amount"`
	Fee      int64   `json:"fee"`
	Address  string  `json:"address"`
}

//TychePrepare is the model for the response of the prepare endpoint
type TychePrepare struct {
	Token     string    `json:"token"`
	Timestamp int64     `json:"timestamp"`
	Rate      TycheRate `json:"rate"`
}

//Shift is the model for the response of the prepare endpoint
type Shift struct {
	Token          string `json:"token"`
	RawTransaction string `json:"raw_tx"`
}

//TycheReceive is the object that Tyche receives from frontend
type TycheReceive struct {
	FromCoin string `json:"fromcoin"`
	ToCoin   string `json:"tocoin"`
	Amount   int64  `json:"amount"`
}
