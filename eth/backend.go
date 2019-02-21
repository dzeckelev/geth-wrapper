package eth

import (
	"context"
	"github.com/dzeckelev/geth-wrapper/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"log"
	"math/big"
	"time"
)

var syncSleepTime = time.Second * 5

type Backend struct {
	client     *ethclient.Client
	queryPause time.Duration
	ctx        context.Context
	netID      *big.Int
}

func NewBackend(ctx context.Context, cfg *config.Eth) (*Backend, error) {
	rpcClient, err := rpc.DialContext(ctx, cfg.NodeURL)
	if err != nil {
		return nil, err
	}

	client := ethclient.NewClient(rpcClient)

	netID, err := client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	return &Backend{
		ctx:    ctx,
		client: client,
		netID:  netID,
	}, nil
}

func (b *Backend) waitSync() error {
	for {
		progress, err := b.client.SyncProgress(b.ctx)
		if err != nil {
			return err
		}

		if progress == nil {
			block, err := b.client.BlockByNumber(b.ctx, nil)
			if err != nil {
				return err
			}

			if block.Number().Cmp(big.NewInt(0)) > 0 {
				break
			}

			continue
		}

		log.Printf("SyncProgress - StartingBlock: %d, CurrentBlock: %d,"+
			" HighestBlock: %d, PulledStates; %d, KnownStates %d",
			progress.StartingBlock, progress.CurrentBlock,
			progress.HighestBlock, progress.PulledStates, progress.KnownStates)

		time.Sleep(syncSleepTime)
	}

	return nil
}

func (b *Backend) WaitSync() error {
	if err := b.waitSync(); err != nil {
		return err
	}

	log.Println("ethereum node synchronized")

	return nil
}

func (b *Backend) Stop() {
	b.client.Close()
}

func (b *Backend) BlockByNumber(number *big.Int) (*types.Block, error) {
	return b.client.BlockByNumber(b.ctx, number)
}

func (b *Backend) TransactionReceipt(
	hash common.Hash) (*types.Receipt, error) {
	return b.client.TransactionReceipt(b.ctx, hash)
}

func (b *Backend) NetID() *big.Int {
	return b.netID
}

func (b *Backend) Balance(
	account common.Address, block *big.Int) (*big.Int, error) {
	return b.client.BalanceAt(b.ctx, account, block)
}
