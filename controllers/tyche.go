package controllers

import (
	"encoding/json"
	"errors"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/tyche/models"
	"sync"
	"time"

	"github.com/grupokindynos/common/obol"
	"github.com/grupokindynos/common/utils"
	"github.com/grupokindynos/olympus-utils/amount"
	"github.com/grupokindynos/tyche/services"
)

type TycheController struct {
	PrepareShifts map[string]models.PrepareShiftInfo
	mapLock       sync.RWMutex
}

func (s *TycheController) Status(uid string, payload []byte, params models.Params) (interface{}, error) {
	status, err := services.GetServicesStatus()
	if err != nil {
		return nil, err
	}
	return status.Shift, nil
}

func (s *TycheController) Balance(uid string, payload []byte, params models.Params) (interface{}, error) {
	balance, err := services.GetWalletBalance(params.Coin)
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
	amountHandler := amount.AmountType(prepareData.Amount)
	rate, err := obol.GetCoin2CoinRatesWithAmount(obol.ProductionURL, prepareData.FromCoin, prepareData.ToCoin, amountHandler.String())
	if err != nil {
		return nil, err
	}
	coinRates, err := obol.GetCoinRates(obol.ProductionURL, prepareData.FromCoin)
	if err != nil {
		return nil, err
	}
	polisRates, err := obol.GetCoinRates(obol.ProductionURL, "POLIS")
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
	fee, err := amount.NewAmount((fromCoinToUSD / polisRatesUSD) * 0.01)
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
	paymentAddress, err := services.GetNewPaymentAddress(prepareData.FromCoin)
	if err != nil {
		return nil, err
	}
	feeAddress, err := services.GetNewPaymentAddress("POLIS")
	if err != nil {
		return nil, err
	}
	payment := models.PaymentInfo{
		Address: paymentAddress,
		Amount:  prepareData.Amount,
	}
	feePayment := models.PaymentInfo{
		Address: feeAddress,
		Amount:  int64(fee.ToUnit(amount.AmountSats)),
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
	shift := hestia.Shift{
		ID:        storedShift.ID,
		UID:       uid,
		Status:    hestia.GetShiftStatusString(hestia.ShiftStatusPending),
		Timestamp: time.Now().Unix(),
		Payment: hestia.Payment{
			Address:       storedShift.Payment.Address,
			Amount:        storedShift.Payment.Amount,
			Coin:          storedShift.FromCoin,
			RawTx:         shiftPayment.RawTX,
			Txid:          "",
			Confirmations: 0,
		},
		FeePayment: hestia.Payment{
			Address:       storedShift.FeePayment.Address,
			Amount:        storedShift.FeePayment.Amount,
			Coin:          "POLIS",
			RawTx:         shiftPayment.FeeTX,
			Txid:          "",
			Confirmations: 0,
		},
		ToCoin:    storedShift.ToCoin,
		ToAmount:  storedShift.ToAmount,
		ToAddress: storedShift.ToAddress,
	}
	shiftid, err := services.UpdateShift(shift)
	if err != nil {
		return nil, err
	}
	return shiftid, nil
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
