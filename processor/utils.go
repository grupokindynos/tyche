package processor

import (
	"errors"
	"github.com/grupokindynos/common/blockbook"
	coinFactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/hestia"
	cf "github.com/grupokindynos/common/coin-factory"
	"strconv"
)

func checkTxId(payment *hestia.Payment) error {
	if payment.Txid == "" {
		txId, err := getMissingTxId(payment.Coin, payment.Address, payment.Amount)
		if err != nil {
			return err
		}
		payment.Txid = txId
	}
	return nil
}

func checkTxIdWithFee(payment *hestia.PaymentWithFee) error {
	if payment.Txid == "" {
		txId, err := getMissingTxId(payment.Coin, payment.Address, payment.Amount)
		if err != nil {
			return err
		}
		payment.Txid = txId
	}
	return nil
}

func getMissingTxId(coin string, address string, amount int64) (string, error) {
	coinConfig, _ := coinFactory.GetCoin(coin)
	blockBook := blockbook.NewBlockBookWrapper(coinConfig.Info.Blockbook)
	return blockBook.FindDepositTxId(address, amount)
}

func getUserReceivedAmount(currency string, addr string, txId string) (float64, error) { // Currently doesnt support tokens
	var blockExplorer blockbook.BlockBook
	coin, err := cf.GetCoin(currency)
	if err != nil {
		return 0.0, errors.New("Unable to get coin")
	}
	blockExplorer.Url = "https://" + coin.BlockchainInfo.ExternalSource
	res, err := blockExplorer.GetTx(txId)
	if err != nil {
		return 0.0, errors.New("Error while getting tx " + err.Error())
	}
	if res.Confirmations > 0 {
		for _, txVout := range res.Vout {
			for _, address := range txVout.Addresses {
				if address == addr {
					value, err := strconv.ParseFloat(txVout.Value, 64)
					if err != nil {
						return 0.0, err
					}
					return value, nil
				}
			}
		}
	}

	return 0.0, errors.New("tx not found or still not confirmed")
}