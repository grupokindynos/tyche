package models

type BlockbookTxInfo struct {
	Txid     string `json:"txid"`
	Version  int    `json:"version"`
	LockTime int64  `json:"lockTime"`
	Vin      []struct {
		Sequence int64  `json:"sequence"`
		N        int    `json:"n"`
		Coinbase string `json:"coinbase"`
	} `json:"vin"`
	Vout []struct {
		Value     string   `json:"value"`
		N         int      `json:"n"`
		Hex       string   `json:"hex"`
		Addresses []string `json:"addresses"`
	} `json:"vout"`
	BlockHash     string `json:"blockHash"`
	BlockHeight   int    `json:"blockHeight"`
	Confirmations int    `json:"confirmations"`
	BlockTime     int    `json:"blockTime"`
	Value         string `json:"value"`
	ValueIn       string `json:"valueIn"`
	Fees          string `json:"fees"`
	Hex           string `json:"hex"`
}

type BlockbookBroadcastResponse struct {
	Result string `json:"result"`
	Error  string `json:"error"`
}
