package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/grupokindynos/tyche/config"
	"github.com/grupokindynos/tyche/models/microservices"
	"github.com/grupokindynos/tyche/services"
)

type TycheController struct {
	ObolService   *services.ObolService
	HestiaService *services.HestiaService
	PlutusService *services.PlutusService
}

// ValidateAddress checks that the sending address belongs to the hot-wallets
func (s *TycheController) ValidateAddress(c *gin.Context) {

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

// GetShiftStatus gets the current status of a given tyche ID
func (s *TycheController) GetShiftStatus(c *gin.Context) {

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

// StoreShift validates and stores the shift on firebase
func (s *TycheController) StoreShift(c *gin.Context) {

}
