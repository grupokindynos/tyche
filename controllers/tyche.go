package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	coinfactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/coin-factory/coins"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/tyche/models"

	"github.com/grupokindynos/common/obol"
	"github.com/grupokindynos/common/utils"
	"github.com/grupokindynos/olympus-utils/amount"
	"github.com/grupokindynos/tyche/services"
)

type TycheController struct {
	PrepareShifts map[string]models.PrepareShiftInfo
	mapLock       sync.RWMutex
	txsAvailable  bool
	Hestia        services.HestiaService
	Plutus        services.PlutusService
	Obol          obol.ObolService
}

func (s *TycheController) Status(uid string, payload []byte, params models.Params) (interface{}, error) {
	status, err := s.Hestia.GetShiftStatus()
	if err != nil {
		return nil, err
	}
	return status.Shift.Service, nil
}

func (s *TycheController) Balance(uid string, payload []byte, params models.Params) (interface{}, error) {
	balance, err := s.Plutus.GetWalletBalance(params.Coin)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (s *TycheController) Prepare(uid string, payload []byte, params models.Params) (interface{}, error) {
	var prepareData models.PrepareShiftRequest
	err := json.Unmarshal(payload, &prepareData)
	if err != nil {
		return nil, err
	}
	status, err := s.Hestia.GetShiftStatus()
	if err != nil {
		return nil, err
	}
	if !status.Shift.Service {
		return nil, err
	}
	coinsConfig, err := s.Hestia.GetCoinsConfig()
	if err != nil {
		return nil, err
	}
	var selectedCoin hestia.Coin
	for _, coin := range coinsConfig {
		if coin.Ticker == prepareData.FromCoin {
			selectedCoin = coin
		}
	}
	if selectedCoin.Ticker == "" {
		return nil, err
	}
	if !selectedCoin.Shift.Available {
		return nil, err
	}

	amountHandler := amount.AmountType(prepareData.Amount)
	productionURL, err := s.Obol.GetProductionURL()
	rate, err := s.Obol.GetCoin2CoinRatesWithAmount(productionURL, prepareData.FromCoin, prepareData.ToCoin, amountHandler.String())
	if err != nil {
		return nil, err
	}
	coinRates, err := s.Obol.GetCoinRates(productionURL, prepareData.FromCoin)
	if err != nil {
		return nil, err
	}
	polisRates, err := s.Obol.GetCoinRates(productionURL, "POLIS")
	if err != nil {
		return nil, err
	}
	var coinRatesUSD float64
	for _, r := range coinRates {
		if r.Code == "USD" {
			coinRatesUSD = r.Rate
		}
	}
	var polisRatesUSD float64
	for _, r := range polisRates {
		if r.Code == "USD" {
			polisRatesUSD = r.Rate
		}
	}
	fromCoinToUSD := amountHandler.ToNormalUnit() * coinRatesUSD
	fee, err := amount.NewAmount((fromCoinToUSD / polisRatesUSD) * float64(selectedCoin.Shift.FeePercentage) / float64(100))
	if err != nil {
		return nil, err
	}
	rateAmountHandler, err := amount.NewAmount(rate.AveragePrice)
	if err != nil {
		return nil, err
	}
	ToAmount, err := amount.NewAmount(amountHandler.ToNormalUnit() / rateAmountHandler.ToNormalUnit())
	if err != nil {
		return nil, err
	}
	paymentAddress, err := s.Plutus.GetNewPaymentAddress(prepareData.FromCoin)
	if err != nil {
		return nil, err
	}
	var feeAddress string
	if prepareData.FromCoin != "POLIS" {
		feeAddress, err = s.Plutus.GetNewPaymentAddress("POLIS")
		if err != nil {
			return nil, err
		}
	}

	payment := models.PaymentInfo{
		Address: paymentAddress,
		Amount:  prepareData.Amount,
	}
	var feePayment models.PaymentInfo
	if prepareData.FromCoin != "POLIS" {
		feePayment = models.PaymentInfo{
			Address: feeAddress,
			Amount:  int64(fee.ToUnit(amount.AmountSats)),
		}
	}
	prepareResponse := models.PrepareShiftResponse{
		Payment:        payment,
		Fee:            feePayment,
		ReceivedAmount: int64(ToAmount.ToUnit(amount.AmountSats)),
	}
	prepareShift := models.PrepareShiftInfo{
		ID:         utils.RandomString(),
		FromCoin:   prepareData.FromCoin,
		Payment:    payment,
		FeePayment: feePayment,
		ToCoin:     prepareData.ToCoin,
		ToAddress:  prepareData.ToAddress,
		ToAmount:   int64(ToAmount.ToUnit(amount.AmountSats)),
		Timestamp:  time.Now().Unix(),
	}
	s.AddShiftToMap(uid, prepareShift)
	return prepareResponse, nil
}

func (s *TycheController) Store(uid string, payload []byte, params models.Params) (interface{}, error) {
	var shiftPayment models.StoreShift
	err := json.Unmarshal(payload, &shiftPayment)
	if err != nil {
		return nil, err
	}
	storedShift, err := s.GetShiftFromMap(uid)
	if err != nil {
		return nil, err
	}

	var feePayment hestia.Payment
	if storedShift.FromCoin != "POLIS" {
		feePayment = hestia.Payment{
			Address:       storedShift.FeePayment.Address,
			Amount:        storedShift.FeePayment.Amount,
			Coin:          "POLIS",
			Txid:          "",
			Confirmations: 0,
		}
	}

	shift := hestia.Shift{
		ID:        storedShift.ID,
		UID:       uid,
		Status:    hestia.GetShiftStatusString(hestia.ShiftStatusPending),
		Timestamp: time.Now().Unix(),
		Payment: hestia.Payment{
			Address:       storedShift.Payment.Address,
			Amount:        storedShift.Payment.Amount,
			Coin:          storedShift.FromCoin,
			Txid:          "",
			Confirmations: 0,
		},
		FeePayment:     feePayment,
		ToCoin:         storedShift.ToCoin,
		ToAmount:       storedShift.ToAmount,
		ToAddress:      storedShift.ToAddress,
		RefundAddr:     shiftPayment.RefundAddr,
		PaymentProof:   "",
		ProofTimestamp: 0,
	}
	s.RemoveShiftFromMap(uid)
	shiftid, err := s.Hestia.UpdateShift(shift)
	fmt.Println("stored shift : " + shiftid)
	if err != nil {
		return nil, err
	}
	//fmt.Println(shiftPayment.FeeTX)

	go s.decodeAndCheckTx(shift, storedShift, shiftPayment.RawTX, shiftPayment.FeeTX)
	return shiftid, nil
}

func (s *TycheController) decodeAndCheckTx(shiftData hestia.Shift, storedShiftData models.PrepareShiftInfo, rawTx string, feeTx string) {
	var feeTxId string

	if storedShiftData.FromCoin != "POLIS" {
		// Decode fee rawTx and verify
		feeOutputs, err := s.getRawTx("POLIS", feeTx)
		if err != nil {
			// If outputs fail, we should mark error, no spent anything.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
			log.Println(err)
			log.Println("221 - ")
			_, _ = s.Hestia.UpdateShift(shiftData)
			return
		}

		feeAmount := amount.AmountType(shiftData.FeePayment.Amount)
		err = s.verifyTransaction(feeOutputs, shiftData.FeePayment.Address, feeAmount)
		if err != nil {
			fmt.Println("232")
			// If verify fail, we should mark error, no spent anything.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
			_, err = s.Hestia.UpdateShift(shiftData)
			if err != nil {
				return
			}
			return
		}
		// Broadcast fee rawTx
		polisCoinConfig, err := coinfactory.GetCoin("POLIS")
		if err != nil {
			fmt.Println("244")
			// If get coin fail, we should mark error, no spent anything.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
			_, err = s.Hestia.UpdateShift(shiftData)
			if err != nil {
				return
			}
			return
		}
		feeTxId, err = s.broadCastTx(polisCoinConfig, feeTx)
		if err != nil {
			fmt.Println("255")
			// If broadcast fail, we should mark error, no spent anything.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
			_, err = s.Hestia.UpdateShift(shiftData)
			if err != nil {
				return
			}
			return
		}
	}

	// Decode payment rawTx and verify
	paymentOutputs, err := s.getRawTx(shiftData.Payment.Coin, rawTx)
	if err != nil {
		// If decode fail and payment is POLIS, we should mark error.
		fmt.Println("270")
		shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
		if storedShiftData.FromCoin != "POLIS" {
			// If decode fail and payment is not POLIS, we should mark Refund to send back the fees.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusRefund)
		}
		_, err = s.Hestia.UpdateShift(shiftData)
		if err != nil {
			return
		}
		return
	}
	paymentAmount := amount.AmountType(shiftData.Payment.Amount)
	err = s.verifyTransaction(paymentOutputs, shiftData.Payment.Address, paymentAmount)
	if err != nil {
		// If verify fail and payment is POLIS, we should mark error.
		fmt.Println("286")
		shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
		if storedShiftData.FromCoin != "POLIS" {
			// If decode fail and payment is not POLIS, we should mark Refund to send back the fees.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusRefund)
		}
		_, err = s.Hestia.UpdateShift(shiftData)
		if err != nil {
			return
		}
		return
	}
	// Broadcast rawTx
	coinConfig, err := coinfactory.GetCoin(shiftData.Payment.Coin)
	if err != nil {
		// If get coin fail and payment is POLIS, we should mark error.
		fmt.Println("302")
		shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
		if storedShiftData.FromCoin != "POLIS" {
			// If get coin fail and payment is not POLIS, we should mark Refund to send back the fees.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusRefund)
		}
		_, err = s.Hestia.UpdateShift(shiftData)
		if err != nil {
			return
		}
		return
	}
	paymentTxid, err := s.broadCastTx(coinConfig, rawTx)
	if err != nil {
		// If broadcast fail and payment is POLIS, we should mark error.
		fmt.Println("317")
		shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
		if storedShiftData.FromCoin != "POLIS" {
			// If broadcast fail and payment is not POLIS, we should mark Refund to send back the fees.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusRefund)
		}
		_, err = s.Hestia.UpdateShift(shiftData)
		if err != nil {
			return
		}
		return
	}
	// Update shift model include txid.
	shiftData.Payment.Txid = paymentTxid
	shiftData.FeePayment.Txid = feeTxId
	_, err = s.Hestia.UpdateShift(shiftData)
	if err != nil {
		return
	}
}

func (s *TycheController) getRawTx(coin string, rawTx string) ([]hestia.Outputs, error) {
	log.Println("Entra getRawTx")
	rawData, err := s.Plutus.DecodeRawTx(coin, rawTx)
	if err != nil {
		log.Println("Error getRaw")
		log.Println(err)
		return nil, err
	}
	log.Println("llega al switch")
	switch coin {
	case "POLIS":
		fmt.Println("Es polis")
		rawInfo, err := json.Marshal(rawData)
		if err != nil {
			return nil, err
		}
		var txInfo hestia.PolisTxInfo
		err = json.Unmarshal(rawInfo, &txInfo)
		if err != nil {
			return nil, err
		}
		var outputs []hestia.Outputs
		for _, out := range txInfo.Vout {
			amountHandler, err := amount.NewAmount(out.Value)
			if err != nil {
				return nil, err
			}
			newOutput := hestia.Outputs{
				Address: out.ScriptPubKey.Addresses[0],
				Amount:  amountHandler,
			}
			outputs = append(outputs, newOutput)
		}
		return outputs, nil
	case "BTC":
		fmt.Println("es bitcoin")
		rawInfo, err := json.Marshal(rawData)
		if err != nil {
			return nil, err
		}
		var txInfo hestia.BitcoinTxInfo
		err = json.Unmarshal(rawInfo, &txInfo)
		if err != nil {
			return nil, err
		}
		var outputs []hestia.Outputs
		for _, out := range txInfo.Vout {
			amountHandler, err := amount.NewAmount(out.Value)
			if err != nil {
				return nil, err
			}
			newOutput := hestia.Outputs{
				Address: out.ScriptPubKey.Addresses[0],
				Amount:  amountHandler,
			}
			outputs = append(outputs, newOutput)
		}
		return outputs, nil
	case "DASH":
		fmt.Println("es dash")
		rawInfo, err := json.Marshal(rawData)
		if err != nil {
			return nil, err
		}
		var txInfo hestia.DashTxInfo
		err = json.Unmarshal(rawInfo, &txInfo)
		if err != nil {
			return nil, err
		}
		var outputs []hestia.Outputs
		for _, out := range txInfo.Vout {
			amountHandler, err := amount.NewAmount(out.Value)
			if err != nil {
				return nil, err
			}
			newOutput := hestia.Outputs{
				Address: out.ScriptPubKey.Addresses[0],
				Amount:  amountHandler,
			}
			outputs = append(outputs, newOutput)
		}
		return outputs, nil
	}
	return nil, nil
}

func (s *TycheController) broadCastTx(coinConfig *coins.Coin, rawTx string) (txid string, err error) {
	fmt.Println("Entro broadcast")
	if !s.txsAvailable {
		return "not published due no-txs flag", nil
	}

	resp, err := http.Get(coinConfig.BlockExplorer + "/api/v2/sendtx/" + rawTx)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var response models.BlockbookBroadcastResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}
	if response.Error != "" {
		return "", errors.New(response.Error)
	}
	return response.Result, nil
}

