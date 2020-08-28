package controllers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"log"
	"strconv"
	"sync"
	"time"

	"github.com/eabz/btcutil"
	"github.com/eabz/btcutil/txscript"
	cerrors "github.com/grupokindynos/common/errors"
	"github.com/grupokindynos/tyche/services"
	"github.com/shopspring/decimal"

	"github.com/grupokindynos/common/blockbook"

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

func (s *TycheControllerV2) StatusV2(uid string, _ []byte, _ models.Params) (interface{}, error) {
	if s.DevMode {
		return false, nil
	}
	whitelistIds := os.Getenv("WHITELIST")
	whitelist := strings.Split(whitelistIds, ",")
	for _, id := range whitelist {
		if uid == id {
			return true, nil
		}
	}
	// if uid == "gwY3fy79LZMtUbSNBDoom7llGfh2" || uid == "oXuH5LwghkQG2JPYEYt1jJ08WU72" || uid == "dCtcq9M4JGMo5TraWv2GhkYclHR2" || uid == "WUNEUCLsoeRsXbtVOythROXqXk93" || uid == "m6hadvwAb4Z7IaOZAd1MDPSUVtk1" || uid == "QqDLwEfxKKZMFr2jMSwu1Mfh2I53" || uid == "aB0bQYzk5LhADlDGoeE80bEzSaw1" || uid == "HMOXcoZJxfMKFca9IukZIaqI2Z02" || uid == "Vcjnoyoog2RmzJpqk7Afef5W0ds1" || uid == "yz70K4OwehRgjVGSeUfN6AcM1yR2" || uid == "DAcEJJ00VnPQiThF0IoYW6su9LU2" || uid == "yEF8YP4Ou9aCEqSPQPqDslviGfT2"{

	/* if uid == "gwY3fy79LZMtUbSNBDoom7llGfh2" || uid == "HMOXcoZJxfMKFca9IukZIaqI2Z02" || uid == "yEF8YP4Ou9aCEqSPQPqDslviGfT2" || uid == "dCtcq9M4JGMo5TraWv2GhkYclHR2" || uid == "aB0bQYzk5LhADlDGoeE80bEzSaw1" || uid == "QqDLwEfxKKZMFr2jMSwu1Mfh2I53" || uid == "ZUucrGooOOXyGUEj6AGaH8epoBn2" {
		return true, nil
	}*/
	status, err := s.Hestia.GetShiftStatus()
	if err != nil {
		return nil, err
	}
	return status.Shift.Service, nil
}

func (s *TycheControllerV2) BalanceV2(_ string, _ []byte, params models.Params) (interface{}, error) {
	balance, err := s.Adrestia.StockBalance(params.Coin)
	if err != nil {
		return nil, err
	}
	rate, err := s.Obol.GetCoin2CoinRates(balance.Asset, params.Coin)
	if err != nil {
		return nil, err
	}
	fmt.Println(rate, balance.Balance)
	response := plutus.Balance{
		Confirmed:   balance.Balance * rate,
		Unconfirmed: 0,
	}
	return response, nil
}

