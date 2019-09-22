package microservices

//Payment is the model for the payment object
type Payment struct {
	Address       string `bson:"address" json:"address"`
	Amount        string `bson:"amount" json:"amount"`
	Coin          string `bson:"coin" json:"coin"`
	Txid          string `bson:"txid" json:"txid"`
	Confirmations string `bson:"confirmations" json:"confirmations"`
}

//Shift is the model for the shift object stored in the database
type Shift struct {
	ID         string  `bson:"id" json:"id"`
	UID        string  `bson:"uid" json:"uid"`
	Status     string  `bson:"status" json:"status"`
	Timestamp  string  `bson:"timestamp" json:"timestamp"`
	Payment    Payment `bson:"payment" json:"payment"`
	Conversion Payment `bson:"conversion" json:"conversion"`
}
