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

	"github.com/gin-gonic/gin"
	coinfactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/obol"
	"github.com/grupokindynos/common/plutus"
	"github.com/grupokindynos/common/responses"
	"github.com/grupokindynos/tyche/config"
	"github.com/grupokindynos/tyche/models/microservices"
	"github.com/grupokindynos/tyche/services"
)

//TycheController has the functions for handling the API endpoints
type TycheController struct {
	ObolService *services.ObolService

	PlutusService *services.PlutusService
	Cache         map[string]microservices.TycheRate
}

//WaitRate is used for storing rates on the cache
func (s *TycheController) WaitRate(rate microservices.TycheRate, hashString string) {
	// Store hash in cache
	s.Cache[hashString] = rate

	// Wait for N seconds, then delete from cache
	seconds, _ := strconv.Atoi(os.Getenv("PREPARE_SECONDS"))
	duration := time.Duration(seconds)

	time.Sleep(time.Duration(duration) * time.Second)
	delete(s.Cache, hashString)
}

// GetNewAddress fetches a new address from the hot-wallets
func (s *TycheController) GetNewAddress(c *gin.Context) {
	coin := c.Param("coin")
	address, err := s.PlutusService.GetWalletAddress(coin)

	if err != nil {
		config.GlobalResponse(nil, err, c)
		return
	}

	config.GlobalResponse(address, err, c)

	return
}

// GetShiftAmount calculates the amount of balance that an individual can do
func (s *TycheController) GetShiftAmount(c *gin.Context) {
	coin := c.Param("coin")
	balance, err := s.PlutusService.GetWalletBalance(coin)

	if err != nil {
		config.GlobalResponse(nil, err, c)
		return
	}
	balanceModel := microservices.TycheBalance{balance}

	config.GlobalResponse(balanceModel, err, c)

	return
}

// GetRateStatus calculates the amount of balance that an individual can do
func (s *TycheController) GetRateStatus(c *gin.Context) {
	coin := c.Param("coin")
	balance, err := s.PlutusService.GetWalletBalance(coin)

	if err != nil {
		config.GlobalResponse(nil, err, c)
		return
	}
	balanceModel := microservices.TycheBalance{balance}

	config.GlobalResponse(balanceModel, err, c)

	return
}

