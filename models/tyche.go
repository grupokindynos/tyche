package models

import "github.com/grupokindynos/olympus-utils/amount"

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

type PrepareShiftResponse struct {
	Payment        PaymentInfo `json:"payment"`
	Fee            PaymentInfo `json:"fee"`
	ReceivedAmount int64       `json:"received_amount"`
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

type PaymentInfo struct {
	Address string `json:"address"`
	Amount  int64  `json:"amount"`
}

type StoreShift struct {
	RawTX      string `json:"raw_tx"`
	FeeTX      string `json:"fee_tx"`
	RefundAddr string `json:"refund_addr"`
}

type Outputs struct {
	Address string
	Amount  amount.AmountType
}
