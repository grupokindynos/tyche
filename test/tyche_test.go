package test

/*
import (
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	models2 "github.com/grupokindynos/adrestia-go/models"
	"github.com/grupokindynos/common/hestia"
	"github.com/grupokindynos/common/obol"
	obolMocks "github.com/grupokindynos/common/obol/mocks"
	"github.com/grupokindynos/common/plutus"
	"github.com/grupokindynos/tyche/controllers"
	"github.com/grupokindynos/tyche/mocks"
	"github.com/grupokindynos/tyche/models"
	"testing"
)

func TestStatus(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testError := errors.New("Testing error")
	emptyHestiaConfig := hestia.Config{}
	hestiaAvailable := hestia.Config{Shift: hestia.Available{Service: true}}

	mockHestiaService := mocks.NewMockHestiaService(mockCtrl)
	testTyche := &controllers.TycheControllerV2{Hestia: mockHestiaService}

	gomock.InOrder(
		mockHestiaService.EXPECT().GetShiftStatus().Return(emptyHestiaConfig, testError),
		mockHestiaService.EXPECT().GetShiftStatus().Return(hestiaAvailable, nil),
	)

	// Test error returned from hestia
	status, err := testTyche.StatusV2("dummy string", []byte{10, 10, 10}, models.Params{Coin: "BTC"})

	if err != testError {
		t.Fatal("Test error returned - error is not equal to testing error")
	}

	if status != nil {
		t.Fatal("Test error returned - status is not equal to nil")
	}

	// Test shift is available
	status, err = testTyche.StatusV2("dummy uid", []byte{10, 10, 10}, models.Params{Coin: "BTC"})

	if err != nil {
		t.Fatal("Test shift available - error is not equal to nil")
	}

	if status != true {
		t.Fatal("Test shift available - status is not equal to true")
	}
}

func TestBalance(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	testError := errors.New("Testing error")
	emptyPlutusBalance := plutus.Balance{}
	plutusBalance := plutus.Balance{Confirmed: 120.35, Unconfirmed: 30.34}

	mockPlutusService := mocks.NewMockPlutusService(mockCtrl)
	testTyche := &controllers.TycheControllerV2{Plutus: mockPlutusService}

	gomock.InOrder(
		mockPlutusService.EXPECT().GetWalletBalance(gomock.Eq("BTC")).Return(emptyPlutusBalance, testError),
		mockPlutusService.EXPECT().GetWalletBalance(gomock.Eq("POLIS")).Return(plutusBalance, nil),
	)

	// Test error returned from plutus
	balance, err := testTyche.BalanceV2("dummy uid", []byte{10, 10, 10}, models.Params{Coin: "BTC"})

	if err != testError {
		t.Fatal("Test error returned - error is not equal to testing error")
	}

	if balance != nil {
		t.Fatal("Test error returned - balance is not equal to nil")
	}

	// Test balance returned from plutus
	balance, err = testTyche.BalanceV2("dummy uid", []byte{10, 10, 10}, models.Params{Coin: "POLIS"})

	if err != nil {
		t.Fatal("Test returned balance - error is not equal to nil")
	}

	if balance != plutusBalance {
		t.Fatal("Test returned balance - returned balance doesn't match")
	}
}

/*func TestPrepare(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	coin2CoinResponse := obol.CoinToCoinWithAmountResponse{AveragePrice: 45.324998}
	prepareData := models.PrepareShiftRequest{
		FromCoin:  "POLIS",
		Amount:    50,
		ToCoin:    "BTC",
		ToAddress: "dummy address",
	}
	payload, _ := json.Marshal(prepareData)

	paymentAddress := "123dummyAddress456"
	shiftId := "123456789"
	userId := "uid-random"
	params := models.Params{Coin: "POLIS"}

	// Empty fee payment because is from POLIS
	// feePayment := models.PaymentInfo{}

	amountHandler := amount.AmountType(prepareData.Amount)
	rateAmountHandler, _ := amount.NewAmount(coin2CoinResponse.AveragePrice)
	ToAmount, _ := amount.NewAmount(amountHandler.ToNormalUnit() * rateAmountHandler.ToNormalUnit())


		//payment := models.PaymentInfo{
		//		Address: paymentAddress,
		//		Amount:  prepareData.Amount,
		//	}

	payment := models.PaymentInfoV2{
		Address: models2.AddressResponse{
			Coin:            "POLIS",
			ExchangeAddress: models2.ExchangeAddress{
				Address:  paymentAddress,
				Exchange: "southxchange",
			},
		},
		Fee: 0.0,
		Amount:  prepareData.Amount,
		Total: 0.0 + prepareData.Amount,
		HasFee: false,
		Rate: int64(rateAmountHandler.ToNormalUnit()),
		FiatInfo: models.ExpectedFiatAmount{},
		Conversions: models2.PathResponse{},
	}

	//prepareResponse := models.PrepareShiftResponse{
	//	Payment:        payment,
	//	Fee:            feePayment,
	//	ReceivedAmount: int64(ToAmount.ToUnit(amount.AmountSats)),
	//}
	//prepareResponse := models.PrepareShiftResponseV2{
	//	Payment:        payment,
	//	ReceivedAmount: int64(ToAmount.ToUnit(amount.AmountSats)),
	//	ShiftId: shiftId,
	//}

	hestiaAvailable := hestia.Config{Shift: hestia.Available{Service: true}}
	shiftProp := hestia.Properties{FeePercentage: 15, Available: true}
	coinsRate := []obol.Rate{
		obol.Rate{Code: "USD", Name: "USD", Rate: 0.5},
		obol.Rate{Code: "MXN", Name: "MXN", Rate: 0.3},
	}
	coinsConfig := []hestia.Coin{
		hestia.Coin{Ticker: "BTC", Shift: shiftProp},
		hestia.Coin{Ticker: "POLIS", Shift: shiftProp},
		hestia.Coin{Ticker: "XSG", Shift: shiftProp},
	}

	preparedShift := models.PrepareShiftInfoV2{
		FromCoin:   prepareData.FromCoin,
		Payment:    payment,
		ToCoin:     prepareData.ToCoin,
		ToAddress:  prepareData.ToAddress,
		ToAmount:   int64(ToAmount.ToUnit(amount.AmountSats)),
	}

	mockPlutusService := mocks.NewMockPlutusService(mockCtrl)
	mockHestiaService := mocks.NewMockHestiaService(mockCtrl)
	mockObolService := obolMocks.NewMockObolService(mockCtrl)

	shiftsMap := make(map[string]models.PrepareShiftInfoV2)

	testTyche := &controllers.TycheControllerV2{PrepareShifts: shiftsMap, Hestia: mockHestiaService, Plutus: mockPlutusService, Obol: mockObolService}

	gomock.InOrder(
		mockHestiaService.EXPECT().GetShiftStatus().Return(hestiaAvailable, nil),
		mockHestiaService.EXPECT().GetCoinsConfig().Return(coinsConfig, nil),
		mockObolService.EXPECT().GetCoin2CoinRatesWithAmount(gomock.Eq(prepareData.FromCoin), gomock.Eq(prepareData.ToCoin), gomock.Eq(amountHandler.String())).Return(coin2CoinResponse, nil),
		mockObolService.EXPECT().GetCoinRates(gomock.Eq(prepareData.FromCoin)).Return(coinsRate, nil),
		mockObolService.EXPECT().GetCoinRates(gomock.Eq("POLIS")).Return(coinsRate, nil),
		mockPlutusService.EXPECT().GetNewPaymentAddress(gomock.Eq(prepareData.FromCoin)).Return(paymentAddress, nil),
	)

	response, err := testTyche.PrepareV2(userId, payload, params)

	// Test returned response
	if err != nil {
		t.Fatal("Test returned response - returned error is not equal to nil")
	}

	field, ok := response.(*models.PrepareShiftResponseV2)
	if !ok {
		t.Fatal("Test returned response - returned response doesn't match")
	}
	t.Log("shift response v2 prepare", field)
	//if response != prepareResponse {
	//	fmt.Println(response)
	//	fmt.Println(prepareResponse)
	//	t.Fatal("Test returned response - returned response doesn't match")
	//}

	shift, e := testTyche.GetShiftFromMap(shiftId)

	// Test prepared shift stored in mapLock
	if e != nil {
		t.Fatal("Test returned response - get shift from map returns error")
	}

	// We don't care about these values, so we just make them equal
	preparedShift.ID = shift.ID
	preparedShift.Timestamp = shift.Timestamp

	if preparedShift.ID != shift.ID {
		t.Fatal("Test returned response - stored shift doesn't match")
	}
}*/

