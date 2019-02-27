package api

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/dzeckelev/geth-wrapper/data"
	"github.com/dzeckelev/geth-wrapper/eth"
	"github.com/dzeckelev/geth-wrapper/gen"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// Handler is an API RPC handler.
type Handler struct {
	mtx       sync.Mutex
	netID     *big.Int
	db        *reform.DB
	ethClient *eth.Client
}

// GetLastResult is result of GetLast method.
type GetLastResult struct {
	Hash    string
	Date    string
	Address string
	// In Wei (1 ETH = 10^18 Wai)
	// String type because it can go beyond uint64.
	Amount        string
	Confirmations uint64
}

// NewHandler creates a new handler.
func NewHandler(networkID *big.Int,
	db *reform.DB, ethClient *eth.Client) *Handler {
	return &Handler{
		netID:     networkID,
		db:        db,
		ethClient: ethClient,
	}
}

// GetLast returns latest transactions.
func (h *Handler) GetLast(limit uint64) ([]GetLastResult, error) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	tail := fmt.Sprintf(`WHERE transactions."to" 
		IN (SELECT public_key FROM accounts) 
		AND (confirmations < %s OR NOT marked) ORDER BY block ASC LIMIT %s`,
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
			Hash:          tx.Hash,
			Date:          tm.Format(time.RFC3339),
			Address:       tx.To,
			Amount:        tx.Amount,
			Confirmations: tx.Confirmations,
		}
	}

	return result, nil
}

// SendETH sends ETH to specific address.
func (h *Handler) SendETH(from, to, amount string) (*string, error) {
	if !common.IsHexAddress(from) {
		return nil, errors.New(`invalid "from" argument`)
	}

	if !common.IsHexAddress(from) {
		return nil, errors.New(`invalid "to" argument`)
	}

	val, success := new(big.Int).SetString(amount, 10)
	if !success {
		return nil, errors.New(`invalid "amount" argument`)
	}

	hash, err := h.ethClient.SendTransaction(context.Background(),
		common.HexToAddress(from), common.HexToAddress(to), val)
	if err != nil {
		return nil, err
	}

	output := &data.Output{
		ID:      gen.NewUUID(),
		Hash:    *hash,
		Account: strings.ToLower(from),
	}

	if err := h.db.Save(output); err != nil {
		return nil, err
	}

	return hash, nil
}
