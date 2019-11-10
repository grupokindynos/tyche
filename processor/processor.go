package processor

import (
	"encoding/json"
	"errors"
	"fmt"
	coinfactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/coin-factory/coins"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/plutus"
	"github.com/grupokindynos/common/tokens/mrt"
	"github.com/grupokindynos/common/tokens/mvt"
	"github.com/grupokindynos/tyche/models"
	"github.com/grupokindynos/tyche/services"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"sync"
	"time"
)

func Start() {
	fmt.Println("Starting Shifts Processor")
	var wg sync.WaitGroup
	wg.Add(3)
	go handlePendingShifts(&wg)
	go handleConfirmingShifts(&wg)
	go handleConfirmedShifts(&wg)
	wg.Wait()
	fmt.Println("Shifts Processor Finished")
}

func handlePendingShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	shifts, err := getPendingShifts()
	if err != nil {
		fmt.Println("Pending shifts processor finished with errors: " + err.Error())
		return
	}
	for _, s := range shifts {
		if s.Timestamp+7200 < time.Now().Unix() {
			s.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
			_, err = services.UpdateShift(s)
			if err != nil {
				fmt.Println("Unable to update shift confirmations: " + err.Error())
				continue
			}
			continue
		}
		// TODO validate txs

		// Once the tx are fully validated we broadcast the information and mark the shift as confirming
		paymentCoinConfig, err := coinfactory.GetCoin(s.Payment.Coin)
		if err != nil {
			fmt.Println("Unable to get payment coin configuration: " + err.Error())
			continue
		}
		feeCoinConfig, err := coinfactory.GetCoin(s.FeePayment.Coin)
		if err != nil {
			fmt.Println("Unable to get fee coin configuration: " + err.Error())
			continue
		}
		txidFee, err := broadCastTx(feeCoinConfig, s.FeePayment.RawTx)
		if err != nil {
			fmt.Println("Unable to broadcast fee rawTx: " + err.Error())
			continue
		}
		s.FeePayment.RawTx = ""
		s.FeePayment.Txid = txidFee
		txidPayment, err := broadCastTx(paymentCoinConfig, s.Payment.RawTx)
		if err != nil {
			fmt.Println("Unable to broadcast payment rawTx: " + err.Error())
			// TODO if this happens, we need to refund fee payment.
			continue
		}
		s.Payment.RawTx = ""
		s.Payment.Txid = txidPayment
		s.Status = hestia.GetVoucherStatusString(hestia.VoucherStatusConfirming)
		_, err = services.UpdateShift(s)
		if err != nil {
			fmt.Println("Unable to update shift" + err.Error())
			continue
		}
	}
}

func handleConfirmedShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	shifts, err := getConfirmedShifts()
	if err != nil {
		fmt.Println("Confirmed shifts processor finished with errors: " + err.Error())
		return
	}
	for _, _ = range shifts {
		// TODO handle payment
	}
}

func handleConfirmingShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	shifts, err := getConfirmingShifts()
	if err != nil {
		fmt.Println("Confirming shifts processor finished with errors: " + err.Error())
		return
	}
	// Check confirmations and return
	for _, s := range shifts {
		paymentCoinConfig, err := coinfactory.GetCoin(s.Payment.Coin)
		if err != nil {
			fmt.Println("Unable to get payment coin configuration: " + err.Error())
			continue
		}
		feeCoinConfig, err := coinfactory.GetCoin(s.FeePayment.Coin)
		if err != nil {
			fmt.Println("Unable to get fee coin configuration: " + err.Error())
			continue
		}
		// Check if shift has enough confirmations
		if s.Payment.Confirmations >= int32(paymentCoinConfig.BlockchainInfo.MinConfirmations) && s.FeePayment.Confirmations >= int32(feeCoinConfig.BlockchainInfo.MinConfirmations) {
			s.Status = hestia.GetShiftStatusString(hestia.ShiftStatusConfirmed)
			_, err = services.UpdateShift(s)
			if err != nil {
				fmt.Println("Unable to update shift confirmations: " + err.Error())
				continue
			}
			continue
		}
		paymentConfirmations, err := getConfirmations(paymentCoinConfig, s.Payment.Txid)
		if err != nil {
			fmt.Println("Unable to get payment coin confirmations: " + err.Error())
			continue
		}
		feeConfirmations, err := getConfirmations(feeCoinConfig, s.FeePayment.Txid)
		if err != nil {
			fmt.Println("Unable to get fee coin confirmations: " + err.Error())
			continue
		}
		s.Payment.Confirmations = int32(paymentConfirmations)
		s.FeePayment.Confirmations = int32(feeConfirmations)
		_, err = services.UpdateShift(s)
		if err != nil {
			fmt.Println("Unable to update shift confirmations: " + err.Error())
			continue
		}
	}
}

func getPendingShifts() ([]hestia.Shift, error) {
	req, err := mvt.CreateMVTToken("GET", hestia.ProductionURL+"/shift/all?filter="+hestia.GetShiftStatusString(hestia.ShiftStatusPending), "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
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
	var response []hestia.Shift
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func getConfirmingShifts() ([]hestia.Shift, error) {
	req, err := mvt.CreateMVTToken("GET", hestia.ProductionURL+"/shift/all?filter="+hestia.GetVoucherStatusString(hestia.VoucherStatusConfirming), "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
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
	var response []hestia.Shift
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func getConfirmedShifts() ([]hestia.Shift, error) {
	req, err := mvt.CreateMVTToken("GET", hestia.ProductionURL+"/shift/all?filter="+hestia.GetVoucherStatusString(hestia.VoucherStatusConfirmed), "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
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
	var response []hestia.Shift
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func getConfirmations(coinConfig *coins.Coin, txid string) (int, error) {
	resp, err := http.Get(coinConfig.BlockExplorer + "/api/v1/tx/" + txid)
	if err != nil {
		return 0, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var response models.BlockbookTxInfo
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, err
	}
	return response.Confirmations, nil
}

func broadCastTx(coinConfig *coins.Coin, rawTx string) (txid string, err error) {
	resp, err := http.Get(coinConfig.BlockExplorer + "/api/v2/sendtx/" + rawTx)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var response models.BlockbookBroadcastResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}
	if response.Error != "" {
		return "", errors.New(response.Error)
	}
	return response.Result, nil
}

func verifyTransaction(transaction plutus.DecodedRawTX, toAddress string, amount int64) error {
	var isAddressOnTx, isAmountCorrect = false, false
	for _, vout := range transaction.Vout {
		if vout.ScriptPubKey.Addresses[0] == toAddress {
			isAddressOnTx = true
		}
		amountToSat := int64(math.Round(vout.Value * 1e8))
		totalAmount := amount
		if amountToSat == totalAmount {
			isAmountCorrect = true
		}
	}
	if isAddressOnTx == false {
		return errors.New("no matching address in raw tx")
	}
	if isAmountCorrect == false {
		return errors.New("incorrect amount in raw tx")
	}

	return nil
}
