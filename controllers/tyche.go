package controllers

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/tyche/config"
	"github.com/grupokindynos/tyche/models/microservices"
	"github.com/grupokindynos/tyche/services"
)

//TycheController has the functions for handling the API endpoints
type TycheController struct {
	ObolService   *services.ObolService
	HestiaService *services.HestiaService
	PlutusService *services.PlutusService
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

// PrepareShift prepares a shift given the coins and amount, and returns a token and a timestamp
func (s *TycheController) PrepareShift(c *gin.Context) {

	fbToken := c.GetHeader("Token")

	// Verify firebase token
	hestia.VerifyToken(os.Getenv("SERVICE_NAME"), os.Getenv("MASTER_PASSWORD"), fbToken, os.Getenv("HESTIA_USERNAME"), os.Getenv("HESTIA_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"), os.Getenv("HESTIA_PUBLIC_KEY"))
	// Decrypt payload

	// Get Rate

	// Generate token

	// Add token to array and start go routine to exit

	return
}

// StoreShift validates and stores the shift on firebase
func (s *TycheController) StoreShift(c *gin.Context) {
	// Validations
	// 1. Make sure all values are filled.

	// 2. Get coin data, return false if a user payed or want a coin that doesn't exist

	// 3. Make sure payment address is ours and raw tx has the correct information
	// 3.1 Check address

	// 3.2 Deserialize Raw Tx with node

	// 3.3 Check if address and amount match

	// 4. Get rate

	return
}
