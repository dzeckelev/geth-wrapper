package eth

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// MockClient is a mock for Ethereum client.
type MockClient struct {
	Acc     map[string]*bind.TransactOpts
	NetID   *big.Int
	Block   *types.Block
	Backend *backends.SimulatedBackend
}

// NewMockClient creates a new Ethereum client.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// Accounts is a mock for Accounts function.
func (c *MockClient) Accounts(
	ctx context.Context) (result []string, err error) {
	for k := range c.Acc {
		result = append(result, k)
	}

	return result, nil
}

// SendTransaction is a mock for SendTransaction function.
func (c *MockClient) SendTransaction(ctx context.Context,
	from, to common.Address, amount *big.Int) (result *string, err error) {
	gasLimit := uint64(4700000)

	acc := c.Acc[strings.ToLower(from.String())]

	nonce, err := c.Backend.NonceAt(context.Background(), acc.From, nil)
	if err != nil {
		return nil, err
	}

	gasPrice, err := c.Backend.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}

	rawTx := types.NewTransaction(nonce, to,
		amount, gasLimit, gasPrice, nil)

	signTx, err := acc.Signer(types.HomesteadSigner{}, acc.From, rawTx)
	if err := c.Backend.SendTransaction(ctx, signTx); err != nil {
		return nil, err
	}

	c.Backend.Commit()

	hash := strings.ToLower(signTx.Hash().String())
	return &hash, nil
}

// NetworkID is a mock for NetworkID function.
func (c *MockClient) NetworkID(ctx context.Context) (*big.Int, error) {
	return c.NetID, nil
}

// BlockByNumber is a mock for BlockByNumber function.
func (c *MockClient) BlockByNumber(ctx context.Context,
	number *big.Int) (*types.Block, error) {
	return c.Block, nil
}

// TransactionReceipt is a mock for TransactionReceipt function.
func (c *MockClient) TransactionReceipt(ctx context.Context,
	txHash common.Hash) (*types.Receipt, error) {
	return c.TransactionReceipt(ctx, txHash)
}

// BalanceAt is a mock for BalanceAt function.
func (c *MockClient) BalanceAt(ctx context.Context, account common.Address,
	blockNumber *big.Int) (*big.Int, error) {
	return c.BalanceAt(ctx, account, blockNumber)
}

// SyncProgress is a mock for SyncProgress function.
func (c *MockClient) SyncProgress(
	ctx context.Context) (*ethereum.SyncProgress, error) {
	return nil, nil
}

// NewTestBlock creates a new test block.
func NewTestBlock(number *big.Int, txs []*types.Transaction,
	trx []*types.Receipt) *types.Block {
	header := &types.Header{
		Number: number,
	}

	return types.NewBlock(header, txs, nil, trx)
}
