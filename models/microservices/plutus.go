package microservices

//Plutus is the model for the response from the Plutus microservice
type Plutus struct {
	Data   interface{} `json:"data"`
	Status int64       `json:"status"`
}

type PlutusEncoded struct {
	Data   string `json:"data"`
	Status int64  `json:"status"`
}

//PlutusAddress is the model for the response from the Plutus address microservice
type PlutusAddress struct {
	Data string `json:"address"`
}

//Address is the model for the response from the hot-wallets address
type Address struct {
	Address string `json:"address"`
}

//PlutusBalance is the model for the response from the Plutus address microservice
type PlutusBalance struct {
	Data   Balance "json:data"
	Status int64   "json:status"
}

//Balance is the model for the hot-wallets balance
type Balance struct {
	Confirmed   float64 `json:"confirmed"`
	Unconfirmed float64 `json:"unconfirmed"`
}
