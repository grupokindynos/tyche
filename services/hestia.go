package services

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/tokens/mrt"
	"github.com/grupokindynos/common/tokens/mvt"
)

type HestiaRequests struct {
	HestiaURL string
}

func (a *HestiaRequests) GetShiftStatus() (hestia.Config, error) {
	req, err := mvt.CreateMVTToken("GET", os.Getenv(a.HestiaURL)+"config", "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return hestia.Config{}, err
	}
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return hestia.Config{}, err
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return hestia.Config{}, err
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return hestia.Config{}, err
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		return hestia.Config{}, errors.New("no header signature")
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("HESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return hestia.Config{}, err
	}
	var response hestia.Config
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return hestia.Config{}, err
	}
	return response, nil
}

func (a *HestiaRequests) GetCoinsConfig() ([]hestia.Coin, error) {
	req, err := mvt.CreateMVTToken("GET", os.Getenv(a.HestiaURL)+"/coins", "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return nil, err
	}
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return nil, err
	}
	headerSignature := res.Header.Get("service")
	if headerSignature == "" {
		return nil, errors.New("no header signature")
	}
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("HESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return nil, err
	}
	var response []hestia.Coin
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (a *HestiaRequests) UpdateShift(shiftData hestia.Shift) (string, error) {
	req, err := mvt.CreateMVTToken("POST", os.Getenv(a.HestiaURL)+"/shift", "tyche", os.Getenv("MASTER_PASSWORD"), shiftData, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return "", err
	}
	client := http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       time.Second * 30,
	}
	res, err := client.Do(req)
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

func (a *HestiaRequests) UpdateShiftV2(shiftData hestia.ShiftV2) (string, error) {
	// TODO new shift v2 endpoint
	req, err := mvt.CreateMVTToken("POST", os.Getenv(a.HestiaURL)+"/shift2", "tyche", os.Getenv("MASTER_PASSWORD"), shiftData, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return "", err
	}
	client := http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       time.Second * 30,
	}
	res, err := client.Do(req)
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

func (a *HestiaRequests) GetShiftsByTimestampV2(uid string, timestamp string) (shifts []hestia.ShiftV2, err error) {
	req, err := mvt.CreateMVTToken("GET", os.Getenv(a.HestiaURL)+"/shift2/all_by_timestamp?timestamp="+timestamp+"&userid="+uid, "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
	if err != nil {
		return shifts, err
	}
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return shifts, err
	}
	defer res.Body.Close()
	tokenResponse, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return shifts, err
	}
	var tokenString string
	err = json.Unmarshal(tokenResponse, &tokenString)
	if err != nil {
		return shifts, err
	}
	headerSignature := res.Header.Get("service")
	valid, payload := mrt.VerifyMRTToken(headerSignature, tokenString, os.Getenv("HESTIA_PUBLIC_KEY"), os.Getenv("MASTER_PASSWORD"))
	if !valid {
		return shifts, err
	}
	err = json.Unmarshal(payload, &shifts)
	if err != nil {
		return shifts, err
	}
	return shifts, nil
}
