package microservices

//Plutus is the model for the response from the Plutus microservice
type Plutus struct {
	Data   interface{} `json:"data"`
	Status int64       `json:"status"`
}
