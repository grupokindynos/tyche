package processor

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/grupokindynos/adrestia-go/models"
	"github.com/grupokindynos/common/blockbook"
	coinFactory "github.com/grupokindynos/common/coin-factory"
	"github.com/grupokindynos/common/coin-factory/coins"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/obol"
	"github.com/grupokindynos/common/plutus"
	"github.com/grupokindynos/common/tokens/mrt"
	"github.com/grupokindynos/common/tokens/mvt"
	"github.com/grupokindynos/tyche/services"
	msgBot "github.com/grupokindynos/tyche/telegram"
	"github.com/olympus-protocol/ogen/utils/amount"
)

type TycheProcessorV2 struct {
	Hestia          services.HestiaService
	Plutus          services.PlutusService
	Obol            obol.ObolService
	Adrestia        services.AdrestiaService
	HestiaURL       string
	SkipValidations bool
}

var teleBot2 msgBot.Bot

func (p *TycheProcessorV2) Start() {
	fmt.Println("Starting Shifts Processor")
	teleBot2 = *msgBot.GetInstance()
	status, err := p.Hestia.GetShiftStatus()
	if err != nil {
		panic(err)
	}
	if !status.Shift.Processor {
		fmt.Println("Shift Processor is Disabled")
		return
	}
	var wg sync.WaitGroup
	wg.Add(3)
	go p.handleCreatedShifts(&wg)
	go p.handleProcessingShifts(&wg)
	go p.handleRefundShifts(&wg)
	wg.Wait()
	fmt.Println("Shifts Processor Finished")
}

/*
Handles created shifts, checks for minimum number of confirmations on blockchain. If txid is missing, it tries to find the txid.
Kick starts the ProcessingShifts method.
*/
func (p *TycheProcessorV2) handleCreatedShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	shifts, err := p.getShifts(hestia.ShiftStatusV2Created)
	fmt.Println("Created Shifts", shifts)
	if err != nil {
		fmt.Println("Confirming shifts processor finished with errors: " + err.Error())
		teleBot.SendError("Confirming shifts processor finished with errors: " + err.Error())
		return
	}
	// Check confirmations and return
	for _, s := range shifts {
		paymentCoinConfig, err := coinFactory.GetCoin(s.Payment.Coin)
		if err != nil {
			fmt.Println("Unable to get payment coin configuration: " + err.Error())
			teleBot.SendError("Unable to get payment coin configuration: " + err.Error() + "\n Shift ID: " + s.ID)
			continue
		}
		if paymentCoinConfig.Info.Token && paymentCoinConfig.Info.Tag != "ETH" {
			ethConfig, err := coinFactory.GetCoin("ETH")
			if err != nil {
				s.Message = "could not get token data"
				continue
			}
			paymentCoinConfig.BlockchainInfo = ethConfig.BlockchainInfo
		}

		//Check for missing txid
		err = checkTxIdWithFee(&s.Payment)
		if err != nil {
			fmt.Println("Unable to get txId " + err.Error())
			teleBot.SendError("Unable to get txId: " + err.Error() + "\n Shift ID: " + s.ID)
			continue
		}

		paymentConfirmations, err := p.getConfirmations(paymentCoinConfig, s.Payment.Txid)
		if err != nil {
			s.Message = "could not get payment confirmations"
			continue
		}
		s.Payment.Confirmations = int32(paymentConfirmations)

		// New one payment global validation. Check if shift has enough confirmations. @TODO Handle ERC20
		if p.SkipValidations || s.Payment.Confirmations >= int32(paymentCoinConfig.BlockchainInfo.MinConfirmations) {
			s.Status = hestia.ShiftStatusV2ProcessingOrders
			s.OutboundTrade.Status = hestia.ShiftV2TradeStatusCreated
			_, err = p.Hestia.UpdateShiftV2(s)
			if err != nil {
				fmt.Println(s.ID, " Unable to update shift confirmations: " + err.Error())
				continue
			}
			if s.InboundTrade.Conversions == nil {
				s.InboundTrade.Status = hestia.ShiftV2TradeStatusCompleted
			}
			if s.OutboundTrade.Conversions == nil {
				s.OutboundTrade.Status = hestia.ShiftV2TradeStatusCompleted
			}
			continue
		}


		s.Status = hestia.ShiftStatusV2ProcessingOrders
		_, err = p.Hestia.UpdateShiftV2(s)
		if err != nil {
			fmt.Println("Unable to update shift confirmations: " + err.Error())
			continue
		}
	}
}