/*func TestStore(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	prepareData := models.PrepareShiftRequest{
		FromCoin:  "POLIS",
		Amount:    50,
		ToCoin:    "BTC",
		ToAddress: "dummy address",
	}
	payload, _ := json.Marshal(prepareData)

	uid := "123456789"
	params := models.Params{Coin: "POLIS"}
	paymentAddress := "123dummyAddress456"
	shiftId := "1234567890123"
	userId := "uid-random"

	payment := models.PaymentInfo{
		Address: paymentAddress,
		Amount:  prepareData.Amount,
	}
	// Empty fee payment because is from POLIS
	feePayment := models.PaymentInfo{}
	hestiaFeePayment := hestia.Payment{}
	shiftPayment := models.StoreShift{}
	emptyPreparedShift := models.PrepareShiftInfo{}

	_ = json.Unmarshal(payload, &shiftPayment)

	amountHandler := amount.AmountType(prepareData.Amount)
	rateAmountHandler, _ := amount.NewAmount(45.324998)
	ToAmount, _ := amount.NewAmount(amountHandler.ToNormalUnit() / rateAmountHandler.ToNormalUnit())

	preparedShift := models.PrepareShiftInfo{
		ID:         "1234567890",
		FromCoin:   prepareData.FromCoin,
		Payment:    payment,
		FeePayment: feePayment,
		ToCoin:     prepareData.ToCoin,
		ToAddress:  prepareData.ToAddress,
		ToAmount:   int64(ToAmount.ToUnit(amount.AmountSats)),
		Timestamp:  1234567890,
	}

	shift := hestia.Shift{
		ID:        preparedShift.ID,
		UID:       uid,
		Status:    hestia.GetShiftStatusString(hestia.ShiftStatusPending),
		Timestamp: time.Now().Unix(),
		Payment: hestia.Payment{
			Address:       preparedShift.Payment.Address,
			Amount:        preparedShift.Payment.Amount,
			Coin:          preparedShift.FromCoin,
			Txid:          "",
			Confirmations: 0,
		},
		FeePayment:     hestiaFeePayment,
		ToCoin:         preparedShift.ToCoin,
		ToAmount:       preparedShift.ToAmount,
		ToAddress:      preparedShift.ToAddress,
		RefundAddr:     shiftPayment.RefundAddr,
		PaymentProof:   "",
		ProofTimestamp: 0,
	}

	mockHestiaService := mocks.NewMockHestiaService(mockCtrl)
	mockPlutusService := mocks.NewMockPlutusService(mockCtrl)

	shiftsMap := make(map[string]models.PrepareShiftInfo)
	shiftsMap[uid] = preparedShift

	testTyche := &controllers.TycheControllerV2{PrepareShifts: shiftsMap, Hestia: mockHestiaService, Plutus: mockPlutusService}

	gomock.InOrder(
		mockHestiaService.EXPECT().UpdateShift(gomock.Eq(shift)).Return(shiftId, nil),
		mockPlutusService.EXPECT().ValidateRawTx(gomock.Any()).AnyTimes().Return(true, nil),
	)

	idResponse, err := testTyche.Store(uid, payload, params)

	// Test get shift from mapLock and store it
	if err != nil {
		t.Fatal("Test shift stored - Error is not equal to nil")
	}

	if idResponse != shiftId {
		t.Fatal("Test shift stored - Returned shift Id doesn't match")
	}

	prepShift, er := testTyche.GetShiftFromMap(uid)

	// Test shift get deleted from mapLock
	if prepShift != emptyPreparedShift {
		t.Fatal("Test shift stored - Shift wasn't deleted from mapLock")
	}

	if er == nil {
		t.Fatal("Test shift stored - Expected map error is equal to nil")
	}
}*/
