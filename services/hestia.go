package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/tokens/mrt"
	"github.com/grupokindynos/common/tokens/mvt"
	"github.com/grupokindynos/tyche/config"
)

//UpdateShift stores a shift in the database
func UpdateShift(hestiaURL string, shiftData hestia.Shift) (string, error) {

	fmt.Println(hestiaURL + "/shift")
	req, err := mvt.CreateMVTToken("POST", hestiaURL+"/shift", "tyche", os.Getenv("MASTER_PASSWORD"), shiftData, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))

	if err != nil {
		return "", err
	}
	res, err := config.HTTPClient.Do(req)

	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return "", err
	}
	headerSignature := res.Header.Get("service")
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("HESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return "", err
	}
	var response string
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return "", err
	}
	return response, nil
}
