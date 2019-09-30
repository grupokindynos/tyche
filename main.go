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

/*
func ValidateRequest(c *gin.Context, method func(payload []byte, uid string, voucherid string) (interface{}, error)) {
	fbToken := c.GetHeader("Token")

	if fbToken == "" {
		responses.GlobalResponseNoAuth(c)
		return
	}
	tokenBytes, _ := c.GetRawData()
	var tokenStr string
	if len(tokenBytes) > 0 {
		err := json.Unmarshal(tokenBytes, &tokenStr)
		responses.GlobalResponseError(nil, err, c)
		return
	}
	valid, payload, uid, err := ppat.VerifyPPATToken("ladon", os.Getenv("MASTER_PASSWORD"), fbToken, tokenStr, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("LADON_PRIVATE_KEY"), os.Getenv("HESTIA_PUBLIC_KEY"))
	if !valid {
		responses.GlobalResponseNoAuth(c)
		return
	}
	response, err := method(payload, uid, voucherid)
	responses.GlobalResponseError(response, err, c)
	return
}
*/
