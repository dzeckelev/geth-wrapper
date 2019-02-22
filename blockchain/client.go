package blockchain

import (
	"context"

	"github.com/dzeckelev/geth-wrapper/config"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// NewClient creates a new JSON-RPC Ethereum client.
func NewClient(ctx context.Context,
	cfg *config.Eth) (*ethclient.Client, error) {
	rpcClient, err := rpc.DialContext(ctx, cfg.NodeURL)
	if err != nil {
		return nil, err
	}

	return ethclient.NewClient(rpcClient), nil
}

// Close closes JSON-RPC Ethereum client.
func Close(client *ethclient.Client) {
	client.Close()
}
