package api

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/dzeckelev/geth-wrapper/data"
	"github.com/dzeckelev/geth-wrapper/eth"
	"github.com/dzeckelev/geth-wrapper/gen"
)

// Handler is an API RPC handler.
type Handler struct {
	database  *reform.DB
	ethClient eth.Client
	networkID *big.Int

	// Mutex is needed to synchronize requests.
	mtx sync.Mutex
}

// GetLastResult is result of GetLast method.
type GetLastResult struct {
	Hash    string
	Date    string
	Address string
	// In Wei (1 ETH = 10^18 Wei)
	// String type because it can go beyond uint64.
	Amount        string
	Confirmations uint64
}

// NewHandler creates a new handler.
func NewHandler(networkID *big.Int,
	database *reform.DB, ethClient eth.Client) *Handler {
	return &Handler{
		networkID: networkID,
		database:  database,
		ethClient: ethClient,
	}
}

// GetLast returns latest transactions.
func (h *Handler) GetLast(limit uint64) ([]GetLastResult, error) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	query := `WHERE transactions."to" 
				 IN (SELECT public_key FROM accounts) 
				AND (confirmations < %s OR NOT marked)
			  ORDER BY block ASC LIMIT %s`

	tail := fmt.Sprintf(query,
		h.database.Placeholder(1), h.database.Placeholder(2))

	items, err := h.database.SelectAllFrom(
		data.TransactionTable, tail, 3, limit)
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

		tx.Marked = true
		if err := h.database.Save(&tx); err != nil {
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

	if err := h.database.Save(output); err != nil {
		return nil, err
	}

	return hash, nil
}
