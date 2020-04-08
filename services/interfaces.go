package services

import (
	"github.com/grupokindynos/adrestia-go/models"
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
	ValidateRawTx(body plutus.ValidateRawTxReq) (valid bool, err error)
	SubmitPayment(body plutus.SendAddressBodyReq) (txid string, err error)
}

type AdrestiaService interface {
	GetAddress(coin string) (address models.AddressResponse, err error)
	GetPath(fromCoin string, toCoin string) (path models.PathResponse, err error)
}
