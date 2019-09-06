package services

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/grupokindynos/shift/config"
	"github.com/grupokindynos/shift/models/microservices"
)

//ObolService is the connection to the Obol rate microservice
type ObolService struct {
	ObolURL string
}

//GetRatesSimple gets the rate from a given coin
func (o *ObolService) GetRatesSimple(coin string) (rates map[string]float64, err error) {
	requestURL := o.ObolURL + "/simple/" + coin
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
