//go:generate reform

package data

import (
	"encoding/json"
	"time"
)

type TxStatus int

const (
	Pending = iota
	Mined
	Ignored
)

//reform:accounts
type Account struct {
	ID        string `json:"id" reform:"id,pk"`
	Balance   string `json:"balance" reform:"balance"`
	PublicKey string `json:"publicKey" reform:"public_key"`
	// The field is not exported to JSON.
	PrivateKey *json.RawMessage `reform:"private_key"`
	// The field is not exported to JSON.
	Password   *string    `reform:"password"`
	LastUpdate *time.Time `json:"lastUpdate" reform:"last_update"`
}

//reform:transactions
type Transaction struct {
	ID           string  `json:"id" reform:"id,pk"`
	Hash         string  `json:"hash" reform:"hash"`
	From         string  `json:"from" reform:"from"`
	To           *string `json:"to" reform:"to"`
	Amount       string  `json:"amount" reform:"amount"`
	Status       *uint64 `json:"status" reform:"status"`
	ReceiptBlock *uint64 `json:"receiptBlock" reform:"receipt_block"`
	Timestamp    *uint64 `json:"timestamp" reform:"timestamp"`
}

//reform:settings
type Setting struct {
	Key   string `json:"key" reform:"key,pk"`
	Value string `json:"value" reform:"value"`
}