func (s *TycheControllerV2) broadCastTx(coinConfig *coins.Coin, rawTx string) (string, error, string) {
	if !s.TxsAvailable {
		return "not published due no-txs flag", nil, ""
	}
	if coinConfig.Info.Token && coinConfig.Info.Tag != "ETH" {
		coinConfig, _ = coinFactory.GetCoin("ETH")
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
func (s *TycheControllerV2) PrepareV2(uid string, payload []byte, _ models.Params) (interface{}, error) {
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

	if payment.FiatInfo.Amount < 15.0 {
		return nil, cerrors.ErrorShiftMinimumAmount
	}

	timestamp := strconv.FormatInt(time.Now().Unix()-24*3600, 10)
	shifts, err := s.Hestia.GetShiftsByTimestampV2(uid, timestamp)
	if err != nil {
		return nil, err
	}

	totalAmountFiat := payment.FiatInfo.Amount + payment.FiatInfo.Fee

	for _, shift := range shifts {
		if shift.Status != hestia.ShiftStatusV2Error && shift.Status != hestia.ShiftStatusV2Refunded {
			fiatAmount := shift.InboundTrade.Conversions[0].Amount / shift.OriginalUsdRate
			totalAmountFiat += fiatAmount
		}
	}
	if totalAmountFiat > 200.0 {
		return nil, cerrors.ErrorShiftDailyLimit
	}

	prepareShift := models.PrepareShiftInfoV2{
		ID:               utils.RandomString(),
		FromCoin:         prepareData.FromCoin,
		Payment:          payment,
		ToCoin:           prepareData.ToCoin,
		ToAddress:        prepareData.ToAddress,
		ToAmount:         amountTo.IntPart(),
		Timestamp:        time.Now().Unix(),
		Path:             payment.Conversions,
		StableCoinAmount: payment.FiatInfo.Amount,
	}
	fmt.Println(prepareShift)
	prepareResponse := models.PrepareShiftResponseV2{
		Payment:        payment,
		ReceivedAmount: amountTo.IntPart(),
		ShiftId:        prepareShift.ID,
	}
	fmt.Println(prepareResponse)

	s.AddShiftToMap(prepareShift.ID, prepareShift)
	log.Println(s.PrepareShifts)
	return prepareResponse, nil
}

func (s *TycheControllerV2) StoreV2(uid string, payload []byte, _ models.Params) (interface{}, error) {
	var shiftPayment models.StoreShiftV2
	err := json.Unmarshal(payload, &shiftPayment)
	if err != nil {
		return nil, err
	}
	log.Println("Shift ID TO CREATE: ", shiftPayment.ShiftId)
	storedShift, err := s.GetShiftFromMap(shiftPayment.ShiftId)
	if err != nil {
		return nil, err
	}
	inExchange := ""
	outExchange := ""

	// Create Trade Objects for two-way orders
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
			Exchange:       trade.Exchange,
			CreatedTime:    0,
			FulfilledTime:  0,
		}
		inExchange = trade.Exchange
		inTrade = append(inTrade, newTrade)
	}
	if len(inTrade) > 0 {
		log.Println("Total satoshis ", storedShift.Payment.Total)
		log.Println("Total satoshis ", decimal.NewFromInt(storedShift.Payment.Total).DivRound(decimal.NewFromInt(1e8), 8).String())
		floatTrade, _ := decimal.NewFromInt(storedShift.Payment.Total).DivRound(decimal.NewFromInt(1e8), 8).Float64()
		inTrade[0].Amount = floatTrade
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
			Exchange:       trade.Exchange,
			CreatedTime:    0,
			FulfilledTime:  0,
		}
		outExchange = trade.Exchange
		outTrade = append(outTrade, newTrade)
	}
	withdrawAmount := 0.0 // only to be used when conversion path has no conversions (out coin is exchange's stable coin)
	if outTrade != nil && len(outTrade) > 0 {
		// Sets the initial trade output amount for the outbound trades
		outTrade[0].Amount = storedShift.StableCoinAmount
	} else {
		withdrawAmount = storedShift.StableCoinAmount
	}
	originUSDRateFloat, _ := decimal.NewFromInt(storedShift.Payment.Amount).DivRound(decimal.NewFromFloat(storedShift.StableCoinAmount), 3).Float64()

	shift := hestia.ShiftV2{
		ID:        storedShift.ID,
		UID:       uid,
		Status:    hestia.ShiftStatusV2Created,
		Timestamp: time.Now().Unix(),
		Payment: hestia.PaymentWithFee{
			Address:       storedShift.Payment.Address.ExchangeAddress.Address,
			Amount:        storedShift.Payment.Total,
			Fee:           storedShift.Payment.Fee,
			Usable:        storedShift.Payment.Amount,
			Coin:          storedShift.FromCoin,
			Txid:          "",
			Confirmations: 0,
		},
		ToCoin:         storedShift.ToCoin,
		ToAmount:       storedShift.ToAmount,
		ToAddress:      storedShift.ToAddress,
		RefundAddr:     shiftPayment.RefundAddr,
		PaymentProof:   "",
		ProofTimestamp: 0,
		InboundTrade: hestia.DirectionalTrade{
			Conversions:    inTrade,
			Status:         hestia.ShiftV2TradeStatusInitialized,
			Exchange:       inExchange,
			WithdrawAmount: 0.0,
		},
		OutboundTrade: hestia.DirectionalTrade{
			Conversions:    outTrade,
			Status:         hestia.ShiftV2TradeStatusCreated,
			Exchange:       outExchange,
			WithdrawAmount: withdrawAmount,
		},
		OriginalUsdRate: originUSDRateFloat,
	}

	shiftId, err := s.Hestia.UpdateShiftV2(shift)
	if err != nil {
		return nil, err
	}
	go s.decodeAndCheckTx(shift, storedShift, shiftPayment.RawTX)
	s.RemoveShiftFromMap(shiftPayment.ShiftId)
	return shiftId, nil
}

