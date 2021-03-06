package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/grupokindynos/common/jwt"
	"github.com/grupokindynos/tyche/models"
	"github.com/grupokindynos/tyche/processor"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/obol"
	"github.com/grupokindynos/common/responses"
	"github.com/grupokindynos/common/tokens/ppat"
	"github.com/grupokindynos/tyche/controllers"
	"github.com/grupokindynos/tyche/services"

	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()
}

type CurrentTime struct {
	Hour   int
	Day    int
	Minute int
	Second int
}

var (
	prepareShiftsMap   = make(map[string]models.PrepareShiftInfo)
	prepareShiftsMapV2 = make(map[string]models.PrepareShiftInfoV2)
)

var (
	hestiaEnv       string
	adrestiaEnv     string
	plutusEnv       string
	noTxsAvailable  bool
	skipValidations bool
	devMode         bool
)

const prepareShiftTimeframe = 60 * 5 // 5 minutes

func main() {
	// Read input flag
	localRun := flag.Bool("local", false, "set this flag to run tyche with local requests")
	noTxs := flag.Bool("no-txs", false, "set this flag to avoid txs being executed"+
		"IMPORTANT: -local flag needs to be set in order to use this.")
	skipVal := flag.Bool("skip-val", false, "set this flag to avoid validations on txs."+
		"IMPORTANT: -local flag needs to be set in order to use this.")
	stopProcessor := flag.Bool("stop-proc", false, "set this flag to stop the automatic run of processor")
	port := flag.String("port", os.Getenv("PORT"), "set different port for local run")
	dev := flag.Bool("dev", false, "return shift status as always available")

	flag.Parse()

	// If flag was set, change the hestia request url to be local
	if *localRun {
		hestiaEnv = "HESTIA_LOCAL_URL"
		adrestiaEnv = "ADRESTIA_LOCAL_URL"
		plutusEnv = "PLUTUS_PRODUCTION_URL"

		// check if testing flags were set
		noTxsAvailable = *noTxs
		skipValidations = *skipVal

	} else {
		adrestiaEnv = "ADRESTIA_PRODUCTION_URL"
		hestiaEnv = "HESTIA_PRODUCTION_URL"
		plutusEnv = "PLUTUS_PRODUCTION_URL"

		if *noTxs || *skipVal {
			fmt.Println("cannot set testing flags without -local flag")
			os.Exit(1)
		}
	}

	devMode = *dev

	if !*stopProcessor {
		go runProcessorV2()
	}

	App := GetApp()
	_ = App.Run(":" + *port)
}

func GetApp() *gin.Engine {
	App := gin.Default()
	corsConf := cors.DefaultConfig()
	corsConf.AllowAllOrigins = true
	corsConf.AllowHeaders = []string{"token", "service", "content-type"}
	App.Use(cors.New(corsConf))
	ApplyRoutes(App)
	return App
}

