package models

import "github.com/grupokindynos/adrestia-go/models"

type Params struct {
	Coin string
}

type Balance struct {
	Balance string `json:"balance"`
}

type Shift struct {
	Token          string `json:"token"`
	RawTransaction string `json:"raw_tx"`
}

type PrepareShiftRequest struct {
	FromCoin  string `json:"from_coin"`
	Amount    int64  `json:"amount"`
	ToCoin    string `json:"to_coin"`
	ToAddress string `json:"to_address"`
}

type OpenPrepareShiftRequest struct {
	FromCoin  string `json:"from_coin"`
	Amount    int64  `json:"amount"`
	ToCoin    string `json:"to_coin"`
	ToAddress string `json:"to_address"`
	UserId    string `json:"user_id"`
}

type PrepareShiftResponse struct {
	Payment        PaymentInfo `json:"payment"`
	Fee            PaymentInfo `json:"fee"`
	ReceivedAmount int64       `json:"received_amount"`
	ShiftId			string `json:"shift_id"`
}

type PrepareShiftResponseV2 struct {
	Payment        PaymentInfoV2 `json:"payment"`
	ReceivedAmount int64       `json:"received_amount"`
	ShiftId			string `json:"shift_id"`
}

type PrepareShiftInfo struct {
	ID         string      `json:"id"`
	FromCoin   string      `json:"from_coin"`
	Payment    PaymentInfo `json:"payment"`
	FeePayment PaymentInfo `json:"fee_payment"`
	ToCoin     string      `json:"to_coin"`
	ToAddress  string      `json:"to_address"`
	ToAmount   int64       `json:"to_amount"`
	Timestamp  int64       `json:"timestamp"`
}

type PrepareShiftInfoV2 struct {
	ID         string      `json:"id"`
	FromCoin   string      `json:"from_coin"`
	Payment    PaymentInfoV2 `json:"payment"`
	ToCoin     string      `json:"to_coin"`
	ToAddress  string      `json:"to_address"`
	ToAmount   int64       `json:"to_amount"`
	ToAmountUSD   int64       `json:"to_amount_usd"`
	Timestamp  int64       `json:"timestamp"`
	Path models.PathResponse `json:"paths"`
}

type PaymentInfo struct {
	Address string `json:"address"`
	Amount  int64  `json:"amount"`
	HasFee  bool   `json:"has_fee"`
}

type PaymentInfoV2 struct {
	Address models.AddressResponse `json:"address"`
	Fee int64 `json:"fee"`
	Amount  int64  `json:"amount"`
	Total  int64  `json:"total"`
	HasFee  bool   `json:"has_fee"`
	Rate int64 `json:"rate"`
	FiatInfo ExpectedFiatAmount `json:"fiat_info"`
	Conversions models.PathResponse `json:"conversions"`
}

type ExpectedFiatAmount struct {
	Amount float64 `json:"amount"`
	Fee float64 `json:"fee"`
}

type StoreShift struct {
	RawTX      string `json:"raw_tx"`
	FeeTX      string `json:"fee_tx"`
	RefundAddr string `json:"refund_addr"`
	HasFee     bool   `json:"has_fee"`
}

type StoreShiftV11 struct {
	RawTX      	string `json:"raw_tx"`
	FeeTX      	string `json:"fee_tx"`
	RefundAddr 	string `json:"refund_addr"`
	HasFee     	bool   `json:"has_fee"`
	ShiftId		string `json:"shift_id"`
}

type StoreShiftV2 struct {
	RawTX      	string `json:"raw_tx"`
	FeeTX      	string `json:"fee_tx"`
	RefundAddr 	string `json:"refund_addr"`
	HasFee     	bool   `json:"has_fee"`
	ShiftId		string `json:"shift_id"`
}