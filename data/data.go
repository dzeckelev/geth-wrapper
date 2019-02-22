//go:generate reform

package data

import (
	"encoding/json"
	"time"
)

// Transaction statuses.
const (
	TxFailed     = "failed"
	TxSuccessful = "successful"
)

// Account is an Ethereum account.
//reform:accounts
type Account struct {
	ID        string `json:"id" reform:"id,pk"`
	Balance   string `json:"balance" reform:"balance"`
	PublicKey string `json:"publicKey" reform:"public_key"`
	// The field is not exported to JSON.
	PrivateKey *json.RawMessage `reform:"private_key"`
	// The field is not exported to JSON.
	Password   *string    `reform:"password"`
	LastBlock  uint64     `json:"lastBlock" reform:"last_block"`
	LastUpdate *time.Time `json:"lastUpdate" reform:"last_update"`
}

// Transaction is an Ethereum transaction.
//reform:transactions
type Transaction struct {
	ID            string  `json:"id" reform:"id,pk"`
	Hash          string  `json:"hash" reform:"hash"`
	From          string  `json:"from" reform:"from"`
	To            string  `json:"to" reform:"to"`
	Amount        string  `json:"amount" reform:"amount"`
	Status        *string `json:"status" reform:"status"`
	ReceiptBlock  *uint64 `json:"receiptBlock" reform:"receipt_block"`
	Timestamp     *uint64 `json:"timestamp" reform:"timestamp"`
	Marked        bool    `json:"marked" reform:"marked"`
	Confirmations uint64  `json:"confirmations" reform:"confirmations"`
}

// Setting is an application setting.
//reform:settings
type Setting struct {
	Key   string `json:"key" reform:"key,pk"`
	Value string `json:"value" reform:"value"`
}
