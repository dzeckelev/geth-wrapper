//go:generate reform

package data

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
	Block         *uint64 `json:"block" reform:"block"`
	Timestamp     *uint64 `json:"timestamp" reform:"timestamp"`
	Marked        bool    `json:"marked" reform:"marked"`
	Confirmations uint64  `json:"confirmations" reform:"confirmations"`
}

// Output is an outgoing transaction.
//reform:outputs
type Output struct {
	ID      string `json:"id" reform:"id,pk"`
	Hash    string `json:"hash" reform:"hash"`
	Account string `json:"account" reform:"account"`
}

// Setting is an application setting.
//reform:settings
type Setting struct {
	Key   string `json:"key" reform:"key,pk"`
	Value string `json:"value" reform:"value"`
}
