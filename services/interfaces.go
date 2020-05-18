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
	UpdateShiftV2(shiftData hestia.ShiftV2) (string, error)
	GetShiftsByTimestampV2(uid string, timestamp string) (shifts []hestia.ShiftV2, err error)
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
	Withdraw(withdrawParams models.WithdrawParams) (withdrawal models.WithdrawInfo, err error)
	Trade(tradeParams hestia.Trade) (txId string, err error)
	GetTradeStatus(tradeParams hestia.Trade) (tradeInfo hestia.ExchangeOrderInfo, err error)
	GetWithdrawalTxHash(withdrawParams models.WithdrawInfo) (txId string, err error)
	// Retrieves information on the status for a given deposit, using a txid or an address (some exchanges).
	DepositInfo(depositParams models.DepositParams) (depositInfo models.DepositInfo, err error)
	// Retrieves the maximum balance of an asset relative to the amount of base stable coins in the output coin's exchange.
	StockBalance(asset string) (balance models.BalanceResponse, err error)
}