func GetRatesV2(prepareData models.PrepareShiftRequest, selectedCoin hestia.Coin, obolService obol.ObolService, adrestiaService services.AdrestiaService) (amountTo decimal.Decimal, paymentData models.PaymentInfoV2, err error) {
	amountHandler := decimal.NewFromInt(prepareData.Amount)
	// Get rates from coin to target coin. Determines input coin workable and fee amount, both come in the same transaction.
	inputAmount := amountHandler.Mul(decimal.NewFromFloat(1.0).Sub(decimal.NewFromFloat(selectedCoin.Shift.FeePercentage).DivRound(decimal.NewFromInt(100), 3)))
	fee := amountHandler.Mul(decimal.NewFromFloat(selectedCoin.Shift.FeePercentage).DivRound(decimal.NewFromInt(100), 3))
	//fee := amount.AmountType(amountHandler.ToUnit(amount.AmountSats) * selectedCoin.Shift.FeePercentage / float64(100))

	// Retrieve conversion rates
	rate, err := obolService.GetCoin2CoinRatesWithAmount(prepareData.FromCoin, prepareData.ToCoin, inputAmount.String())
	if err != nil {
		err = cerrors.ErrorObtainingRates
		return
	}
	// Handler for exchange rate
	rateAmountHandler := decimal.NewFromFloat(rate.AveragePrice)

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
	fromCoinToUSD := inputAmount.Mul(decimal.NewFromFloat(coinRatesUSD))
	feeToUsd := fee.Mul(decimal.NewFromFloat(coinRatesUSD))

	amountTo = inputAmount.Mul(rateAmountHandler)

	paymentAddress, err := adrestiaService.GetAddress(prepareData.FromCoin)
	if err != nil {
		log.Println("could not get sending address for coin " + prepareData.FromCoin)
		err = cerrors.ErrorFillingPaymentInformation
		return
	}

	pathInfo, err := adrestiaService.GetPath(prepareData.FromCoin, prepareData.ToCoin)
	if err != nil {
		err = cerrors.ErrorFillingPaymentInformation
		return
	}

	if !pathInfo.Trade {
		err = errors.New("shift pair not supported")
		return
	}

	feeFlag := true
	if selectedCoin.Shift.FeePercentage == 0 {
		feeFlag = false
	}

	fiatAmount, _ := fromCoinToUSD.Float64()
	feeUsdAmount, _ := feeToUsd.Float64()
	paymentData = models.PaymentInfoV2{
		Address: paymentAddress,
		Amount:  inputAmount.IntPart(), // Amount + Fee
		Fee:     fee.IntPart(),
		Total:   inputAmount.Add(fee).IntPart(),
		HasFee:  feeFlag,
		Rate:    rateAmountHandler.IntPart(),
		FiatInfo: models.ExpectedFiatAmount{
			Amount: fiatAmount,
			Fee:    feeUsdAmount,
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

func (s *TycheControllerV2) decodeAndCheckTx(shiftData hestia.ShiftV2, storedShiftData models.PrepareShiftInfoV2, rawTx string) {
	// Validate Payment RawTx
	body := plutus.ValidateRawTxReq{
		Coin:    shiftData.Payment.Coin,
		RawTx:   rawTx,
		Amount:  shiftData.Payment.Amount,
		Address: shiftData.Payment.Address,
	}
	valid, err := s.VerifyTxData(body)
	if err != nil {
		shiftData.Status = hestia.ShiftStatusV2Error
		shiftData.Message = "could not validate rawtx" + err.Error()
		_, err = s.Hestia.UpdateShiftV2(shiftData)
		if err != nil {
			return
		}
		return
	}
	if !valid {
		shiftData.Status = hestia.ShiftStatusV2Error
		shiftData.Message = "rawtx was invalid"
		_, err = s.Hestia.UpdateShiftV2(shiftData)
		if err != nil {
			return
		}
		return

	}
	// Broadcast rawTx
	coinConfig, err := coinFactory.GetCoin(shiftData.Payment.Coin)
	if err != nil {
		shiftData.Status = hestia.ShiftStatusV2Error
		shiftData.Message = "failed to acquire coin from coin factory"
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

func (s *TycheControllerV2) VerifyTxData(data plutus.ValidateRawTxReq) (bool, error) {
	coinConfig, err := coinFactory.GetCoin(data.Coin)
	if err != nil {
		return false, err
	}
	var isValue bool
	var isAddress bool
	if coinConfig.Info.Token || coinConfig.Info.Tag == "ETH" {
		return s.Plutus.ValidateRawTx(data)
	} else {
		//bitcoin-like coins
		value := btcutil.Amount(data.Amount)

		rawTxBytes, err := hex.DecodeString(data.RawTx)
		if err != nil {
			return false, err
		}
		tx, err := btcutil.NewTxFromBytes(rawTxBytes)
		if err != nil {
			return false, err
		}
		// Address Serialization
		currentAddress, err := btcutil.DecodeAddress(data.Address, coinConfig.NetParams)
		if err != nil {
			return false, err
		}
		scriptAddr, err := txscript.PayToAddrScript(currentAddress)
		if err != nil {
			return false, err
		}
		for _, out := range tx.MsgTx().TxOut {
			outAmount := btcutil.Amount(out.Value)
			if outAmount == value {
				isValue = true
			}
			if bytes.Equal(scriptAddr, out.PkScript) {
				isAddress = true
			}
		}
	}
	return isAddress && isValue, nil
}
