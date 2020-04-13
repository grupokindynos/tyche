package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	cerrors "github.com/grupokindynos/common/errors"
	"github.com/grupokindynos/tyche/services"
	"log"
	"sync"
	"time"

	"github.com/grupokindynos/common/blockbook"
	"github.com/olympus-protocol/ogen/utils/amount"

	"github.com/grupokindynos/common/plutus"

	coinFactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/coin-factory/coins"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/tyche/models"

	"github.com/grupokindynos/adrestia-go/exchanges"
	"github.com/grupokindynos/common/obol"
	"github.com/grupokindynos/common/utils"
	//"github.com/grupokindynos/tyche/services"
)

type TycheControllerV2 struct {
	PrepareShifts map[string]models.PrepareShiftInfoV2
	mapLock       sync.RWMutex
	TxsAvailable  bool
	Hestia        services.HestiaService
	Plutus        services.PlutusService
	Obol          obol.ObolService
	Adrestia      services.AdrestiaService
	DevMode       bool
	ExFactory     *exchanges.ExchangeFactory
}

func (s *TycheControllerV2) StatusV2(string, []byte, models.Params) (interface{}, error) {
	if s.DevMode {
		return true, nil
	}
	status, err := s.Hestia.GetShiftStatus()
	if err != nil {
		return nil, err
	}
	return status.Shift.Service, nil
}

