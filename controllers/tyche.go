package controllers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/grupokindynos/common/jwt"

	"github.com/gin-gonic/gin"
	coinfactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/obol"
	"github.com/grupokindynos/common/plutus"
	"github.com/grupokindynos/common/responses"
	"github.com/grupokindynos/tyche/config"
	tyche "github.com/grupokindynos/tyche/models"
	"github.com/grupokindynos/tyche/services"
)

//TycheController has the functions for handling the API endpoints
type TycheController struct {
	Cache map[string]hestia.ShiftRate
}

func (s *TycheController) GetServiceStatus(uid string, payload []byte) (interface{}, error) {
	status, err := services.GetServicesStatus()
	if err != nil {
		return nil, err
	}
	return jwt.EncryptJWE(uid, status.Shift)
}

//WaitRate is used for storing rates on the cache
func (s *TycheController) WaitRate(rate hestia.ShiftRate, hashString string) {
	// Store hash in cache
	s.Cache[hashString] = rate

	// Wait for N seconds, then delete from cache
	seconds, _ := strconv.Atoi(os.Getenv("PREPARE_SECONDS"))
	duration := time.Duration(seconds)

	time.Sleep(time.Duration(duration) * time.Second)
	delete(s.Cache, hashString)
}

// GetShiftAmount calculates the amount of balance that an individual can do
func (s *TycheController) GetShiftAmount(c *gin.Context) {
	coin := c.Param("coin")
	balance, err := plutus.GetWalletAddress(os.Getenv("PLUTUS_URL"), coin, os.Getenv("TYCHE_PRIV_KEY"), "tyche", os.Getenv("PLUTUS_AUTH_USERNAME"), os.Getenv("PLUTUS_AUTH_PASSWORD"), os.Getenv("PLUTUS_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))

	if err != nil {
		config.GlobalResponse(nil, err, c)
		return
	}
	balanceModel := tyche.Balance{balance}

	config.GlobalResponse(balanceModel, err, c)

	return
}

func verifyTransaction(transaction plutus.DecodedRawTX, data hestia.ShiftRate) error {
	var isAddressOnTx, isAmountCorrect = false, false
	for _, vout := range transaction.Vout {
		if vout.ScriptPubKey.Addresses[0] == data.ToAddress {
			isAddressOnTx = true
		}
		amountToSat := int64(math.Round(vout.Value * 1e8))
		totalAmount := data.ToAmount + data.FeeAmount
		if amountToSat == totalAmount {
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

// PrepareShift prepares a shift given the coins and amount, and returns a token and a timestamp`
func (s *TycheController) PrepareShift(uid string, payload []byte) (interface{}, error) {

	// Get Data from Payload
	var payloadStr string
	json.Unmarshal(payload, &payloadStr)

	var shiftData tyche.Receive
	json.Unmarshal([]byte(payloadStr), &shiftData)

	fromCoin := "POLIS"
	toCoin := "BTC"
	amount := 1e8 * 200
	feeCoin := "DASH"
	amountStr := fmt.Sprintf("%f", amount/1e8)

	// Verify coin is on coin factory
	_, err := coinfactory.GetCoin(fromCoin)
	if err != nil {
		return fromCoin, err
	}

	_, err = coinfactory.GetCoin(toCoin)
	if err != nil {
		return fromCoin, err
	}

	// Get rate
	rate, err := obol.GetCoin2CoinRatesWithAmmount(fromCoin, toCoin, amountStr)

	if err != nil {
		return rate, err
	}

	// Calculate receive amount
	receiveAmount := math.Round(float64(amount) / rate)

	// Calculate fee
	fee := float64(receiveAmount) * .01 / 1e8
	feeStr := fmt.Sprintf("%f", fee)
	var rateFee, finalFee float64

	if toCoin != feeCoin {
		rateFee, err = obol.GetCoin2CoinRatesWithAmmount(toCoin, feeCoin, feeStr)
		finalFee = (fee / rateFee) * 1e8
	} else if fromCoin != feeCoin {
		finalFee = fee * 1e8
	}

	if err != nil {
		return rateFee, err
	}

	//Get address from Plutus
	address, err := plutus.GetWalletAddress(os.Getenv("PLUTUS_URL"), fromCoin, os.Getenv("TYCHE_PRIV_KEY"), "tyche", os.Getenv("PLUTUS_AUTH_USERNAME"), os.Getenv("PLUTUS_AUTH_PASSWORD"), os.Getenv("PLUTUS_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	addressFee, err := plutus.GetWalletAddress(os.Getenv("PLUTUS_URL"), feeCoin, os.Getenv("TYCHE_PRIV_KEY"), "tyche", os.Getenv("PLUTUS_AUTH_USERNAME"), os.Getenv("PLUTUS_AUTH_PASSWORD"), os.Getenv("PLUTUS_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))

	//Create rate object
	rateObject := hestia.ShiftRate{Rate: rate, FromAmount: int64(amount), FromCoin: fromCoin, ToCoin: toCoin, FeeAmount: int64(finalFee), ToAddress: address, FeeAddress: addressFee, FeeCoin: feeCoin, ToAmount: int64(receiveAmount)}

	// Generate token hashing the uid
	h := sha256.New()
	h.Write([]byte(uid))
	hashString := base64.URLEncoding.EncodeToString(h.Sum(nil))
	seconds, _ := strconv.Atoi(os.Getenv("PREPARE_SECONDS"))

	// Create response object
	responseObject := tyche.Prepare{Token: hashString, Rate: rateObject, Timestamp: time.Now().Unix() + int64(seconds)}

	// Store token in cache
	go s.WaitRate(rateObject, hashString)

	// token, err := jwt.EncryptJWE(uid, responseObject)

	return responseObject, err
}

// StoreShift validates and stores the shift on firebase
func (s *TycheController) StoreShift(c *gin.Context) {
	rawTX := c.Query("raw_tx")
	feeTX := c.Query("fee_tx")
	token := c.Query("token")
	//payAddress := c.Query("pay_address")

	// Get data from cache
	data, valid := s.Cache[token]

	if valid != true {
		responses.GlobalResponseError("", errors.New("token not found"), c)
	}

	// Decode Raw transaction
	transaction, _ := plutus.DecodeRawTX(os.Getenv("PLUTUS_URL"), []byte(rawTX), data.FromCoin, os.Getenv("TYCHE_PRIV_KEY"), "tyche", os.Getenv("PLUTUS_AUTH_USERNAME"), os.Getenv("PLUTUS_AUTH_PASSWORD"), os.Getenv("PLUTUS_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	transactionFee, _ := plutus.DecodeRawTX(os.Getenv("PLUTUS_URL"), []byte(feeTX), data.FeeCoin, os.Getenv("TYCHE_PRIV_KEY"), "tyche", os.Getenv("PLUTUS_AUTH_USERNAME"), os.Getenv("PLUTUS_AUTH_PASSWORD"), os.Getenv("PLUTUS_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	// Verify amount and address from prepared shift are the same as raw transaction

	err := verifyTransaction(transaction, data)

	if err != nil {
		responses.GlobalResponseError("", err, c)
	}

	err = verifyTransaction(transactionFee, data)
	if err != nil {
		responses.GlobalResponseError("", err, c)
	}

	/*
		// Broadcast transaction
		coinInfo, _ := coinfactory.GetCoin(data.FromCoin)

		res, err := config.HTTPClient.Get("https://" + coinInfo.BlockchainInfo.ExternalSource + "/api/v2/sendtx/" + rawTX)
		if err != nil {
			responses.GlobalResponseError(err, errors.New("could not broadcast transaction"), c)
		}
	*/

	shiftPayment := hestia.Payment{
		Address:       data.ToAddress,
		Amount:        data.ToAmount,
		Coin:          data.FromCoin,
		RawTx:         rawTX,
		Txid:          transaction.Txid,
		Confirmations: 0,
	}

	rate := hestia.ShiftRate{
		Rate:       data.Rate,
		FromCoin:   data.FromCoin,
		ToCoin:     data.ToCoin,
		FromAmount: data.FromAmount,
		FeeAmount:  data.FeeAmount,
		ToAddress:  data.ToAddress,
	}
	shift := hestia.Shift{
		ID:         "TEST_SHIFT",
		Payment:    shiftPayment,
		UID:        "XYZ12345678910",
		FeePayment: shiftPayment,
		Rate:       rate,
	}

	_, err = services.UpdateShift(shift)

	if err != nil {
		responses.GlobalResponseError("", errors.New("could not store shift in database"), c)
	}

	responses.GlobalResponseError("Success", nil, c)
}
