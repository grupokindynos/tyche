package processor

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grupokindynos/common/explorer"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	coinfactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/coin-factory/coins"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/plutus"
	"github.com/grupokindynos/common/tokens/mrt"
	"github.com/grupokindynos/common/tokens/mvt"
	"github.com/grupokindynos/tyche/services"
	msgBot "github.com/grupokindynos/tyche/telegram"
	"github.com/shopspring/decimal"
)

type Processor struct {
	Hestia          services.HestiaService
	Plutus          services.PlutusService
	HestiaURL       string
	SkipValidations bool
}

var teleBot msgBot.Bot

func (p *Processor) Start() {
	fmt.Println("Starting Shifts Processor")
	teleBot = *msgBot.GetInstance()
	status, err := p.Hestia.GetShiftStatus()
	if err != nil {
		panic(err)
	}
	if !status.Shift.Processor {
		fmt.Println("Shift Processor is Disabled")
		return
	}
	var wg sync.WaitGroup
	wg.Add(4)
	go p.handlePendingShifts(&wg)
	go p.handleConfirmingShifts(&wg)
	go p.handleConfirmedShifts(&wg)
	go p.handleRefundShifts(&wg)
	wg.Wait()
	fmt.Println("Shifts Processor Finished")
}

func (p *Processor) handlePendingShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	shifts, err := p.getPendingShifts()
	if err != nil {
		fmt.Println("Pending shifts processor finished with errors: " + err.Error())
		teleBot.SendError("Pending shifts processor finished with errors: " + err.Error())
		return
	}
	for _, s := range shifts {
		if s.Timestamp+7200 < time.Now().Unix() {
			s.Status = hestia.GetShiftStatusString(hestia.ShiftStatusError)
			_, err = p.Hestia.UpdateShift(s)
			if err != nil {
				fmt.Println("Unable to update shift confirmations: " + err.Error())
				continue
			}
			continue
		}
		s.Status = hestia.GetShiftStatusString(hestia.ShiftStatusConfirming)
		_, err = p.Hestia.UpdateShift(s)
		if err != nil {
			fmt.Println("Unable to update shift " + err.Error())
			teleBot.SendError("Unable to update shift: " + err.Error() + "\n Shift ID: " + s.ID)
			continue
		}
	}
}

func (p *Processor) handleConfirmedShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	shifts, err := p.getConfirmedShifts()
	if err != nil {
		fmt.Println("Confirmed shifts processor finished with errors: " + err.Error())
		teleBot.SendError("Confirmed shifts processor finished with errors: " + err.Error())
		return
	}
	for _, shift := range shifts {
		amount, _ := decimal.NewFromInt(shift.ToAmount).DivRound(decimal.NewFromInt(1e8), 8).Float64()
		paymentData := plutus.SendAddressBodyReq{
			Address: shift.ToAddress,
			Coin:    shift.ToCoin,
			Amount:  amount,
		}
		txid, err := p.Plutus.SubmitPayment(paymentData)
		if err != nil {
			fmt.Println("unable to submit payment")
			teleBot.SendError("Unable to submit payment: " + err.Error() + "\n Shift ID: " + shift.ID)
			continue
		}
		shift.PaymentProof = txid
		shift.ProofTimestamp = time.Now().Unix()
		shift.Status = hestia.GetShiftStatusString(hestia.ShiftStatusComplete)
		_, err = p.Hestia.UpdateShift(shift)
		if err != nil {
			fmt.Println("unable to update shift")
			teleBot.SendError("Unable to update shift: " + err.Error() + "\n Shift ID: " + shift.ID)
			continue
		}
	}
}

func (p *Processor) handleConfirmingShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	shifts, err := p.getConfirmingShifts()
	if err != nil {
		fmt.Println("Confirming shifts processor finished with errors: " + err.Error())
		teleBot.SendError("Confirming shifts processor finished with errors: " + err.Error())

		return
	}
	// Check confirmations and return
	for _, s := range shifts {
		paymentCoinConfig, err := coinfactory.GetCoin(s.Payment.Coin)
		if err != nil {
			fmt.Println("Unable to get payment coin configuration: " + err.Error())
			teleBot.SendError("Unable to get payment coin configuration: " + err.Error() + "\n Shift ID: " + s.ID)
			continue
		}
		// Processor should only validate Payment coin if tx comes from or to POLIS
		// Conditional statement is logic for the negation for "if its from or to polis"
		if s.Payment.Coin != "POLIS" && s.FeePayment.Coin != "POLIS" {
			feeCoinConfig, err := coinfactory.GetCoin(s.FeePayment.Coin)
			if err != nil {
				fmt.Println("Unable to get fee coin configuration: " + err.Error())
				teleBot.SendError("Unable to get fee coin configuration: " + err.Error() + "\n Shift ID: " + s.ID)
				continue
			}
			// Check if shift has enough confirmations
			if p.SkipValidations || (s.Payment.Confirmations >= int32(paymentCoinConfig.BlockchainInfo.MinConfirmations) && s.FeePayment.Confirmations >= int32(feeCoinConfig.BlockchainInfo.MinConfirmations)) {
				s.Status = hestia.GetShiftStatusString(hestia.ShiftStatusConfirmed)
				_, err = p.Hestia.UpdateShift(s)
				if err != nil {
					fmt.Println("Unable to update shift confirmations: " + err.Error())
					continue
				}
				continue
			}
			err = checkTxId(&s.FeePayment)
			if err != nil {
				fmt.Println("Unable to get fee txId " + err.Error())
				teleBot.SendError("Unable to get fee txId: " + err.Error() + "\n Shift ID: " + s.ID)
				continue
			}
			feeConfirmations, err := p.getConfirmations(feeCoinConfig, s.FeePayment.Txid)
			if err != nil {
				fmt.Println("Unable to get fee coin confirmations: " + err.Error())
				continue
			}
			s.FeePayment.Confirmations = int32(feeConfirmations)
		} else {
			// Check if shift has enough confirmations
			if p.SkipValidations || s.Payment.Confirmations >= int32(paymentCoinConfig.BlockchainInfo.MinConfirmations) {
				s.Status = hestia.GetShiftStatusString(hestia.ShiftStatusConfirmed)
				_, err = p.Hestia.UpdateShift(s)
				if err != nil {
					fmt.Println("Unable to update shift confirmations: " + err.Error())
					continue
				}
				continue
			}
		}

		err = checkTxId(&s.Payment)
		if err != nil {
			fmt.Println("Unable to get txId ", s.Payment, err.Error())
			teleBot.SendError("Unable to get txId: " + err.Error() + "\n Shift ID: " + s.ID)
			continue
		}
		paymentConfirmations, err := p.getConfirmations(paymentCoinConfig, s.Payment.Txid)
		if err != nil {
			fmt.Println("Unable to get payment coin confirmations: " + err.Error())
			continue
		}
		s.Payment.Confirmations = int32(paymentConfirmations)
		_, err = p.Hestia.UpdateShift(s)
		if err != nil {
			fmt.Println("Unable to update shift confirmations: " + err.Error())
			continue
		}
	}
}

