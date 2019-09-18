package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/grupokindynos/tyche/config"
	"github.com/grupokindynos/tyche/models/microservices"
)

//ObolService is the connection to the Obol rate microservice
type ObolService struct {
	ObolURL string
}

//GetRatesSimple gets the rate from a given coin
func (o *ObolService) GetRatesSimple(coin string) (rates []map[string]interface{}, err error) {
	requestURL := o.ObolURL + "/simple/" + coin

	contents, err := o.GetObolData(requestURL)

	var Obol microservices.ObolSimple
	err = json.Unmarshal(contents, &Obol)
	rates = Obol.Data

	return rates, err

}

//GetRatesComplex gets the rate from a given coin, given the amount of coins it wants to change
func (o *ObolService) GetRatesComplex(fromcoin string, tocoin string) (rates float64, err error) {
	requestURL := o.ObolURL + "/complex/" + fromcoin + "/" + tocoin

	contents, err := o.GetObolData(requestURL)

	var Obol microservices.ObolComplex
	err = json.Unmarshal(contents, &Obol)
	rates = Obol.Data

	return rates, err

}

//GetRatesAmount gets the rate from a given coin, given the amount of coins it wants to change
func (o *ObolService) GetRatesAmount(fromcoin string, tocoin string, amount int) (rates float64, err error) {
	requestURL := o.ObolURL + "/complex/" + fromcoin + "/" + tocoin + "?amount=" + strconv.Itoa(amount)

	fmt.Println(requestURL)
	contents, err := o.GetObolData(requestURL)

	var Obol microservices.ObolComplex
	err = json.Unmarshal(contents, &Obol)
	rates = Obol.Data

	return rates, err

}

//GetObolData makes a GET request to the plutus API and returns the data as a json array
func (o *ObolService) GetObolData(requestURL string) (contents []byte, err error) {
	res, err := config.HTTPClient.Get(requestURL)

	if err != nil {
		return contents, config.ErrorRequestTimeout
	}
	defer func() {
		_ = res.Body.Close()
	}()
	contents, err = ioutil.ReadAll(res.Body)

	if err != nil {
		return contents, err
	}

	return contents, err
}

//InitObolService initializes the connection with the Obol rate microservice
func InitObolService() *ObolService {

	rs := &ObolService{
		ObolURL: os.Getenv("OBOL_URL"),
	}
	return rs
}
