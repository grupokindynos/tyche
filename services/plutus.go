package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/grupokindynos/tyche/config"
	"github.com/grupokindynos/tyche/models/microservices"
)

//PlutusService is the connection handler to the Plutus wallets microservice
type PlutusService struct {
	PlutusURL    string
	AuthUsername string
	AuthPassword string
}

//GetWalletStatus gets the wallet status from a given coin
func (ps *PlutusService) GetWalletStatus(coin string) (status interface{}, err error) {
	requestURL := ps.PlutusURL + "/status/" + coin

	status, err = ps.GetPlutusData(requestURL)

	return status, err

}

//GetWalletInfo gets the wallet information from a given coin
func (ps *PlutusService) GetWalletInfo(coin string) (status interface{}, err error) {
	requestURL := ps.PlutusURL + "/info/" + coin

	status, err = ps.GetPlutusData(requestURL)

	return status, err

}

//GetWalletBalance gets the wallet balance from a given coin
func (ps *PlutusService) GetWalletBalance(coin string) (status interface{}, err error) {
	requestURL := ps.PlutusURL + "/info/" + coin

	status, err = ps.GetPlutusData(requestURL)

	return status, err

}

//GetWalletTXID gets the transaction id information of a given coin and txid
func (ps *PlutusService) GetWalletTXID(coin string, txid string) (status interface{}, err error) {
	requestURL := ps.PlutusURL + "/tx/" + coin + "/" + txid

	status, err = ps.GetPlutusData(requestURL)

	return status, err

}

//GetWalletAddress gets a deposit address from a given coin
func (ps *PlutusService) GetWalletAddress(coin string) (status interface{}, err error) {
	requestURL := ps.PlutusURL + "/address/" + coin

	data, err := ps.GetPlutusData(requestURL)

	status = data
	fmt.Println(status)
	fmt.Println(coin)

	return status, err

}

//GetPlutusData makes a GET request to the plutus API and returns the data as a json array
func (ps *PlutusService) GetPlutusData(requestURL string) (data interface{}, err error) {
	req, _ := http.NewRequest("GET", requestURL, nil)
	req.SetBasicAuth(ps.AuthUsername, ps.AuthPassword)

	res, err := config.HTTPClient.Do(req)

	if err != nil {
		return data, config.ErrorRequestTimeout
	}
	defer func() {
		_ = res.Body.Close()
	}()
	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return data, err
	}

	fmt.Println(string(contents))
	var Plutus microservices.Plutus
	err = json.Unmarshal(contents, &Plutus)
	data = Plutus.Data

	return data, err

}

//InitPlutusService initializes the connection with the Plutus wallets microservice
func InitPlutusService() *PlutusService {

	rs := &PlutusService{
		PlutusURL:    os.Getenv("PLUTUS_URL"),
		AuthUsername: os.Getenv("PLUTUS_USERNAME"),
		AuthPassword: os.Getenv("PLUTUS_PASSWORD"),
	}
	return rs
}
