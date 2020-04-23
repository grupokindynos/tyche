package services

import (
	"encoding/json"
	"errors"
	"github.com/grupokindynos/adrestia-go/models"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/tokens/mrt"
	"github.com/grupokindynos/common/tokens/mvt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type AdrestiaRequests struct {
	AdrestiaUrl string
}

func (a *AdrestiaRequests) GetAddress(coin string) (address models.AddressResponse, err error) {
	url := os.Getenv(a.AdrestiaUrl) + "address/" + coin
	log.Println(url)
	req, err := mvt.CreateMVTToken("GET", url, "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		err = errors.New("no header signature")
		return
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("ADRESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return
	}
	err = json.Unmarshal(payload, &address)
	if err != nil {
		return
	}
	return
}

func (a *AdrestiaRequests) GetPath(fromCoin string, toCoin string) (path models.PathResponse, err error) {
	url := os.Getenv(a.AdrestiaUrl) + "path"
	pathParams := models.PathParams{
		FromCoin:      fromCoin,
		ToCoin:        toCoin,
	}
	req, err := mvt.CreateMVTToken("POST", url, "tyche", os.Getenv("MASTER_PASSWORD"), pathParams, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	log.Println(string(tokenResponse))
	if err != nil {
		return
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		err = errors.New("no header signature")
		return
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("ADRESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return
	}
	err = json.Unmarshal(payload, &path)
	if err != nil {
		return
	}
	return
}

func (a *AdrestiaRequests) Withdraw(withdrawParams models.WithdrawParams) (withdrawal models.WithdrawInfo, err error) {
	url := os.Getenv(a.AdrestiaUrl) + "withdraw"
	req, err := mvt.CreateMVTToken("POST", url, "tyche", os.Getenv("MASTER_PASSWORD"), withdrawParams, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	log.Println(string(tokenResponse))
	if err != nil {
		return
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		err = errors.New("no header signature")
		return
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("ADRESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return
	}
	err = json.Unmarshal(payload, &withdrawal)
	if err != nil {
		return
	}
	return
}

func (a *AdrestiaRequests) Trade(tradeParams hestia.Trade) (txId string, err error) {
	url := os.Getenv(a.AdrestiaUrl) + "trade"
	req, err := mvt.CreateMVTToken("POST", url, "tyche", os.Getenv("MASTER_PASSWORD"), tradeParams, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	log.Println(string(tokenResponse))
	if err != nil {
		return
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		err = errors.New("no header signature")
		return
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("ADRESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return
	}
	txId = string(payload)
	txId = strings.ReplaceAll(txId, "\"", "")
	return
}

func (a *AdrestiaRequests) StockBalance(asset string) (balance models.BalanceResponse, err error) {
	url := os.Getenv(a.AdrestiaUrl) + "stock/balance/" + asset
	req, err := mvt.CreateMVTToken("GET", url, "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	log.Println(string(tokenResponse))
	if err != nil {
		return
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		err = errors.New("no header signature")
		return
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("ADRESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return
	}
	err = json.Unmarshal(payload, &balance)
	if err != nil {
		return
	}
	return
}

func (a *AdrestiaRequests) DepositInfo(depositParams models.DepositParams) (depositInfo models.DepositInfo, err error) {
	url := os.Getenv(a.AdrestiaUrl) + "deposit"
	req, err := mvt.CreateMVTToken("POST", url, "tyche", os.Getenv("MASTER_PASSWORD"), depositParams, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	log.Println(string(tokenResponse))
	if err != nil {
		return
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		err = errors.New("no header signature")
		return
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("ADRESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return
	}
	err = json.Unmarshal(payload, &depositInfo)
	if err != nil {
		return
	}
	return
}

func (a *AdrestiaRequests) GetTradeStatus (tradeParams hestia.Trade) (tradeInfo hestia.ExchangeOrderInfo, err error) {
	url := os.Getenv(a.AdrestiaUrl) + "trade/status"
	req, err := mvt.CreateMVTToken("POST", url, "tyche", os.Getenv("MASTER_PASSWORD"), tradeParams, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	log.Println(string(tokenResponse))
	if err != nil {
		return
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		err = errors.New("no header signature")
		return
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("ADRESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return
	}
	err = json.Unmarshal(payload, &tradeInfo)
	if err != nil {
		return
	}
	return
}

func (a *AdrestiaRequests) GetWithdrawalTxHash (withdrawParams models.WithdrawInfo) (txId string, err error) {
	url := os.Getenv(a.AdrestiaUrl) + "withdraw/hash"
	req, err := mvt.CreateMVTToken("POST", url, "tyche", os.Getenv("MASTER_PASSWORD"), withdrawParams, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	log.Println(string(tokenResponse))
	if err != nil {
		return
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		err = errors.New("no header signature")
		return
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("ADRESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return
	}
	txId = string(payload)
	txId = strings.ReplaceAll(txId, "\"", "")
	return
}