package controllers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eabz/btcutil"
	"github.com/eabz/btcutil/txscript"
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

func (s *TycheControllerV2) StatusV2(uid string, _ []byte, _ models.Params) (interface{}, error) {
	if s.DevMode {
		return true, nil
	}
	// if uid == "gwY3fy79LZMtUbSNBDoom7llGfh2" || uid == "oXuH5LwghkQG2JPYEYt1jJ08WU72" || uid == "dCtcq9M4JGMo5TraWv2GhkYclHR2" || uid == "WUNEUCLsoeRsXbtVOythROXqXk93" || uid == "m6hadvwAb4Z7IaOZAd1MDPSUVtk1" || uid == "QqDLwEfxKKZMFr2jMSwu1Mfh2I53" || uid == "aB0bQYzk5LhADlDGoeE80bEzSaw1" || uid == "HMOXcoZJxfMKFca9IukZIaqI2Z02" || uid == "Vcjnoyoog2RmzJpqk7Afef5W0ds1" || uid == "yz70K4OwehRgjVGSeUfN6AcM1yR2"{
	if uid == "gwY3fy79LZMtUbSNBDoom7llGfh2" || uid == "HMOXcoZJxfMKFca9IukZIaqI2Z02" || uid == "yEF8YP4Ou9aCEqSPQPqDslviGfT2" || uid == "dCtcq9M4JGMo5TraWv2GhkYclHR2" || uid == "aB0bQYzk5LhADlDGoeE80bEzSaw1" || uid == "berrueta.enrique@gmail.com" {
		return true, nil
	}
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

	if payment.FiatInfo.Amount < 18.0 {
		return nil, cerrors.ErrorShiftMinimumAmount
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
		StableCoinAmount: payment.FiatInfo.Amount,
	}
	fmt.Println(prepareShift)
	prepareResponse := models.PrepareShiftResponseV2{
		Payment:        payment,
		ReceivedAmount: int64(amountTo.ToUnit(amount.AmountSats)),
		ShiftId: prepareShift.ID,
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
			Exchange: 		trade.Exchange,
			CreatedTime:    0,
			FulfilledTime:  0,
		}
		inExchange = trade.Exchange
		inTrade = append(inTrade, newTrade)
	}
	if len(inTrade) > 0 {
		log.Println("Total satoshis ", storedShift.Payment.Total)
		log.Println("Total satoshis ", amount.AmountType(storedShift.Payment.Total).ToNormalUnit())
		inTrade[0].Amount = amount.AmountType(storedShift.Payment.Total).ToNormalUnit()
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
			Exchange: 		trade.Exchange,
			CreatedTime:    0,
			FulfilledTime:  0,
		}
		outExchange = trade.Exchange
		outTrade = append(outTrade, newTrade)
	}

	if len(outTrade) > 0 {
		// Sets the initial trade output amount for the ooutbound trades
		//outTrade[0].Amount = amount.AmountType(storedShift.Payment.Amount).ToNormalUnit()
		log.Println("Amount Check: ", storedShift.StableCoinAmount, "or precalculated ", storedShift.ToAmountUSD)
		outTrade[0].Amount = storedShift.StableCoinAmount
	}

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
			Conversions: inTrade,
			Status:      hestia.ShiftV2TradeStatusInitialized,
			Exchange: inExchange,
		},
		OutboundTrade: hestia.DirectionalTrade{
			Conversions: outTrade,
			Status:      hestia.ShiftV2TradeStatusCreated,
			Exchange: outExchange,
		},
		OriginalUsdRate: amount.AmountType(storedShift.Payment.Amount).ToNormalUnit() / storedShift.StableCoinAmount,
	}


	shiftId, err := s.Hestia.UpdateShiftV2(shift)
	if err != nil {
		return nil, err
	}
	go s.decodeAndCheckTx(shift, storedShift, shiftPayment.RawTX)
	s.RemoveShiftFromMap(shiftPayment.ShiftId)
	return shiftId, nil
}

func GetRatesV2(prepareData models.PrepareShiftRequest, selectedCoin hestia.Coin, obolService obol.ObolService, adrestiaService services.AdrestiaService) (amountTo amount.AmountType, paymentData models.PaymentInfoV2, err error){
	amountHandler := amount.AmountType(prepareData.Amount)
	// Get rates from coin to target coin. Determines input coin workable and fee amount, both come in the same transaction.
	inputAmount := amount.AmountType(amountHandler.ToUnit(amount.AmountSats) * (1.0 - selectedCoin.Shift.FeePercentage / 100.0))
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

	if !pathInfo.Trade {
		err = errors.New("shift pair not supported")
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

func (s *TycheControllerV2) decodeAndCheckTx(shiftData hestia.ShiftV2, storedShiftData models.PrepareShiftInfoV2, rawTx string) {
	// Validate Payment RawTx
	body := plutus.ValidateRawTxReq{
		Coin:    shiftData.Payment.Coin,
		RawTx:   rawTx,
		Amount:  shiftData.Payment.Amount,
		Address: shiftData.Payment.Address,
	}
	valid, err := VerifyTxData(body)
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

func VerifyTxData(data plutus.ValidateRawTxReq) (bool, error) {
	coinConfig, err := coinFactory.GetCoin(data.Coin)
	if err != nil {
		return false, err
	}
	var isValue bool
	var isAddress bool
	if coinConfig.Info.Token || coinConfig.Info.Tag == "ETH" {
		/*value := ValidateTxData.Amount
		var tx *types.Transaction
		rawtx, err := hex.DecodeString(ValidateTxData.RawTx)
		if err != nil {
			return nil, err
		}
		err = rlp.DecodeBytes(rawtx, &tx)
		if err != nil {
			return nil, err
		}
		//compare amount from the tx and the input body
		var txBodyAmount int64
		var txAddr common.Address
		if coinConfig.Info.Token {
			address, amount := DecodeERC20Data([]byte(hex.EncodeToString(tx.Data())))
			txAddr = common.HexToAddress(string(address))
			txBodyAmount = amount.Int64()
		} else {
			txBodyAmount = tx.Value().Int64()
			txAddr = *tx.To()
		}
		if txBodyAmount == value {
			isValue = true
		}
		bodyAddr := common.HexToAddress(ValidateTxData.Address)
		//compare the address from the tx and the input body
		if bytes.Equal(bodyAddr.Bytes(), txAddr.Bytes()) {
			isAddress = true
		}
	*/
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