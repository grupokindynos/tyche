package controllers

import (
	"github.com/grupokindynos/tyche/services"
	"github.com/gin-gonic/gin"
)

type TycheController struct {
	ObolService *services.ObolService
	HestiaService    *services.HestiaService
	PlutusService  *services.PlutusService

}

// ValidateAddress checks that the sending address belongs to the hot-wallets
func (s *TycheController) ValidateAddress(c *gin.Context) {

}

// GetNewAddress fetches a new address from the hot-wallets
func (s *TycheController) GetNewAddress(c *gin.Context) {

}

// GetTycheStatus gets the current status of a given tyche ID
func (s *TycheController) GetTycheStatus(c *gin.Context) {

}

// GetTycheBalance calculates the amount of tyche that an individual can do
func (s *TycheController) GetTycheAmount(c *gin.Context) {

}

// StoreTyche validates and stores the  tyche on firebase
func (s *TycheController) StoreTyche(c *gin.Context) {

}