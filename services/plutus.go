package services

import (
	"encoding/json"
	"errors"
	"github.com/grupokindynos/common/plutus"
	"github.com/grupokindynos/common/tokens/mrt"
	"github.com/grupokindynos/common/tokens/mvt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func GetWalletBalance(coin string) (plutus.Balance, error) {
	req, err := mvt.CreateMVTToken("GET", plutus.ProductionURL+"/balance/"+coin, "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("PLUTUS_AUTH_USERNAME"), os.Getenv("PLUTUS_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return plutus.Balance{}, err
	}
	client := http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       time.Second * 30,
	}
	res, err := client.Do(req)
	if err != nil {
		return plutus.Balance{}, err
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return plutus.Balance{}, err
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return plutus.Balance{}, err
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		return plutus.Balance{}, errors.New("no header signature")
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("PLUTUS_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return plutus.Balance{}, err
	}
	var response plutus.Balance
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return plutus.Balance{}, err
	}
	return response, nil
}

func GetNewPaymentAddress(coin string) (addr string, err error) {
	req, err := mvt.CreateMVTToken("GET", plutus.ProductionURL+"/address/"+coin, "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("PLUTUS_AUTH_USERNAME"), os.Getenv("PLUTUS_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return addr, err
	}
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return addr, err
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return addr, err
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return addr, err
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		return addr, err
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("PLUTUS_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return addr, err
	}
	var response string
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return addr, err
	}
	return response, nil
}