func (p *Processor) handleRefundShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	shifts, err := p.getRefundShifts()
	if err != nil {
		fmt.Println("Refund shifts processor finished with errors: " + err.Error())
		teleBot.SendError("Refund shifts processor finished with errors: " + err.Error())
		return
	}
	for _, shift := range shifts {
		if shift.Payment.Coin == "POLIS" {
			amount, _ := decimal.NewFromInt(shift.Payment.Amount).DivRound(decimal.NewFromInt(1e8), 8).Float64()
			paymentBody := plutus.SendAddressBodyReq{
				Address: shift.RefundAddr,
				Coin:    "POLIS",
				Amount:  amount,
			}
			_, err = p.Plutus.SubmitPayment(paymentBody)
			if err != nil {
				fmt.Println("unable to submit refund payment")
				teleBot.SendError("Unable to submit refund payment: " + err.Error() + "\n Shift ID: " + shift.ID)
				continue
			}
			shift.Status = hestia.GetShiftStatusString(hestia.ShiftStatusRefunded)
			_, err = p.Hestia.UpdateShift(shift)
			if err != nil {
				fmt.Println("unable to update shift")
				teleBot.SendError("Unable to update shift: " + err.Error() + "\n Shift ID: " + shift.ID)
				continue
			}
			continue
		}
		amountFee, _ := decimal.NewFromInt(shift.FeePayment.Amount).DivRound(decimal.NewFromInt(1e8), 8).Float64()
		paymentBody := plutus.SendAddressBodyReq{
			Address: shift.RefundAddr,
			Coin:    "POLIS",
			Amount:  amountFee,
		}
		_, err = p.Plutus.SubmitPayment(paymentBody)
		if err != nil {
			fmt.Println("unable to submit refund payment")
			teleBot.SendError("Unable to submit refund payment: " + err.Error() + "\n Shift ID: " + shift.ID)
			continue
		}
		shift.Status = hestia.GetShiftStatusString(hestia.ShiftStatusRefunded)
		_, err = p.Hestia.UpdateShift(shift)
		if err != nil {
			fmt.Println("unable to update shift")
			teleBot.SendError("Unable to update shift: " + err.Error() + "\n Shift ID: " + shift.ID)
			continue
		}
	}
}

func (p *Processor) getPendingShifts() ([]hestia.Shift, error) {
	return p.getShifts(hestia.ShiftStatusPending)
}

func (p *Processor) getConfirmingShifts() ([]hestia.Shift, error) {
	return p.getShifts(hestia.ShiftStatusConfirming)
}

func (p *Processor) getConfirmedShifts() ([]hestia.Shift, error) {
	return p.getShifts(hestia.ShiftStatusConfirmed)
}

func (p *Processor) getRefundShifts() ([]hestia.Shift, error) {
	return p.getShifts(hestia.ShiftStatusRefund)
}

func (p *Processor) getShifts(status hestia.ShiftStatus) ([]hestia.Shift, error) {
	req, err := mvt.CreateMVTToken("GET", os.Getenv(p.HestiaURL)+"/shift/all?filter="+hestia.GetShiftStatusString(status), "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
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

func (p *Processor) getConfirmations(coinConfig *coins.Coin, txid string) (int, error) {
	if coinConfig.Info.Token && coinConfig.Info.Tag != "ETH" {
		coinConfig, _ = coinfactory.GetCoin("ETH")
	}
	explorerWrapper, _ := explorer.NewExplorerFactory().GetExplorerByCoin(*coinConfig)
	txData, err := explorerWrapper.GetTx(txid)
	if err != nil {
		return 0, err
	}
	return txData.Confirmations, nil
}
