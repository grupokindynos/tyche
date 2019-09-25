package main

import (
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/grupokindynos/tyche/controllers"
	"github.com/grupokindynos/tyche/models/microservices"
	"github.com/grupokindynos/tyche/services"

	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()

}

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	App := GetApp()
	_ = App.Run(":" + port)

}

//GetApp initializes gin API library
func GetApp() *gin.Engine {
	App := gin.Default()
	App.Use(cors.Default())
	ApplyRoutes(App)
	return App

}

//ApplyRoutes applies the API routes to their controllers
func ApplyRoutes(r *gin.Engine) {
	api := r.Group("/")
	{
		obolService := services.InitObolService()
		hestiaService := services.InitHestiaService()
		plutusService := services.InitPlutusService()
		var cache = map[string]microservices.TycheRate{}

		tycheCtrl := controllers.TycheController{ObolService: obolService, HestiaService: hestiaService, PlutusService: plutusService, Cache: cache}

		api.GET("tyche/address/new/:coin", tycheCtrl.GetNewAddress)
		api.GET("tyche/balance/:coin", tycheCtrl.GetShiftAmount)
		api.POST("tyche/shift/prepare/:fromcoin/:tocoin", tycheCtrl.PrepareShift)
		api.POST("tyche/shift/new", tycheCtrl.StoreShift)

	}
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})
}
