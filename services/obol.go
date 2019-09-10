package services

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/grupokindynos/shift/config"
	"github.com/grupokindynos/shift/models/microservices"
)

//ObolService is the connection to the Obol rate microservice
type ObolService struct {
	ObolURL string
}

//GetRatesSimple gets the rate from a given coin
func (o *ObolService) GetRatesSimple(coin string) (rates map[string]interface{}, err error) {
	requestURL := o.ObolURL + "/simple/" + coin

	rates, err = o.GetObolData(requestURL)

	return rates, err

}

//GetRatesAmount gets the rate from a given coin, given the amount of coins it wants to change
func (o *ObolService) GetRatesAmount(fromcoin string, tocoin string, amount int) (rates map[string]interface{}, err error) {
	requestURL := o.ObolURL + "/complex/" + fromcoin + "/" + tocoin + "/?amount=" + strconv.Itoa(amount)

	rates, err = o.GetObolData(requestURL)

	return rates, err

}

//GetObolData makes a GET request to the plutus API and returns the data as a json array
func (o *ObolService) GetObolData(requestURL string) (rates map[string]interface{}, err error) {
	res, err := config.HTTPClient.Get(requestURL)

	if err != nil {
		return rates, config.ErrorRequestTimeout
	}
	defer func() {
		_ = res.Body.Close()
	}()
	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return rates, err
	}

	var Obol microservices.Obol
	err = json.Unmarshal(contents, &Obol)
	rates = Obol.Data

	return rates, err

}

//InitObolService initializes the connection with the Obol rate microservice
func InitObolService() *ObolService {

	rs := &ObolService{
		ObolURL: os.Getenv("OBOL_URL"),
	}
	return rs
}