func ApplyRoutes(r *gin.Engine) {
	/*tycheCtrl := &controllers.TycheController{
		PrepareShifts: prepareShiftsMap,
		TxsAvailable:  !noTxsAvailable,
		Hestia:        &services.HestiaRequests{HestiaURL: hestiaEnv},
		Plutus:        &services.PlutusRequests{PlutusUrl: os.Getenv(plutusEnv)},
		Obol:          &obol.ObolRequest{ObolURL: os.Getenv("OBOL_PRODUCTION_URL")},
		DevMode:       devMode,
	}*/

	// Service Instances
	tycheV2Ctrl := &controllers.TycheControllerV2{
		PrepareShifts: prepareShiftsMapV2,
		TxsAvailable:  !noTxsAvailable,
		Hestia:        &services.HestiaRequests{HestiaURL: hestiaEnv},
		Plutus:        &services.PlutusRequests{PlutusUrl: os.Getenv(plutusEnv)},
		Obol:          &obol.ObolRequest{ObolURL: os.Getenv("OBOL_PRODUCTION_URL")},
		Adrestia:      &services.AdrestiaRequests{AdrestiaUrl: adrestiaEnv},
		DevMode:       devMode,
	}

	// Backward compatibility
	go checkAndRemoveV2Shifts(tycheV2Ctrl)

	apiV2 := r.Group("/v2/")
	{
		apiV2.POST("prepare", func(context *gin.Context) { ValidateRequest(context, tycheV2Ctrl.PrepareV2) })
		apiV2.POST("new", func(context *gin.Context) { ValidateRequest(context, tycheV2Ctrl.StoreV2) })
		apiV2.GET("balance/:coin", func(context *gin.Context) { ValidateRequest(context, tycheV2Ctrl.BalanceV2) })
		apiV2.GET("status", func(context *gin.Context) { ValidateRequest(context, tycheV2Ctrl.StatusV2) })
	}
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})

	/*username := os.Getenv("OPEN_API_USER")
	password := os.Getenv("OPEN_API_PASSWORD")
	openApi := r.Group("/shift/open/", gin.BasicAuth(gin.Accounts{
		username: password,
	}))
	{
		openApi.GET("balance/:coin", func(context *gin.Context) { ValidateOpenRequest(context, tycheV2Ctrl.OpenBalance) })
		openApi.GET("status", func(context *gin.Context) { ValidateOpenRequest(context, tycheV2Ctrl.OpenStatus) })
		openApi.POST("prepare", func(context *gin.Context) { ValidateOpenRequest(context, tycheV2Ctrl.OpenPrepare) })
		openApi.POST("new", func(context *gin.Context) { ValidateOpenRequest(context, tycheV2Ctrl.OpenStore) })
	}*/
}

func ValidateRequest(c *gin.Context, method func(uid string, payload []byte, params models.Params) (interface{}, error)) {
	fbToken := c.GetHeader("token")
	fmt.Println("fbtoken: ", fbToken)
	if fbToken == "" {
		responses.GlobalResponseNoAuth(c)
		return
	}
	params := models.Params{
		Coin: c.Param("coin"),
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
	valid, payload, uid, err := ppat.VerifyPPATToken(os.Getenv(hestiaEnv), "tyche", os.Getenv("MASTER_PASSWORD"), fbToken, ReqBody.Payload, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"), os.Getenv("HESTIA_PUBLIC_KEY"))
	if !valid {
		responses.GlobalResponseNoAuth(c)
		return
	}
	response, err := method(uid, payload, params)
	if err != nil {
		responses.GlobalResponseError(nil, err, c)
		return
	}
	token, err := jwt.EncryptJWE(uid, response)
	responses.GlobalResponseError(token, err, c)
	return
}

func ValidateOpenRequest(c *gin.Context, method func(uid string, payload []byte, params models.Params) (interface{}, error)) {
	uid := c.MustGet(gin.AuthUserKey).(string)
	if uid == "" {
		responses.GlobalOpenNoAuth(c)
	}
	params := models.Params{
		Coin: c.Param("coin"),
	}
	payload, err := c.GetRawData()
	response, err := method(uid, payload, params)
	if err != nil {
		responses.GlobalOpenError(nil, err, c)
		return
	}
	responses.GlobalResponse(response, c)
	return
}

func runProcessorV2() {
	proc2 := processor.TycheProcessorV2{
		Hestia:          &services.HestiaRequests{HestiaURL: hestiaEnv},
		Plutus:          &services.PlutusRequests{PlutusUrl: os.Getenv(plutusEnv)},
		HestiaURL:       hestiaEnv,
		SkipValidations: skipValidations,
		Obol:            &obol.ObolRequest{ObolURL: os.Getenv("OBOL_PRODUCTION_URL")},
		Adrestia:        &services.AdrestiaRequests{AdrestiaUrl: adrestiaEnv},
	}

	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		proc2.Start()
	}
}

func checkAndRemoveV2Shifts(ctrl *controllers.TycheControllerV2) {
	for {
		time.Sleep(time.Second * 60)
		log.Print("Removing obsolete shifts request")
		count := 0
		for k, v := range ctrl.PrepareShifts {
			if time.Now().Unix() > v.Timestamp+prepareShiftTimeframe {
				count += 1
				ctrl.RemoveShiftFromMap(k)
			}
		}
		log.Printf("Removed %v shifts", count)
	}
}
