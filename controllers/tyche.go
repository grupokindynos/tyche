package controllers

import (
	"os"
	"strconv"
	"time"

	"crypto/sha256"
	"encoding/base64"

	"github.com/gin-gonic/gin"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/obol"

	"github.com/grupokindynos/tyche/config"
	"github.com/grupokindynos/tyche/models/microservices"
	"github.com/grupokindynos/tyche/services"
)

//TycheController has the functions for handling the API endpoints
type TycheController struct {
	ObolService   *services.ObolService
	HestiaService *services.HestiaService
	PlutusService *services.PlutusService
	Cache         map[string]microservices.TycheRate
}

//WaitRate is used for storing rates on the cache
func (s *TycheController) WaitRate(rate microservices.TycheRate, hashString string) {
	// Store hash in cache
	s.Cache[hashString] = rate

	// Wait for N seconds, then delete from cache
	duration, _ := strconv.Atoi(os.Getenv("PREPARE_SECONDS"))
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

// PrepareShift prepares a shift given the coins and amount, and returns a token and a timestamp
func (s *TycheController) PrepareShift(c *gin.Context) {

	fbToken := c.GetHeader("Token")

	// Verify firebase token
	hestia.VerifyToken(os.Getenv("SERVICE_NAME"), os.Getenv("MASTER_PASSWORD"), fbToken, os.Getenv("HESTIA_USERNAME"), os.Getenv("HESTIA_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"), os.Getenv("HESTIA_PUBLIC_KEY"))
	// Decrypt payload

	// Get Rate
	fromCoin := c.Param("fromcoin")
	toCoin := c.Param("tocoin")
	amount := c.Query("amount")
	uid := c.Query("uid")

	rate, err := obol.GetCoin2CoinRatesWithAmmount(fromCoin, toCoin, amount)

	if err != nil {
		return
	}

	rateObject := microservices.TycheRate{Rate: rate, Timestamp: time.Now().Unix(), Amount: amount, FromCoin: fromCoin, ToCoin: toCoin}

	// Generate token hashing the uid
	h := sha256.New()
	h.Write([]byte(uid))
	hashString := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Create response object
	responseObject := microservices.TychePrepare{Token: hashString, Rate: rateObject}

	// Store token in cache
	go s.WaitRate(rateObject, hashString)

	config.GlobalResponse(responseObject, err, c)

	return
}

// StoreShift validates and stores the shift on firebase
func (s *TycheController) StoreShift(c *gin.Context) {
	// Validations
	// 2. Make sure all values are filled.

	// 3. Get coin data, return false if a user payed or want a coin that doesn't exist

	// 4. Make sure payment address is ours and raw tx has the correct information
	// 4.1 Check address

	// 4.2 Deserialize Raw Tx with node

	// 4.3 Check if address and amount match

	return
}
