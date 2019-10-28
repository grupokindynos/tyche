package tyche

import "github.com/grupokindynos/common/hestia"

//Balance is the model for the GetBalance endpoint
type Balance struct {
	Balance string `json:"balance"`
}

//Prepare is the model for the response of the prepare endpoint
type Prepare struct {
	Token     string           `json:"token"`
	Timestamp int64            `json:"timestamp"`
	Rate      hestia.ShiftRate `json:"rate"`
}

//Shift is the model for the response of the prepare endpoint
type Shift struct {
	Token          string `json:"token"`
	RawTransaction string `json:"raw_tx"`
}

//Receive is the object that Tyche receives from frontend
type Receive struct {
	FromCoin string `json:"fromcoin"`
	ToCoin   string `json:"tocoin"`
	Amount   int64  `json:"amount"`
	FeeCoin  string `json:"feecoin"`
}

//NewShift is the model for new Shifts
type NewShift struct {
	RawTX string `json:"raw_tx"`
	FeeTX string `json:"fee_tx"`
	Token string  `json:"token"`
}