func (p *TycheProcessorV2) handleProcessingShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	processingShifts, err := p.getProcessingShifts()
	fmt.Println("processing shifts", processingShifts)
	if err != nil {
		// telegram bot
		return
	}
	sentToUserShifts, err := p.getSentToUserShifts()
	fmt.Println("sent user shifts", sentToUserShifts)
	if err != nil {
		log.Println(err)
		return
	}
	shifts := append(processingShifts, sentToUserShifts...)
	for _, shift := range shifts {
		var trades [2]*hestia.DirectionalTrade
		trades[0] = &shift.InboundTrade
		trades[1] = &shift.OutboundTrade

		for key, trade := range trades {
			switch trade.Status {
			case hestia.ShiftV2TradeStatusInitialized:
				if key == 0 {
					p.handleInboundDeposit(&shift)
				}
			case hestia.ShiftV2TradeStatusCreated:
				p.handleCreatedTrade(trade)
				break
			case hestia.ShiftV2TradeStatusPerforming:
				p.handlePerformedTrade(trade)
				break
			case hestia.ShiftV2TradeStatusCompleted:
				if key == 1 { // if this is an outbound trade
					lastPos := len(trade.Conversions) - 1
					withdrawAmount := 0.0
					toAmount, _ := decimal.NewFromInt(shift.ToAmount).Mul(decimal.NewFromFloat(1e-8)).Float64()
					if trade.Conversions[lastPos].ReceivedAmount > toAmount {
						withdrawAmount = toAmount
					} else {
						withdrawAmount = trade.Conversions[lastPos].ReceivedAmount
					}
					res, err := p.Adrestia.Withdraw(models.WithdrawParams{
						Address: shift.ToAddress,
						Asset:   shift.ToCoin,
						Amount:  withdrawAmount,
					})
					if err != nil {
						log.Println(err)
						continue
					}
					trade.Status = hestia.ShiftV2TradeStatusWithdrawn
					shift.Status = hestia.ShiftStatusV2SentToUser
					shift.PaymentProof = res.TxId
					shift.ProofTimestamp = time.Now().Unix()
				}
				break
			case hestia.ShiftV2TradeStatusWithdrawn:
				txId, err := p.Adrestia.GetWithdrawalTxHash(models.WithdrawInfo{
					Exchange: trade.Exchange,
					Asset:    shift.ToCoin,
					TxId:     shift.PaymentProof,
				})
				if err != nil {
					log.Println(err)
					continue
				}
				if txId != "" {
					shift.PaymentProof = txId
					trade.Status = hestia.ShiftV2TradeStatusUserDeposit
				} else if (time.Now().Unix() - shift.ProofTimestamp) > 10 * 60 {
					trade.Status = hestia.ShiftV2TradeStatusWithdrawCompleted
				}
				break
			case hestia.ShiftV2TradeStatusUserDeposit:
				amountReceived, err := getUserReceivedAmount(shift.ToCoin, shift.ToAddress, shift.PaymentProof)
				if err != nil {
					log.Println(err)
					continue
				}
				shift.UserReceivedAmount = amountReceived
				trade.Status = hestia.ShiftV2TradeStatusWithdrawCompleted
			}
		}

		if trades[0].Status == hestia.ShiftV2TradeStatusCompleted && trades[1].Status == hestia.ShiftV2TradeStatusWithdrawCompleted {
			shift.Status = hestia.ShiftStatusV2Complete
		}

		_, err := p.Hestia.UpdateShiftV2(shift)
		if err != nil {
			log.Println(err)
			//telegram bot
		}
	}
}

func (p *TycheProcessorV2) handleRefundShifts(wg *sync.WaitGroup) {
	defer wg.Done()
	shifts, err := p.getRefundShifts()
	if err != nil {
		fmt.Println("Refund shifts processor finished with errors: " + err.Error())
		teleBot.SendError("Refund shifts processor finished with errors: " + err.Error())
		return
	}
	for _, shift := range shifts {
		// TODO Handle refunds in a coin's coin
		if shift.Payment.Coin == "POLIS" {
			paymentBody := plutus.SendAddressBodyReq{
				Address: shift.RefundAddr,
				Coin:    "POLIS",
				Amount:  amount.AmountType(shift.Payment.Amount).ToNormalUnit(),
			}
			_, err := p.Plutus.SubmitPayment(paymentBody)
			if err != nil {
				fmt.Println("unable to submit refund payment")
				teleBot.SendError("Unable to submit refund payment: " + err.Error() + "\n Shift ID: " + shift.ID)
				continue
			}
			shift.Status = hestia.ShiftStatusV2Refunded
			_, err = p.Hestia.UpdateShiftV2(shift)
			if err != nil {
				fmt.Println("unable to update shift")
				teleBot.SendError("Unable to update shift: " + err.Error() + "\n Shift ID: " + shift.ID)
				continue
			}
			continue
		}
	}
}

/* func (p *TycheProcessorV2) askForExchangeDeposit(payment hestia.PaymentWithFee) (depositInfo models.DepositInfo, err error){
	depositInfo, err = p.Adrestia.DepositInfo(models.DepositParams{
		Asset:   payment.Coin,
		TxId:    payment.Txid,
		Address: payment.Address,
	})
	if err != nil {
		return
	}
	return
} */

