package microservices

//Obol is the model for the response from the Obol microservice
type Obol struct {
	Data   map[string]float64 `json:"data"`
	Status int64              `json:"status"`
}
