package eth

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var gas = hexutil.EncodeUint64(30400)

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

// EthCli returns an Ethereum RPC API client.
func (c *Client) EthCli() *ethclient.Client {
	return c.ethCli
}

// RPCCli returns a RPC client.
func (c *Client) RPCCli() *rpc.Client {
	return c.rpcCli
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
	from, to common.Address, amount *big.Int) (*string, error) {
	var result *string

	gasPrice, err := c.ethCli.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	args := SendTxArgs{
		From:     from.Hex(),
		To:       to.Hex(),
		Gas:      gas,
		GasPrice: hexutil.EncodeBig(gasPrice),
		Value:    hexutil.EncodeBig(amount),
	}

	err = c.rpcCli.CallContext(ctx, &result, "eth_sendTransaction", args)
	return result, err
}
