package config

import (
	"errors"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	//ObolURL is the URL for the rates
	ObolURL = os.Getenv("OBOL_URL")

	//PlutusURL is the URL for the hot wallets API
	PlutusURL = os.Getenv("PLUTUS_URL")

	//HestiaURL is the URL for the connection to the database
	HestiaURL = os.Getenv("HESTIA_URL")

	//ErrorCoinNotAvailable gets called when a coin is not available in Shift
	ErrorCoinNotAvailable = errors.New("Coin not available")

	//ErrorRequestTimeout prompts when the connection with the services times out
	ErrorRequestTimeout = errors.New("Request timed out")

	//HTTPClient refers to the client that is used for the HTTP connections to microservices
	HTTPClient = &http.Client{
		Timeout: time.Second * 60,
	}
)

// GlobalResponse is used to wrap all the API responses under the same model.
// Automatically detect if there is an error and return status and code according
func GlobalResponse(result interface{}, err error, c *gin.Context) *gin.Context {
	if err != nil {
		c.JSON(500, gin.H{"message": "Error", "error": err.Error(), "status": -1})
		return c
	}
	// If is a float, truncate it to sats
	value, isfloat := result.(float64)
	if isfloat {
		value := math.Floor(value*1e8) / 1e8
		c.JSON(200, gin.H{"data": value, "status": 1})
		return c
	}
	c.JSON(200, gin.H{"data": result, "status": 1})
	return c
}

func init() {
	_ = godotenv.Load("../env")

}
