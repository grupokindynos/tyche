package main

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/grupokindynos/shift/controllers"
	"github.com/grupokindynos/shift/services"

	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()

}

func main() {

	// obolService := services.InitObolService()
	plutusService := services.InitPlutusService()
	plutusService.GetWalletAddress("polis")

}

func GetApp() *gin.Engine {
	App := gin.Default()
	App.Use(cors.Default())
	ApplyRoutes(App)
	return App

}

func ApplyRoutes(r *gin.Engine) {
	api := r.Group("/")
	{
		obolService := services.InitObolService()
		hestiaService := services.InitHestiaService()
		plutusService := services.InitPlutusService()

		shiftCtrl := controllers.ShiftController{ObolService: obolService, HestiaService: hestiaService, PlutusService: plutusService}

		api.GET("shift/address/validate/:coin/:address", shiftCtrl.ValidateAddress)
		api.GET("shift/address/new/:coin", shiftCtrl.GetNewAddress)
		api.GET("shift/status/:shiftID", shiftCtrl.GetShiftStatus)
		api.GET("shift/balance/:coin", shiftCtrl.GetShiftAmount)
		api.POST("shift/new", shiftCtrl.StoreShift)

	}
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})
}
