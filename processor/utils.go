package processor

import (
	"errors"
	cf "github.com/grupokindynos/common/coin-factory"
	coinFactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/explorer"
	"github.com/grupokindynos/common/hestia"
	"math"
	"strconv"
	"strings"
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

func getUserReceivedAmount(currency string, addr string, txId string) (float64, error) {
	token := false
	coin, err := cf.GetCoin(currency)
	if err != nil {
		return 0.0, errors.New("unable to get coin")
	}
	if coin.Info.Token && coin.Info.Tag != "ETH" {
		coin, _ = cf.GetCoin("ETH")
		token = true
	}
	blockbookWrapper := explorer.NewBlockBookWrapper(coin.Info.Blockbook)

	res, err := blockbookWrapper.GetTx(txId)
	if err != nil {
		return 0.0, errors.New("Error while getting tx " + err.Error())
	}

	if res.Confirmations > 0 {
		if token {
			for _, transfer := range res.TokenTransfers {
				if strings.ToLower(transfer.To) == strings.ToLower(addr) {
					value, _ := strconv.Atoi(transfer.Value)
					return float64(value) / math.Pow10(transfer.Decimals), nil
				}
			}
		} else {
			for _, txVout := range res.Vout {
				for _, address := range txVout.Addresses {
					if address == addr {
						value, err := strconv.ParseFloat(txVout.Value, 64)
						if err != nil {
							return 0.0, err
						}
						return value / math.Pow10(8), nil
					}
				}
			}
		}
	}

	return 0.0, errors.New("tx not found or still not confirmed")
}
