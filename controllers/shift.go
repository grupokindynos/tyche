package controllers

import (
	"github.com/grupokindynos/shift/services"
	"github.com/gin-gonic/gin"
)

type ShiftController struct {
	ObolService *services.ObolService
	HestiaService    *services.HestiaService
	PlutusService  *services.PlutusService

}

// ValidateAddress checks that the sending address belongs to the hot-wallets
func (s *ShiftController) ValidateAddress(c *gin.Context) {

}

// GetNewAddress fetches a new address from the hot-wallets
func (s *ShiftController) GetNewAddress(c *gin.Context) {

}

// GetShiftStatus gets the current status of a given shift ID
func (s *ShiftController) GetShiftStatus(c *gin.Context) {

}

// GetShiftBalance calculates the amount of shift that an individual can do
func (s *ShiftController) GetShiftAmount(c *gin.Context) {

}

// StoreShift validates and stores the  shift on firebase
func (s *ShiftController) StoreShift(c *gin.Context) {

}