func (s *TycheControllerV2) BalanceV2(_ string, _ []byte, params models.Params) (interface{}, error) {
	balance, err := s.Plutus.GetWalletBalance(params.Coin)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func (s *TycheControllerV2) broadCastTx(coinConfig *coins.Coin, rawTx string) (string, error, string) {
	if !s.TxsAvailable {
		return "not published due no-txs flag", nil, ""
	}
	blockbookWrapper := blockbook.NewBlockBookWrapper(coinConfig.Info.Blockbook)
	return blockbookWrapper.SendTxWithMessage(rawTx)
}

func (s *TycheControllerV2) AddShiftToMap(uid string, shiftPrepare models.PrepareShiftInfoV2) {
	s.mapLock.Lock()
	s.PrepareShifts[uid] = shiftPrepare
	s.mapLock.Unlock()
}

func (s *TycheControllerV2) GetShiftFromMap(key string) (models.PrepareShiftInfoV2, error) {
	s.mapLock.Lock()
	shift, ok := s.PrepareShifts[key]
	s.mapLock.Unlock()
	if !ok {
		return models.PrepareShiftInfoV2{}, errors.New("no shift found on cache")
	}
	return shift, nil
}



// Tyche v2 API. Most important change is the use of ShiftId instead of UID as Mempool Map Key.
func (s *TycheControllerV2) PrepareV2(_ string, payload []byte, _ models.Params) (interface{}, error) {
	var prepareData models.PrepareShiftRequest
	err := json.Unmarshal(payload, &prepareData)
	if err != nil {
		return nil, err
	}
	selectedCoin, err := GetServiceConfig(prepareData, s.Hestia, true)
	if err != nil {
		return nil, err
	}

	amountTo, payment, err := GetRatesV2(prepareData, selectedCoin, s.Obol, s.Adrestia)
	log.Println("amountTo after GetRatesV2", amountTo)
	if err != nil {
		return nil, err
	}

	prepareShift := models.PrepareShiftInfoV2{
		ID:         utils.RandomString(),
		FromCoin:   prepareData.FromCoin,
		Payment:    payment,
		ToCoin:     prepareData.ToCoin,
		ToAddress:  prepareData.ToAddress,
		ToAmount:   int64(amountTo.ToUnit(amount.AmountSats)),
		Timestamp:  time.Now().Unix(),
		Path: payment.Conversions,
	}
	fmt.Println(prepareShift)
	prepareResponse := models.PrepareShiftResponseV2{
		Payment:        payment,
		ReceivedAmount: int64(amountTo.ToUnit(amount.AmountSats)),
		ShiftId: prepareShift.ID,
	}
	fmt.Println(prepareResponse)

	s.AddShiftToMap(prepareShift.ID, prepareShift)
	return prepareResponse, nil
}

func (s *TycheControllerV2) StoreV2(uid string, payload []byte, _ models.Params) (interface{}, error) {
	var shiftPayment models.StoreShiftV2
	err := json.Unmarshal(payload, &shiftPayment)
	if err != nil {
		return nil, err
	}
	storedShift, err := s.GetShiftFromMap(shiftPayment.ShiftId)
	if err != nil {
		return nil, err
	}

	var feePayment hestia.Payment

	var inTrade []hestia.Trade
	for _, trade := range storedShift.Path.InwardOrder {
		newTrade := hestia.Trade{
			OrderId:        "",
			Amount:         0,
			ReceivedAmount: 0,
			FromCoin:       trade.FromCoin,
			ToCoin:         trade.ToCoin,
			Symbol:         trade.Trade.Book,
			Side:           trade.Trade.Type,
			CreatedTime:    0,
			FulfilledTime:  0,
		}
		inTrade = append(inTrade, newTrade)
	}
	if len(inTrade) > 0 {
		inTrade[0].Amount = amount.AmountType(storedShift.Payment.Amount).ToNormalUnit()
	}

	var outTrade []hestia.Trade
	for _, trade := range storedShift.Path.OutwardOrder {
		newTrade := hestia.Trade{
			OrderId:        "",
			Amount:         0,
			ReceivedAmount: 0,
			FromCoin:       trade.FromCoin,
			ToCoin:         trade.ToCoin,
			Symbol:         trade.Trade.Book,
			Side:           trade.Trade.Type,
			CreatedTime:    0,
			FulfilledTime:  0,
		}
		outTrade = append(outTrade, newTrade)
	}

	shift := hestia.ShiftV2{
		ID:        storedShift.ID,
		UID:       uid,
		Status:    hestia.ShiftStatusV2Created,
		Timestamp: time.Now().Unix(),
		Payment: hestia.Payment{
			Address:       storedShift.Payment.Address.ExchangeAddress.Address,
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
		InboundTrade: inTrade,
		OutboundTrade: outTrade,
	}

	s.RemoveShiftFromMap(shiftPayment.ShiftId)
	shiftId, err := s.Hestia.UpdateShiftV2(shift)
	if err != nil {
		return nil, err
	}
	go s.decodeAndCheckTx(shift, storedShift, shiftPayment.RawTX, shiftPayment.FeeTX)
	return shiftId, nil
}

func GetRatesV2(prepareData models.PrepareShiftRequest, selectedCoin hestia.Coin, obolService obol.ObolService, adrestiaService services.AdrestiaService) (amountTo amount.AmountType, paymentData models.PaymentInfoV2, err error){
	amountHandler := amount.AmountType(prepareData.Amount)
	// Get rates from coin to target coin. Determines input coin workable and fee amount, both come in the same transaction.
	inputAmount := amount.AmountType(amountHandler.ToUnit(amount.AmountSats) * (1.0 - selectedCoin.Shift.FeePercentage/100))
	fee := amount.AmountType(amountHandler.ToUnit(amount.AmountSats) * selectedCoin.Shift.FeePercentage / float64(100))

	// Retrieve conversion rates
	rate, err := obolService.GetCoin2CoinRatesWithAmount(prepareData.FromCoin, prepareData.ToCoin, inputAmount.String())
	if err != nil {
		err = cerrors.ErrorObtainingRates
		return
	}
	// Handler for exchange rate
	rateAmountHandler, err := amount.NewAmount(rate.AveragePrice)
	if err != nil {
		err = cerrors.ErrorObtainingRates
		return
	}

	// Retrieve Dollar conversion
	coinRates, err := obolService.GetCoinRates(prepareData.FromCoin)
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
	// Values for estimated target conversion
	fromCoinToUSD := inputAmount.ToNormalUnit() * coinRatesUSD
	feeToUsd := fee.ToNormalUnit() * coinRatesUSD

	amountTo, err = amount.NewAmount(inputAmount.ToNormalUnit() * rateAmountHandler.ToNormalUnit())
	if err != nil {
		err = cerrors.ErrorFillingPaymentInformation
		return
	}

	paymentAddress, err := adrestiaService.GetAddress(prepareData.FromCoin)
	if err != nil {
		err = cerrors.ErrorFillingPaymentInformation
		return
	}

	pathInfo, err := adrestiaService.GetPath(prepareData.FromCoin, prepareData.ToCoin)
	if err != nil {
		err = cerrors.ErrorFillingPaymentInformation
		return
	}

	feeFlag := true
	if selectedCoin.Shift.FeePercentage == 0 {
		feeFlag = false
	}
	paymentData = models.PaymentInfoV2{
		Address: paymentAddress,
		Amount:  int64(inputAmount.ToUnit(amount.AmountSats)), // Amount + Fee
		Fee: int64(fee.ToUnit(amount.AmountSats)),
		Total: int64(inputAmount.ToUnit(amount.AmountSats)) + int64(fee.ToUnit(amount.AmountSats)),
		HasFee:  feeFlag,
		Rate: int64(rateAmountHandler.ToUnit(amount.AmountSats)),
		FiatInfo: models.ExpectedFiatAmount{
			Amount: fromCoinToUSD,
			Fee:    feeToUsd,
		},
		Conversions: pathInfo,
	}
	return
}
// utils
func (s *TycheControllerV2) RemoveShiftFromMap(uid string) {
	s.mapLock.Lock()
	delete(s.PrepareShifts, uid)
	s.mapLock.Unlock()
}

func (s *TycheControllerV2) decodeAndCheckTx(shiftData hestia.ShiftV2, storedShiftData models.PrepareShiftInfoV2, rawTx string, feeTx string) {
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
			shiftData.Status = hestia.ShiftStatusV2Error
			_, err = s.Hestia.UpdateShiftV2(shiftData)
			if err != nil {
				return
			}
			return
		}

		if !valid {
			shiftData.Status = hestia.ShiftStatusV2Error
			_, err = s.Hestia.UpdateShiftV2(shiftData)
			if err != nil {
				return
			}
			return
		}

		// Broadcast fee rawTx
		polisCoinConfig, err := coinFactory.GetCoin("POLIS")
		if err != nil {
			// If get coin fail, we should mark error, no spent anything.
			shiftData.Status = hestia.ShiftStatusV2Error
			_, err = s.Hestia.UpdateShiftV2(shiftData)
			if err != nil {
				return
			}
			return
		}
		feeTxID, err, _ := s.broadCastTx(polisCoinConfig, feeTx)
		if err != nil {
			// If broadcast fail, we should mark error, no spent anything.
			shiftData.Status = hestia.ShiftStatusV2Error
			_, err = s.Hestia.UpdateShiftV2(shiftData)
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
		shiftData.Status = hestia.ShiftStatusV2Error
		if storedShiftData.FromCoin != "POLIS" && storedShiftData.ToCoin != "POLIS" {
			// If decode fail and payment is not POLIS, we should mark Refund to send back the fees.
			shiftData.Status = hestia.ShiftStatusV2Error
		}
		_, err = s.Hestia.UpdateShiftV2(shiftData)
		if err != nil {
			return
		}
		return
	}
	if !valid {
		if storedShiftData.FromCoin != "POLIS" && storedShiftData.ToCoin != "POLIS" {
			// If is not valid and payment is not POLIS, we should mark Refund to send back the fees.
			shiftData.Status = hestia.ShiftStatusV2Error
		}
	}
	// Broadcast rawTx
	coinConfig, err := coinFactory.GetCoin(shiftData.Payment.Coin)
	if err != nil {
		// If get coin fail and payment is POLIS, we should mark error.
		shiftData.Status = hestia.ShiftStatusV2Error
		if storedShiftData.FromCoin != "POLIS" {
			// If get coin fail and payment is not POLIS, we should mark Refund to send back the fees.
			shiftData.Status = hestia.ShiftStatusV2Refund
		}
		_, err = s.Hestia.UpdateShiftV2(shiftData)
		if err != nil {
			return
		}
		return
	}
	paymentTxid, err, message := s.broadCastTx(coinConfig, rawTx)
	if err != nil {
		// If broadcast fail and payment is POLIS, we should mark error.
		shiftData.Status = hestia.ShiftStatusV2Error
		if storedShiftData.FromCoin != "POLIS" {
			// If broadcast fail and payment is not POLIS, we should mark Refund to send back the fees.
			shiftData.Status = hestia.ShiftStatusV2Refund
		}
		_, err = s.Hestia.UpdateShiftV2(shiftData)
		if err != nil {
			return
		}
		return
	}
	shiftData.Message = message
	// Update shift model include txid.
	shiftData.Payment.Txid = paymentTxid
	_, err = s.Hestia.UpdateShiftV2(shiftData)
	if err != nil {
		return
	}
}