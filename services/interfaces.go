package services

import (
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/plutus"
)

type HestiaService interface {
	GetShiftStatus() (hestia.Config, error)
	GetCoinsConfig() ([]hestia.Coin, error)
	UpdateShift(shiftData hestia.Shift) (string, error)
}

type PlutusService interface {
	GetWalletBalance(coin string) (plutus.Balance, error)
	GetNewPaymentAddress(coin string) (addr string, err error)
	DecodeRawTx(coin string, rawTx string) (txInfo interface{}, err error)
	SubmitPayment(body plutus.SendAddressBodyReq) (txid string, err error)
}
