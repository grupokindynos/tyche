package tyche

//TycheBalance is the model for the GetBalance endpoint
type Balance struct {
	Balance string `json:"balance"`
}

//TycheRate is the model for storing rates in the cache
type Rate struct {
	Rate     float64 `json:"rate"`
	FromCoin string  `json:"fromcoin"`
	ToCoin   string  `json:"tocoin"`
	Amount   int64   `json:"amount"`
	Fee      int64   `json:"fee"`
	Address  string  `json:"address"`
}

//TychePrepare is the model for the response of the prepare endpoint
type Prepare struct {
	Token     string `json:"token"`
	Timestamp int64  `json:"timestamp"`
	Rate      Rate   `json:"rate"`
}

//Shift is the model for the response of the prepare endpoint
type Shift struct {
	Token          string `json:"token"`
	RawTransaction string `json:"raw_tx"`
}

//TycheReceive is the object that Tyche receives from frontend
type Receive struct {
	FromCoin string `json:"fromcoin"`
	ToCoin   string `json:"tocoin"`
	Amount   int64  `json:"amount"`
}
