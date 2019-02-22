package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/dzeckelev/geth-wrapper/data"
	"github.com/dzeckelev/geth-wrapper/gen"

	"github.com/AlekSi/pointer"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

const gaslimit = 21000

// Handler is an API RPC handler.
type Handler struct {
	netID     *big.Int
	db        *reform.DB
	ethClient *ethclient.Client
}

// GetLastResult is result of GetLast method.
type GetLastResult struct {
	Date    string
	Address string
	// In Wai (1 ETH = 10^18 Wai)
	// String type because it can go beyond uint64.
	Amount        string
	Confirmations uint64
}

// NewHandler creates a new handler.
func NewHandler(networkID *big.Int,
	db *reform.DB, ethClient *ethclient.Client) *Handler {
	return &Handler{
		netID:     networkID,
		db:        db,
		ethClient: ethClient,
	}
}

// GetLast returns latest transactions.
// Example: curl -X POST -H "Content-Type: application/json" --data '{"method": "api_getLast", "params": [100], "id": 100}' http://localhost:8081/http
func (h *Handler) GetLast(limit uint64) ([]GetLastResult, error) {
	tail := fmt.Sprintf("WHERE confirmations < %s OR NOT marked"+
		" ORDER BY receipt_block ASC LIMIT %s",
		h.db.Placeholder(1), h.db.Placeholder(2))

	items, err := h.db.SelectAllFrom(data.TransactionTable, tail, 3, limit)
	if err != nil {
		return nil, err
	}

	result := make([]GetLastResult, len(items))

	for k, item := range items {
		tx := *item.(*data.Transaction)
		tm := time.Unix(0, 0)

		if tx.Timestamp != nil {
			tm = time.Unix(int64(*tx.Timestamp), 0)
		}

		// TODO: It not effective.
		tx.Marked = true
		if err := h.db.Save(&tx); err != nil {
			return nil, err
		}

		result[k] = GetLastResult{
			Date:          tm.Format(time.RFC3339),
			Address:       tx.To,
			Amount:        tx.Amount,
			Confirmations: tx.Confirmations,
		}
	}

	return result, nil
}

// Accounts returns accounts for which there is a private key.
// Example: curl -X POST -H "Content-Type: application/json" --data '{"method": "api_accounts", "params": [100], "id": 100}' http://localhost:8081/http
func (h *Handler) Accounts(limit uint64) ([]string, error) {
	tail := fmt.Sprintf("WHERE private_key IS NOT NULL LIMIT %s",
		h.db.Placeholder(1))

	items, err := h.db.SelectAllFrom(data.AccountTable, tail, limit)
	if err != nil {
		return []string{}, err
	}

	result := make([]string, len(items))

	for k, item := range items {
		acc := *item.(*data.Account)
		result[k] = acc.PublicKey
	}

	return result, nil
}

// SendEth sends ETH to specific address. Password is required to decrypt
// a private key stored in the database.
// Example: curl -X POST -H "Content-Type: application/json" --data '{"method": "api_sendEth", "params": ["0xd1dffc3c0537d46cd65b10019d4216f9dcd7e114", "oxd6d39cd7672841789dc3afb97525984b6d31f796", "1000000000000", "password"], "id": 100}' http://localhost:8081/http
func (h *Handler) SendEth(from, to, amount, password string) (*string, error) {
	account := common.HexToAddress(from)

	acc := &data.Account{}
	if err := h.db.FindOneTo(acc, "public_key", account.String()); err != nil {
		return nil, err
	}

	if acc.PrivateKey == nil {
		return nil, errors.New("missing private key")
	}

	key, err := keystore.DecryptKey(*acc.PrivateKey, password)
	if err != nil {
		return nil, err
	}

	nonce, err := h.ethClient.NonceAt(context.Background(),
		common.HexToAddress(acc.PublicKey), nil)
	if err != nil {
		return nil, err
	}

	wai, success := new(big.Int).SetString(amount, 10)
	if !success {
		return nil, errors.New("wrong amount argument")
	}

	gasPrice, err := h.ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}

	tx := types.NewTransaction(nonce, common.HexToAddress(to),
		wai, gaslimit, gasPrice, nil)

	signer := types.NewEIP155Signer(h.netID)
	signedTx, err := types.SignTx(tx, signer, key.PrivateKey)
	if err != nil {
		return nil, err
	}

	if err := h.ethClient.SendTransaction(
		context.Background(), signedTx); err != nil {
		return nil, err
	}

	hash := signedTx.Hash().String()

	return &hash, nil
}

// ImportAccountFromJSON imports account from UTC JSON Keystore.
// Example: curl -X POST -H "Content-Type: application/json" --data '{"method": "api_importAccountFromJSON", "params": [{"address":"d6d39cd7672841789dc3afb97525984b6d31f796","crypto":{"cipher":"aes-128-ctr","ciphertext":"e879d3adc40a2ab366098fa052cc76a23c42724f22345139affb7b7d1db2a41e","cipherparams":{"iv":"64eec3a3e93df66a60d07b0588bff875"},"kdf":"scrypt","kdfparams":{"dklen":32,"n":4096,"p":6,"r":8,"salt":"9b63a93d65c648f2fcbc725aac35edde67a33d660ce9fd7a9f1973cf0cf17888"},"mac":"891ebc85d4a3a1115f379cbb2845ebf8e57771053abadaa923f8567da31b4d8b"},"id":"c4825836-2750-43e6-b7ad-241df1601c6a","version":3}, ""], "id": 100}' http://localhost:8081/http
func (h *Handler) ImportAccountFromJSON(
	jsonBlob json.RawMessage, password string) error {

	key, err := keystore.DecryptKey(
		jsonBlob, password)
	if err != nil {
		return err
	}

	publicKey := key.Address.String()

	if _, err := keystore.EncryptKey(key, password, keystore.StandardScryptN,
		keystore.StandardScryptP); err != nil {
		return err
	}

	acc := &data.Account{}
	if err := h.db.FindOneTo(acc,
		"public_key", publicKey); err != nil {
		if err == reform.ErrNoRows {
			acc.ID = gen.NewUUID()
			acc.PublicKey = publicKey
		} else {
			return err
		}
	}

	if acc.PrivateKey != nil {
		_, err := keystore.DecryptKey(
			*acc.PrivateKey, password)
		if err != nil {
			return err
		}
	}

	lastHeader, err := h.ethClient.BlockByNumber(context.Background(), nil)
	if err != nil {
		return err
	}

	balance, err := h.ethClient.BalanceAt(context.Background(),
		common.HexToAddress(publicKey), lastHeader.Number())
	if err != nil {
		return err
	}

	acc.LastBlock = lastHeader.NumberU64()
	acc.LastUpdate = pointer.ToTime(time.Now())
	acc.PrivateKey = &jsonBlob
	acc.Balance = balance.String()

	return h.db.Save(acc)
}
