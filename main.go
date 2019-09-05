package main

import (
	"net/http"
	"os"

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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	App := GetApp()
	_ = App.Run(":" + port)
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
		shiftService := services.InitRateService()
		shiftCtrl := controllers.ShiftController{ShiftService: shiftService}
		api.GET("shift/status/:shiftID", shiftCtrl.GetShiftStatus)
		api.POST("shift/new", shiftCtrl.StoreShift)

	}
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})
}
