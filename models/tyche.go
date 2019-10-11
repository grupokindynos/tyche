package tyche

//Balance is the model for the GetBalance endpoint
type Balance struct {
	Balance string `json:"balance"`
}

//Rate is the model for storing rates in the cache
type Rate struct {
	Rate     float64 `json:"rate"`
	FromCoin string  `json:"fromcoin"`
	ToCoin   string  `json:"tocoin"`
	Amount   int64   `json:"amount"`
	Fee      int64   `json:"fee"`
	Address  string  `json:"address"`
}

//Prepare is the model for the response of the prepare endpoint
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

//HestiaShift is the shift model that gets stored in the database
type HestiaShift struct {
	Confirmations  int `json:"rate"`
	Rate           float64
	ID             string
	Status         string
	Time           int
	PaymentAddress string
	PaymentCoin    string
	PaymentAmount  int64
	PaymentTXID    string
	ToAddress      string
	ToCoin         string
	ToAmount       int64
}

//TycheReceive is the object that Tyche receives from frontend
type Receive struct {
	FromCoin string `json:"fromcoin"`
	ToCoin   string `json:"tocoin"`
	Amount   int64  `json:"amount"`
}
