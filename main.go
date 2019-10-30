package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/responses"
	"github.com/grupokindynos/common/tokens/ppat"
	"github.com/grupokindynos/tyche/controllers"

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
	corsConf := cors.DefaultConfig()
	corsConf.AllowAllOrigins = true
	corsConf.AllowHeaders = []string{"token", "service", "content-type"}
	App.Use(cors.New(corsConf))
	ApplyRoutes(App)
	return App
}

//ApplyRoutes applies the API routes to their controllers
func ApplyRoutes(r *gin.Engine) {
	api := r.Group("/")
	{

		var cache = map[string]hestia.ShiftRate{}

		tycheCtrl := controllers.TycheController{Cache: cache}

		api.GET("status", func(context *gin.Context) { ValidateRequest(context, tycheCtrl.GetServiceStatus) })
		api.GET("balance/:coin", tycheCtrl.GetShiftAmount)
		api.POST("prepare", func(context *gin.Context) { ValidateRequest(context, tycheCtrl.PrepareShift) })
		api.POST("new", func(context *gin.Context) { ValidateRequest(context, tycheCtrl.StoreShift) })

	}
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})
}

//ValidateRequest validates that the token sent from the frontend is valid
func ValidateRequest(c *gin.Context, method func(uid string, payload []byte) (interface{}, error)) {
	fbToken := c.GetHeader("token")
	if fbToken == "" {
		responses.GlobalResponseNoAuth(c)
		return
	}
	tokenBytes, _ := c.GetRawData()
	var ReqBody hestia.BodyReq
	if len(tokenBytes) > 0 {
		err := json.Unmarshal(tokenBytes, &ReqBody)
		if err != nil {
			responses.GlobalResponseError(nil, err, c)
			return
		}
	}
	valid, payload, uid, err := ppat.VerifyPPATToken(os.Getenv("HESTIA_URL"), "tyche", os.Getenv("MASTER_PASSWORD"), fbToken, ReqBody.Payload, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"), os.Getenv("HESTIA_PUBLIC_KEY"))
	if !valid {
		responses.GlobalResponseNoAuth(c)
		return
	}
	response, err := method(uid, payload)
	responses.GlobalResponseError(response, err, c)
	return
}
