package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/grupokindynos/common/blockbook"
	"github.com/shopspring/decimal"

	"github.com/grupokindynos/common/plutus"

	coinfactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/coin-factory/coins"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/tyche/models"

	cerrors "github.com/grupokindynos/common/errors"
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

func (s *TycheController) Status(string, []byte, models.Params) (interface{}, error) {
	if s.DevMode {
		return true, nil
	}
	status, err := s.Hestia.GetShiftStatus()
	if err != nil {
		return nil, err
	}
	return status.Shift.Service, nil
}

func (s *TycheController) Balance(_ string, _ []byte, params models.Params) (interface{}, error) {
	balance, err := s.Plutus.GetWalletBalance(params.Coin)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (s *TycheController) Prepare(uid string, payload []byte, _ models.Params) (interface{}, error) {
	var prepareData models.PrepareShiftRequest
	err := json.Unmarshal(payload, &prepareData)
	if err != nil {
		return nil, err
	}
	selectedCoin, err := GetServiceConfig(prepareData, s.Hestia, s.DevMode)
	if err != nil {
		return nil, err
	}

	amountTo, payment, feePayment, err := GetRates(prepareData, selectedCoin, s.Obol, s.Plutus)
	if err != nil {
		return nil, err
	}
	prepareResponse := models.PrepareShiftResponse{
		Payment:        payment,
		Fee:            feePayment,
		ReceivedAmount: amountTo.IntPart(),
	}
	prepareShift := models.PrepareShiftInfo{
		ID:         utils.RandomString(),
		FromCoin:   prepareData.FromCoin,
		Payment:    payment,
		FeePayment: feePayment,
		ToCoin:     prepareData.ToCoin,
		ToAddress:  prepareData.ToAddress,
		ToAmount:   amountTo.IntPart(),
		Timestamp:  time.Now().Unix(),
	}
	s.AddShiftToMap(uid, prepareShift)
	return prepareResponse, nil
}

func (s *TycheController) Store(uid string, payload []byte, _ models.Params) (interface{}, error) {
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
	shiftId, err := s.Hestia.UpdateShift(shift)
	if err != nil {
		return nil, err
	}
	go s.decodeAndCheckTx(shift, storedShift, shiftPayment.RawTX, shiftPayment.FeeTX)
	return shiftId, nil
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
	if coinConfig.Info.Token && coinConfig.Info.Tag != "ETH" {
		if coinConfig.Info.Token {
			coinConfig, _ = coinfactory.GetCoin("ETH")
		}
	}
	blockbookWrapper := blockbook.NewBlockBookWrapper(coinConfig.Info.Blockbook)
	return blockbookWrapper.SendTxWithMessage(rawTx)
}

func (s *TycheController) AddShiftToMap(uid string, shiftPrepare models.PrepareShiftInfo) {
	s.mapLock.Lock()
	s.PrepareShifts[uid] = shiftPrepare
	s.mapLock.Unlock()
}

func (s *TycheController) GetShiftFromMap(key string) (models.PrepareShiftInfo, error) {
	s.mapLock.Lock()
	shift, ok := s.PrepareShifts[key]
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

// OpenShift

func (s *TycheController) OpenBalance(_ string, _ []byte, params models.Params) (interface{}, error) {
	balance, err := s.Plutus.GetWalletBalance(params.Coin)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (s *TycheController) OpenStatus(_ string, _ []byte, _ models.Params) (interface{}, error) {
	status, err := s.Hestia.GetShiftStatus()
	if err != nil {
		return nil, err
	}
	return status.Shift.Service, nil
}

func (s *TycheController) OpenPrepare(uid string, payload []byte, _ models.Params) (interface{}, error) {
	res, err := s.PrepareV11(uid, payload, models.Params{})
	fmt.Println(res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *TycheController) OpenStore(uid string, payload []byte, _ models.Params) (interface{}, error) {
	res, err := s.StoreV11(uid, payload, models.Params{})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Tyche v2 API. Most important change is the use of ShiftId instead of UID as Mempool Map Key.
func (s *TycheController) PrepareV11(_ string, payload []byte, _ models.Params) (interface{}, error) {
	var prepareData models.PrepareShiftRequest
	err := json.Unmarshal(payload, &prepareData)
	if err != nil {
		return nil, err
	}
	selectedCoin, err := GetServiceConfig(prepareData, s.Hestia, true)
	if err != nil {
		return nil, err
	}

	amountTo, payment, feePayment, err := GetRates(prepareData, selectedCoin, s.Obol, s.Plutus)
	if err != nil {
		return nil, err
	}

	prepareShift := models.PrepareShiftInfo{
		ID:         utils.RandomString(),
		FromCoin:   prepareData.FromCoin,
		Payment:    payment,
		FeePayment: feePayment,
		ToCoin:     prepareData.ToCoin,
		ToAddress:  prepareData.ToAddress,
		ToAmount:   amountTo.IntPart(),
		Timestamp:  time.Now().Unix(),
	}
	prepareResponse := models.PrepareShiftResponse{
		Payment:        payment,
		Fee:            feePayment,
		ReceivedAmount: amountTo.IntPart(),
		ShiftId:        prepareShift.ID,
	}
	s.AddShiftToMap(prepareShift.ID, prepareShift)
	return prepareResponse, nil
}

func (s *TycheController) StoreV11(uid string, payload []byte, _ models.Params) (interface{}, error) {
	var shiftPayment models.StoreShiftV11
	err := json.Unmarshal(payload, &shiftPayment)
	if err != nil {
		return nil, err
	}
	storedShift, err := s.GetShiftFromMap(shiftPayment.ShiftId)
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

	s.RemoveShiftFromMap(shiftPayment.ShiftId)
	shiftid, err := s.Hestia.UpdateShift(shift)
	if err != nil {
		return nil, err
	}
	go s.decodeAndCheckTx(shift, storedShift, shiftPayment.RawTX, shiftPayment.FeeTX)
	return shiftid, nil
}

// Utility Functions
func GetServiceConfig(data models.PrepareShiftRequest, hestiaService services.HestiaService, dev bool) (selectedCoin hestia.Coin, err error) {
	// This function determines the service status, and if the conversion parameters are valid according to the
	// current service rules, in terms of coin availability and service availability. Returns the object containing
	// the desired target coin's hestia information to proceed with Shift Calculations

	// dev flag represents the development mode, should be true only when local Tyche testing is being performed.
	status, err := hestiaService.GetShiftStatus()
	if err != nil {
		err = cerrors.ErrorServiceUnavailable
		return
	}
	if !dev {
		if !status.Shift.Service {
			err = cerrors.ErrorServiceUnavailable
			return
		}
	}

	coinsConfig, err := hestiaService.GetCoinsConfig()
	if err != nil {
		err = cerrors.ErrorServiceUnavailable
		return
	}
	for _, coin := range coinsConfig {
		if coin.Ticker == data.FromCoin {
			selectedCoin = coin
		}
	}
	if selectedCoin.Ticker == "" {
		err = cerrors.ErrorServiceUnavailable
		return
	}
	if !selectedCoin.Shift.Available {
		err = cerrors.ErrorServiceUnavailable
		return
	}
	return
}

func GetRates(prepareData models.PrepareShiftRequest, selectedCoin hestia.Coin, obolService obol.ObolService, plutusService services.PlutusService) (amountTo decimal.Decimal, payment models.PaymentInfo, feePayment models.PaymentInfo, err error) {
	amountHandler := decimal.NewFromInt(prepareData.Amount)
	rate, err := obolService.GetCoin2CoinRatesWithAmount(prepareData.FromCoin, prepareData.ToCoin, amountHandler.String())
	if err != nil {
		err = cerrors.ErrorObtainingRates
		return
	}
	coinRates, err := obolService.GetCoinRates(prepareData.FromCoin)
	if err != nil {
		err = cerrors.ErrorObtainingRates
		return
	}
	polisRates, err := obolService.GetCoinRates("POLIS")
	if err != nil {
		err = cerrors.ErrorObtainingRates
		return
	}
	var coinRatesUSD float64
	for _, r := range coinRates {
		if r.Code == "USD" {
			coinRatesUSD = r.Rate
			break
		}
	}
	var polisRatesUSD float64
	for _, r := range polisRates {
		if r.Code == "USD" {
			polisRatesUSD = r.Rate
			break
		}
	}
	fromCoinToUSD := amountHandler.Mul(decimal.NewFromFloat(coinRatesUSD))
	fee := fromCoinToUSD.DivRound(decimal.NewFromFloat(polisRatesUSD), 0).Mul(decimal.NewFromFloat(selectedCoin.Shift.FeePercentage)).DivRound(decimal.NewFromInt(100), 0)
	rateAmountHandler := decimal.NewFromFloat(rate.AveragePrice)
	amountTo = amountHandler.Mul(rateAmountHandler)
	paymentAddress, err := plutusService.GetNewPaymentAddress(prepareData.FromCoin)
	if err != nil {
		err = cerrors.ErrorFillingPaymentInformation
		return
	}
	var feeAddress string
	if prepareData.FromCoin != "POLIS" {
		feeAddress, err = plutusService.GetNewPaymentAddress("POLIS")
		if err != nil {
			return
		}
	}

	payment = models.PaymentInfo{
		Address: paymentAddress,
		Amount:  prepareData.Amount,
	}
	if prepareData.FromCoin != "POLIS" {
		feePayment = models.PaymentInfo{
			Address: feeAddress,
			Amount:  fee.IntPart(),
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
	return
}
