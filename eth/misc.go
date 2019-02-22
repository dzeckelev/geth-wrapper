package eth

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

// WaitSync waits for the synchronization of the Geth node.
func WaitSync(ctx context.Context, client *ethclient.Client,
	pauseTime time.Duration) error {
	for {
		progress, err := client.SyncProgress(ctx)
		if err != nil {
			return err
		}

		if progress == nil {
			block, err := client.BlockByNumber(ctx, nil)
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

		time.Sleep(pauseTime)
	}

	return nil
}