// PrepareShift prepares a shift given the coins and amount, and returns a token and a timestamp`
func (s *TycheController) PrepareShift(uid string, payload []byte) (interface{}, error) {

	// Get Data from Payload
	var payloadStr string
	json.Unmarshal(payload, &payloadStr)

	var shiftData microservices.TycheReceive
	json.Unmarshal([]byte(payloadStr), &shiftData)

	fromCoin := "BTC"
	toCoin := "POLIS"
	amount := 7058
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

	fee := float64(amount) * .01

	//Get address from Plutus
	address, err := plutus.GetWalletAddress(os.Getenv("PLUTUS_URL"), fromCoin, os.Getenv("TYCHE_PRIV_KEY"), "tyche", os.Getenv("PLUTUS_AUTH_USERNAME"), os.Getenv("PLUTUS_AUTH_PASSWORD"), os.Getenv("PLUTUS_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	address = "38yE9BaCgUpCawt6bWsDLcvoqcmHepdWoq"
	//Create rate object
	rateObject := microservices.TycheRate{Rate: rate, Amount: int64(amount), FromCoin: fromCoin, ToCoin: toCoin, Fee: int64(fee), Address: address}

	// Generate token hashing the uid
	h := sha256.New()
	h.Write([]byte(uid))
	hashString := base64.URLEncoding.EncodeToString(h.Sum(nil))
	seconds, _ := strconv.Atoi(os.Getenv("PREPARE_SECONDS"))

	// Create response object
	responseObject := microservices.TychePrepare{Token: hashString, Rate: rateObject, Timestamp: time.Now().Unix() + int64(seconds)}

	// Store token in cache
	go s.WaitRate(rateObject, hashString)

	// token, err := jwt.EncryptJWE(uid, responseObject)

	return responseObject, err
}

// StoreShift validates and stores the shift on firebase
func (s *TycheController) StoreShift(c *gin.Context) {
	rawTX := c.Query("raw_tx")
	token := c.Query("token")
	//payAddress := c.Query("pay_address")

	// Get data from cache
	data, valid := s.Cache[token]

	if valid != true {
		responses.GlobalResponseError("", errors.New("token not found"), c)
	}

	// Decode Raw transaction
	transaction, _ := plutus.DecodeRawTX(os.Getenv("PLUTUS_URL"), []byte(rawTX), data.FromCoin, os.Getenv("TYCHE_PRIV_KEY"), "tyche", os.Getenv("PLUTUS_AUTH_USERNAME"), os.Getenv("PLUTUS_AUTH_PASSWORD"), os.Getenv("PLUTUS_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))

	// Verify amount and address from prepared shift are the same as raw transaction
	var isAddressOnTx, isAmountCorrect = false, false
	for _, vout := range transaction.Vout {
		if vout.ScriptPubKey.Addresses[0] == data.Address {
			isAddressOnTx = true
		}
		amountToSat := int64(math.Round(vout.Value * 1e8))
		totalAmount := data.Amount + data.Fee
		if amountToSat == totalAmount {
			isAmountCorrect = true
		}
	}

	if isAddressOnTx == false {
		responses.GlobalResponseError("", errors.New("no matching address in raw tx"), c)
		return
	}

	if isAmountCorrect == false {
		responses.GlobalResponseError("", errors.New("incorrect amount in raw tx"), c)
		return
	}

	// Broadcast transaction

	/*


		// 3. Make sure payment address is ours and raw tx has the correct information
		// 3.1 Check address
		valid, err := s.RPCService.ValidateAddress(paymentCoinData, Shift.PaymentAddress)
		if !valid || err != nil {
			config.CaronteResponse(nil, config.ErrorShiftValidatingAddressPayed, c)
			return
		}

		// 3.2 Deserialize Raw Tx with node
		transaction, err := s.RPCService.DecodeRawTransaction(paymentCoinData, Shift.PaymentRawTx)
		if err != nil {
			config.CaronteResponse(nil, config.ErrorShiftDecodeRawTransaction, c)
			return
		}
		// 3.3 Check if address and amount match
		var isAddressOnTx, isAmountCorrect = false, false
		for _, vout := range transaction.Vout {
			if vout.ScriptPubKey.Addresses[0] == Shift.PaymentAddress {
				isAddressOnTx = true
			}
			amountToSat := vout.Value * 1e8
			amountToString := fmt.Sprintf("%f", amountToSat)
			if amountToString == Shift.PaymentAmount {
				isAmountCorrect = true
			}
		}

		if !isAddressOnTx || isAmountCorrect {
			config.CaronteResponse(nil, config.ErrorShiftAmountOrAddressIncorrect, c)
			return
		}

		// 4. Get rate
		rate, err := s.RateService.GetCoinToCoinRates(paymentCoinData, toCoinData)
		if err != nil {
			config.CaronteResponse(nil, config.ErrorShiftCoinsRates, c)
			return
		}

		paymentAmountToFloat, err := strconv.ParseFloat(Shift.PaymentAmount, 64)
		if err != nil {
			config.CaronteResponse(nil, config.ErrorUnableToParseStringToFloat, c)
			return
		}

		// 5. Broadcast the transaction
		TxID, err := s.RPCService.BroadcastTransaction(paymentCoinData, Shift.PaymentRawTx)
		if err != nil {
			config.CaronteResponse(nil, config.ErrorShiftFailedToBroadcast, c)
			return
		}

		// 6. Submit Shift Element
		shift := models.Shift{
			Confirmations:  0,
			Rate:           rate,
			ID:             s.GenNewID(),
			Status:         "PENDING",
			Time:           time.Now().Unix(),
			PaymentAddress: Shift.PaymentAddress,
			PaymentCoin:    Shift.PaymentCoin,
			PaymentAmount:  Shift.PaymentAmount,
			PaymentTxID:    TxID,
			ToAddress:      Shift.ToAddress,
			ToCoin:         Shift.ToCoin,
			ToAmount:       math.Floor((paymentAmountToFloat/rate)*1e8) / 1e8,
		}

		err = s.Firebase.StoreShift(shift)
		if err != nil {
			// If this fails, we need to notify the dev team somehow.
			// TODO
			config.CaronteResponse(nil, config.ErrorShiftFailedToStore, c)
			return
		}

		config.CaronteResponse(shift.ID, nil, c)
	*/
	return
}
