package processor

import (
	"errors"
	cf "github.com/grupokindynos/common/coin-factory"
	coinFactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/explorer"
	"github.com/grupokindynos/common/hestia"
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
	if coinConfig.Info.Token && coinConfig.Info.Tag != "ETH" {
		coinConfig, _ = coinFactory.GetCoin("ETH")
	}
	explorerWrapper, _ := explorer.NewExplorerFactory().GetExplorerByCoin(*coinConfig)
	return explorerWrapper.FindDepositTxId(address, amount)
}

func getUserReceivedAmount(currency string, addr string, txId string) (float64, error) { // Currently doesnt support tokens
	var blockExplorer explorer.BlockBook
	coin, err := cf.GetCoin(currency)
	if err != nil {
		return 0.0, errors.New("Unable to get coin")
	}
	if coin.Info.Token && coin.Info.Tag != "ETH" {
		coin, _ = cf.GetCoin("ETH")
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
