package eth

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// GethClient describes Ethereum client interface.
type GethClient interface {
	Accounts(ctx context.Context) ([]string, error)
	SendTransaction(ctx context.Context,
		from, to common.Address, amount *big.Int) (*string, error)
	NetworkID(ctx context.Context) (*big.Int, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	TransactionReceipt(ctx context.Context,
		txHash common.Hash) (*types.Receipt, error)
	BalanceAt(ctx context.Context, account common.Address,
		blockNumber *big.Int) (*big.Int, error)
	SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error)
}

// Client is an Ethereum JSON-RPC client.
type Client struct {
	rpcCli *rpc.Client
	ethCli *ethclient.Client
}

// SendTxArgs is an arguments to send transaction.
type SendTxArgs struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Gas      string `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Value    string `json:"value"`
}

// NewClient creates a new  Ethereum JSON-RPC client.
func NewClient(ctx context.Context, url string) (*Client, error) {
	rpcClient, err := rpc.DialContext(ctx, url)
	if err != nil {
		return nil, err
	}

	ethCli := ethclient.NewClient(rpcClient)

	return &Client{
		rpcCli: rpcClient,
		ethCli: ethCli,
	}, nil
}

// Close closes an Ethereum JSON-RPC client.
func (c *Client) Close() {
	c.Close()
}

// Accounts gets accounts from Geth node.
func (c *Client) Accounts(ctx context.Context) ([]string, error) {
	var result []string
	err := c.rpcCli.CallContext(ctx, &result, "personal_listAccounts")
	return result, err
}

// SendTransaction sends a transaction through Geth node.
// Account must be unlocked.
func (c *Client) SendTransaction(ctx context.Context,
	from, to common.Address, amount *big.Int) (result *string, err error) {
	gas := uint64(34000)

	gasPrice, err := c.ethCli.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	args := SendTxArgs{
		From:     from.Hex(),
		To:       to.Hex(),
		Gas:      hexutil.EncodeUint64(gas),
		GasPrice: hexutil.EncodeBig(gasPrice),
		Value:    hexutil.EncodeBig(amount),
	}

	err = c.rpcCli.CallContext(ctx, &result,
		"eth_sendTransaction", args)
	return result, err
}

func (c *Client) NetworkID(ctx context.Context) (*big.Int, error) {
	return c.ethCli.NetworkID(ctx)
}

func (c *Client) BlockByNumber(ctx context.Context,
	number *big.Int) (*types.Block, error) {
	return c.ethCli.BlockByNumber(ctx, number)
}

func (c *Client) TransactionReceipt(ctx context.Context,
	txHash common.Hash) (*types.Receipt, error) {
	return c.ethCli.TransactionReceipt(ctx, txHash)
}

func (c *Client) BalanceAt(ctx context.Context, account common.Address,
	blockNumber *big.Int) (*big.Int, error) {
	return c.ethCli.BalanceAt(ctx, account, blockNumber)
}

func (c *Client) SyncProgress(
	ctx context.Context) (*ethereum.SyncProgress, error) {
	return c.ethCli.SyncProgress(ctx)
}
