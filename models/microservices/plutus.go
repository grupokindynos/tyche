package microservices

//Plutus is the model for the response from the Plutus microservice
type Plutus struct {
	Data   string `json:"data"`
	Status int64  `json:"status"`
}

//PlutusAddress is the model for the response from the Plutus address microservice
type PlutusAddress struct {
	Address string `json:"address"`
}
