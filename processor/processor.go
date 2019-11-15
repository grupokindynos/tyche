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
	"github.com/grupokindynos/olympus-utils/amount"
	"github.com/grupokindynos/tyche/models"
	"github.com/grupokindynos/tyche/services"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

func Start() {
	fmt.Println("Starting Shifts Processor")
	status, err := services.GetShiftStatus()
	if err != nil {
		panic(err)
	}
	if !status.Shift.Processor {
		fmt.Println("Shift Processor is Disabled")
		return
	}
	var wg sync.WaitGroup
	wg.Add(4)
	go handlePendingShifts(&wg)
	go handleConfirmingShifts(&wg)
	go handleConfirmedShifts(&wg)
	go handleRefundShifts(&wg)
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
		s.Status = hestia.GetShiftStatusString(hestia.ShiftStatusConfirming)
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
	for _, shift := range shifts {
		amountHandler := amount.AmountType(shift.ToAmount)
		paymentData := plutus.SendAddressBodyReq{
			Address: shift.ToAddress,
			Coin:    shift.ToCoin,
			Amount:  amountHandler.ToNormalUnit(),
		}
		txid, err := services.SubmitPayment(paymentData)
		if err != nil {
			fmt.Println("unable to submit refund payment")
			continue
		}
		shift.PaymentProof = txid
		shift.ProofTimestamp = time.Now().Unix()
		shift.Status = hestia.GetShiftStatusString(hestia.ShiftStatusComplete)
		_, err = services.UpdateShift(shift)
		if err != nil {
			fmt.Println("unable to update shift")
			continue
		}
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

func handleRefundShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	shifts, err := getRefundShifts()
	if err != nil {
		fmt.Println("Refund shifts processor finished with errors: " + err.Error())
		return
	}
	for _, shift := range shifts {
		paymentBody := plutus.SendAddressBodyReq{
			Address: shift.RefundAddr,
			Coin:    "POLIS",
			Amount:  amount.AmountType(shift.FeePayment.Amount).ToNormalUnit(),
		}
		_, err := services.SubmitPayment(paymentBody)
		if err != nil {
			fmt.Println("unable to submit refund payment")
			continue
		}
		shift.Status = hestia.GetShiftStatusString(hestia.ShiftStatusRefunded)
		_, err = services.UpdateShift(shift)
		if err != nil {
			fmt.Println("unable to update shift")
			continue
		}
	}
}

func getPendingShifts() ([]hestia.Shift, error) {
	return getShifts(hestia.ShiftStatusPending)
}

func getConfirmingShifts() ([]hestia.Shift, error) {
	return getShifts(hestia.ShiftStatusConfirming)
}

func getConfirmedShifts() ([]hestia.Shift, error) {
	return getShifts(hestia.ShiftStatusConfirmed)
}

func getRefundShifts() ([]hestia.Shift, error) {
	return getShifts(hestia.ShiftStatusRefund)
}

func getShifts(status hestia.ShiftStatus) ([]hestia.Shift, error) {
	req, err := mvt.CreateMVTToken("GET", hestia.ProductionURL+"/shift/all?filter="+hestia.GetShiftStatusString(status), "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
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
