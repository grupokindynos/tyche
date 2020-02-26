package controllers

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/grupokindynos/common/blockbook"
	"github.com/olympus-protocol/ogen/utils/amount"

	"github.com/grupokindynos/common/plutus"

	coinfactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/coin-factory/coins"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/tyche/models"

	"github.com/grupokindynos/common/obol"
	"github.com/grupokindynos/common/utils"
	"github.com/grupokindynos/tyche/services"
)

type TycheController struct {
	PrepareShifts map[string]models.PrepareShiftInfo
	mapLock       sync.RWMutex
	TxsAvailable  bool
	Hestia        services.HestiaService
	Plutus        services.PlutusService
	Obol          obol.ObolService
	DevMode       bool
}

func (s *TycheController) Status(uid string, payload []byte, params models.Params) (interface{}, error) {
	if s.DevMode {
		return true, nil
	}
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
	if !s.DevMode {
		if !status.Shift.Service {
			return nil, err
		}
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
	rate, err := s.Obol.GetCoin2CoinRatesWithAmount(prepareData.FromCoin, prepareData.ToCoin, amountHandler.String())
	if err != nil {
		return nil, err
	}
	coinRates, err := s.Obol.GetCoinRates(prepareData.FromCoin)
	if err != nil {
		return nil, err
	}
	polisRates, err := s.Obol.GetCoinRates("POLIS")
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
	ToAmount, err := amount.NewAmount(amountHandler.ToNormalUnit() * rateAmountHandler.ToNormalUnit())
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
			HasFee:  true,
		}
	}
	// Eliminates payment fee when converting to Polis.
	if prepareData.ToCoin == "POLIS" {
		feePayment = models.PaymentInfo{
			Address: "no fee for polis",
			Amount:  0,
			HasFee:  false,
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

	if storedShift.ToCoin == "POLIS" {
		feePayment = hestia.Payment{
			Address:       "N/A",                         // no fee por Polis conversion
			Amount:        storedShift.FeePayment.Amount, // should be aways 0.0
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
	if err != nil {
		return nil, err
	}
	go s.decodeAndCheckTx(shift, storedShift, shiftPayment.RawTX, shiftPayment.FeeTX)
	return shiftid, nil
}

func (s *TycheController) decodeAndCheckTx(shiftData hestia.Shift, storedShiftData models.PrepareShiftInfo, rawTx string, feeTx string) {
	// Only decode raw transaction if tx does not come from or to POLIS
	// Conditional statement is logic for the negation for "if its from or to polis"
	if storedShiftData.FromCoin != "POLIS" && storedShiftData.ToCoin != "POLIS" {
		// Validate Payment FeeRawTx
		body := plutus.ValidateRawTxReq{
			Coin:    shiftData.FeePayment.Coin,
			RawTx:   feeTx,
			Amount:  shiftData.FeePayment.Amount,
			Address: shiftData.FeePayment.Address,
		}
		valid, err := s.Plutus.ValidateRawTx(body)

		if err != nil {
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
			_, err = s.Hestia.UpdateShift(shiftData)
			if err != nil {
				return
			}
			return
		}

		if !valid {
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
			// If get coin fail, we should mark error, no spent anything.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
			_, err = s.Hestia.UpdateShift(shiftData)
			if err != nil {
				return
			}
			return
		}
		feeTxID, err, _ := s.broadCastTx(polisCoinConfig, feeTx)
		if err != nil {
			// If broadcast fail, we should mark error, no spent anything.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
			_, err = s.Hestia.UpdateShift(shiftData)
			if err != nil {
				return
			}
			return
		}
		shiftData.FeePayment.Txid = feeTxID
	}
	// Validate Payment RawTx
	body := plutus.ValidateRawTxReq{
		Coin:    shiftData.Payment.Coin,
		RawTx:   rawTx,
		Amount:  shiftData.Payment.Amount,
		Address: shiftData.Payment.Address,
	}
	valid, err := s.Plutus.ValidateRawTx(body)
	if err != nil {
		// If decode fail and payment is POLIS, we should mark error.
		shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
		if storedShiftData.FromCoin != "POLIS" && storedShiftData.ToCoin != "POLIS" {
			// If decode fail and payment is not POLIS, we should mark Refund to send back the fees.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusRefund)
		}
		_, err = s.Hestia.UpdateShift(shiftData)
		if err != nil {
			return
		}
		return
	}
	if !valid {
		if storedShiftData.FromCoin != "POLIS" && storedShiftData.ToCoin != "POLIS" {
			// If is not valid and payment is not POLIS, we should mark Refund to send back the fees.
			shiftData.Status = hestia.GetShiftStatusString(hestia.ShiftStatusRefund)
		}
	}
	// Broadcast rawTx
	coinConfig, err := coinfactory.GetCoin(shiftData.Payment.Coin)
	if err != nil {
		// If get coin fail and payment is POLIS, we should mark error.
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
	paymentTxid, err, message := s.broadCastTx(coinConfig, rawTx)
	if err != nil {
		// If broadcast fail and payment is POLIS, we should mark error.
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
	shiftData.Message = message
	// Update shift model include txid.
	shiftData.Payment.Txid = paymentTxid
	_, err = s.Hestia.UpdateShift(shiftData)
	if err != nil {
		return
	}
}

func (s *TycheController) broadCastTx(coinConfig *coins.Coin, rawTx string) (string, error, string) {
	if !s.TxsAvailable {
		return "not published due no-txs flag", nil, ""
	}
	blockbookWrapper := blockbook.NewBlockBookWrapper(coinConfig.Info.Blockbook)
	return blockbookWrapper.SendTxWithMessage(rawTx)
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
