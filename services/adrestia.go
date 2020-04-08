package services

import (
	"encoding/json"
	"errors"
	"github.com/grupokindynos/adrestia-go/models"
	"github.com/grupokindynos/common/tokens/mrt"
	"github.com/grupokindynos/common/tokens/mvt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	url := os.Getenv(a.AdrestiaUrl) + "path/"
	pathParams := models.PathParams{
		FromCoin:      toCoin,
		ToCoin:        fromCoin,
	}
	log.Println(url)
	req, err := mvt.CreateMVTToken("GET", url, "tyche", os.Getenv("MASTER_PASSWORD"), pathParams, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
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
	err = json.Unmarshal(payload, &path)
	if err != nil {
		return
	}
	return
}