func (p *TycheProcessorV2) handleCreatedTrade(trade *hestia.DirectionalTrade) {
	txId, err := p.Adrestia.Trade(trade.Conversions[0])
	if err != nil {
		log.Println(err)
		return
	}
	trade.Conversions[0].CreatedTime = time.Now().Unix()
	trade.Conversions[0].OrderId = txId
	trade.Conversions[0].Status = hestia.ExchangeOrderStatusOpen
	trade.Status = hestia.ShiftV2TradeStatusPerforming
}

func (p *TycheProcessorV2) handlePerformedTrade(trade *hestia.DirectionalTrade) {
	if trade.Conversions[0].Status == hestia.ExchangeOrderStatusCompleted {
		status, err := p.checkTradeStatus(&trade.Conversions[1])
		if err != nil {
			log.Println(err)
			return
		}
		if status == hestia.ExchangeOrderStatusCompleted {
			trade.Status = hestia.ShiftV2TradeStatusCompleted
		}
	} else {
		status, err := p.checkTradeStatus(&trade.Conversions[0])
		if err != nil {
			log.Println(err)
			return
		}
		if status == hestia.ExchangeOrderStatusCompleted {
			if len(trade.Conversions) > 1 {
				trade.Conversions[1].Amount = trade.Conversions[0].ReceivedAmount
				txId, err := p.Adrestia.Trade(trade.Conversions[1])
				if err != nil {
					log.Println(err)
					return
				}
				trade.Conversions[1].CreatedTime = time.Now().Unix()
				trade.Conversions[1].Status = hestia.ExchangeOrderStatusOpen
				trade.Conversions[1].OrderId = txId
			} else {
				trade.Status = hestia.ShiftV2TradeStatusCompleted
			}
		}
	}
}

func (p *TycheProcessorV2) checkTradeStatus(trade *hestia.Trade) (hestia.ExchangeOrderStatus, error) {
	tradeInfo, err := p.Adrestia.GetTradeStatus(*trade)
	if err != nil {
		return hestia.ExchangeOrderStatusError, err
	}
	if tradeInfo.Status == hestia.ExchangeOrderStatusCompleted {
		trade.ReceivedAmount = tradeInfo.ReceivedAmount
		trade.Status = hestia.ExchangeOrderStatusCompleted
		trade.FulfilledTime = time.Now().Unix()
	}

	return tradeInfo.Status, nil
}

func (p *TycheProcessorV2) getPendingShifts() ([]hestia.ShiftV2, error) {
	return p.getShifts(hestia.ShiftStatusV2Created)
}

func (p *TycheProcessorV2) getSentToUserShifts() ([]hestia.ShiftV2, error) {
	return p.getShifts(hestia.ShiftStatusV2SentToUser)
}

func (p *TycheProcessorV2) getProcessingShifts() ([]hestia.ShiftV2, error) {
	return p.getShifts(hestia.ShiftStatusV2ProcessingOrders)
}

func (p *TycheProcessorV2) getConfirmingShifts() ([]hestia.ShiftV2, error) {
	return p.getShifts(hestia.ShiftStatusV2ProcessingOrders)
}

func (p *TycheProcessorV2) getConfirmedShifts() ([]hestia.ShiftV2, error) {
	return p.getShifts(hestia.ShiftStatusV2Confirmed)
}

func (p *TycheProcessorV2) getRefundShifts() ([]hestia.ShiftV2, error) {
	return p.getShifts(hestia.ShiftStatusV2Refunded)
}

func (p *TycheProcessorV2) getShifts(status hestia.ShiftStatusV2) ([]hestia.ShiftV2, error) {
	req, err := mvt.CreateMVTToken("GET", os.Getenv(p.HestiaURL)+"/shift2/all?filter="+strconv.FormatInt(int64(status), 10), "tyche", os.Getenv("MASTER_PASSWORD"), nil, os.Getenv("HESTIA_AUTH_USERNAME"), os.Getenv("HESTIA_AUTH_PASSWORD"), os.Getenv("TYCHE_PRIV_KEY"))
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
	var response []hestia.ShiftV2
	err = json.Unmarshal(payload, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (p *TycheProcessorV2) getConfirmations(coinConfig *coins.Coin, txid string) (int, error) {
	if coinConfig.Info.Token && coinConfig.Info.Tag != "ETH" {
		coinConfig, _ = coinFactory.GetCoin("ETH")
	}
	blockbookWrapper := blockbook.NewBlockBookWrapper(coinConfig.Info.Blockbook)
	txData, err := blockbookWrapper.GetTx(txid)
	if err != nil {
		return 0, err
	}
	return txData.Confirmations, nil
}

func (p *TycheProcessorV2) handleInboundDeposit(shift *hestia.ShiftV2) {
	depositInfo, err := p.Adrestia.DepositInfo(models.DepositParams{
		Asset:   shift.Payment.Coin,
		TxId:    shift.Payment.Txid,
		Address: shift.Payment.Address,
	})
	if err != nil {
		log.Println(err)
		return
	}
	if depositInfo.DepositInfo.Status == hestia.ExchangeOrderStatusCompleted {
		shift.InboundTrade.Status = hestia.ShiftV2TradeStatusCreated
		return
	}
	return
}
