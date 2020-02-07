package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
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
	currTime         CurrentTime
	prepareShiftsMap = make(map[string]models.PrepareShiftInfo)
)

var (
	hestiaEnv       string
	noTxsAvailable  bool
	skipValidations bool
	devMode			bool
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
	dev := flag.Bool("dev", false, "return status as always available")

	flag.Parse()

	// If flag was set, change the hestia request url to be local
	if *localRun {
		hestiaEnv = "HESTIA_LOCAL_URL"

		// check if testing flags were set
		noTxsAvailable = *noTxs
		skipValidations = *skipVal

	} else {
		hestiaEnv = "HESTIA_PRODUCTION_URL"
		if *noTxs || *skipVal {
			fmt.Println("cannot set testing flags without -local flag")
			os.Exit(1)
		}
	}

	devMode = *dev

	currTime = CurrentTime{
		Hour:   time.Now().Hour(),
		Day:    time.Now().Day(),
		Minute: time.Now().Minute(),
		Second: time.Now().Second(),
	}

	if !*stopProcessor {
		go timer()
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
	tycheCtrl := &controllers.TycheController{
		PrepareShifts: prepareShiftsMap,
		TxsAvailable:  !noTxsAvailable,
		Hestia:        &services.HestiaRequests{HestiaURL: hestiaEnv},
		Plutus:        &services.PlutusRequests{},
		Obol:          &obol.ObolRequest{ObolURL: os.Getenv("OBOL_PRODUCTION_URL")},
		DevMode:	   devMode,
	}

	go checkAndRemoveShifts(tycheCtrl)
	api := r.Group("/")
	{
		api.GET("balance/:coin", func(context *gin.Context) { ValidateRequest(context, tycheCtrl.Balance) })
		api.GET("status", func(context *gin.Context) { ValidateRequest(context, tycheCtrl.Status) })
		api.POST("prepare", func(context *gin.Context) { ValidateRequest(context, tycheCtrl.Prepare) })
		api.POST("new", func(context *gin.Context) { ValidateRequest(context, tycheCtrl.Store) })

	}
	r.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})
}

func ValidateRequest(c *gin.Context, method func(uid string, payload []byte, params models.Params) (interface{}, error)) {
	fbToken := c.GetHeader("token")
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

func timer() {
	for {
		time.Sleep(1 * time.Second)
		currTime = CurrentTime{
			Hour:   time.Now().Hour(),
			Day:    time.Now().Day(),
			Minute: time.Now().Minute(),
			Second: time.Now().Second(),
		}
		if currTime.Second == 0 {
			var wg sync.WaitGroup
			wg.Add(1)
			runCrons(&wg)
			wg.Wait()
		}
	}
}

func runCrons(mainWg *sync.WaitGroup) {
	defer func() {
		mainWg.Done()
	}()
	var wg sync.WaitGroup
	wg.Add(1)
	proc := processor.Processor{
		Hestia:          &services.HestiaRequests{HestiaURL: hestiaEnv},
		Plutus:          &services.PlutusRequests{},
		HestiaURL:       hestiaEnv,
		SkipValidations: skipValidations,
	}

	go runCronMinutes(1, proc.Start, &wg) // 1 minute
	wg.Wait()
}

func runCronMinutes(schedule int, function func(), wg *sync.WaitGroup) {
	go func() {
		defer func() {
			wg.Done()
		}()
		remainder := currTime.Minute % schedule
		if remainder == 0 {
			function()
		}
		return
	}()
}

func checkAndRemoveShifts(ctrl *controllers.TycheController) {
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
