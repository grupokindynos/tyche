package processor

import (
	"github.com/grupokindynos/common/blockbook"
	coinFactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/hestia"
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

func getMissingTxId(coin string, address string, amount int64) (string, error) {
	coinConfig, _ := coinFactory.GetCoin(coin)
	blockBook := blockbook.NewBlockBookWrapper(coinConfig.Info.Blockbook)
	return blockBook.FindDepositTxId(address, amount)
}