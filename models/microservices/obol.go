package microservices

//ObolSimple is the model for the response from the Obol simple call
type ObolSimple struct {
	Data   []map[string]interface{} `json:"data"`
	Status int64                    `json:"status"`
}

//ObolComplex is the model for the response from the Obol complex call
type ObolComplex struct {
	Data   float64 `json:"data"`
	Status int64   `json:"status"`
}