func (s *TycheController) verifyTransaction(outputs []hestia.Outputs, address string, amount amount.AmountType) error {
	var isAddressOnTx, isAmountCorrect = false, false
	for _, output := range outputs {
		if output.Address == address {
			isAddressOnTx = true
		}
		if output.Amount == amount {
			isAmountCorrect = true
		}
	}
	if isAddressOnTx == false {
		return errors.New("no matching address in raw tx")
	}
	if isAmountCorrect == false {
		return errors.New("incorrect amount in raw tx")
	}
	return nil
}

func (s *TycheController) AddShiftToMap(uid string, shiftPrepare models.PrepareShiftInfo) {
	s.mapLock.Lock()
	s.PrepareShifts[uid] = shiftPrepare
	s.mapLock.Unlock()
}

func (s *TycheController) GetShiftFromMap(uid string) (models.PrepareShiftInfo, error) {
	s.mapLock.Lock()
	shift, ok := s.PrepareShifts[uid]
	s.mapLock.Unlock()
	if !ok {
		return models.PrepareShiftInfo{}, errors.New("no shift found on cache")
	}
	return shift, nil
}

func (s *TycheController) RemoveShiftFromMap(uid string) {
	s.mapLock.Lock()
	delete(s.PrepareShifts, uid)
	s.mapLock.Unlock()
}
