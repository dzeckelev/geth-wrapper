package proc

import (
	"context"
	"log"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/dzeckelev/geth-wrapper/config"
	"github.com/dzeckelev/geth-wrapper/data"
	"github.com/dzeckelev/geth-wrapper/gen"

	"github.com/AlekSi/pointer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"gopkg.in/reform.v1"
)

// Scheduler is a task scheduler.
type Scheduler struct {
	netID  *big.Int
	cfg    *config.Config
	cancel context.CancelFunc
	ctx    context.Context
	eth    *ethclient.Client
	db     *reform.DB

	mtx          sync.RWMutex
	lastBlockNum *big.Int

	wg sync.WaitGroup
}

// NewScheduler creates a new task scheduler.
func NewScheduler(ctx context.Context, networkID *big.Int, cfg *config.Config,
	database *reform.DB, ethClient *ethclient.Client) (*Scheduler, error) {
	ctx, cancel := context.WithCancel(ctx)

	return &Scheduler{
		netID:  networkID,
		cfg:    cfg,
		cancel: cancel,
		ctx:    ctx,
		db:     database,
		eth:    ethClient,
	}, nil
}

// Start starts a task scheduler.
func (s *Scheduler) Start() error {
	last, err := s.eth.BlockByNumber(s.ctx, nil)
	if err != nil {
		return err
	}

	s.lastBlockNum = last.Number()
	s.wg.Add(3)

	go s.updateLastBlock()
	go s.updateTransactions()
	go s.collect()

	return nil
}

// Close closes a task scheduler.
func (s *Scheduler) Close() {
	s.cancel()

	s.wg.Wait()
}

func (s *Scheduler) updateLastBlock() {
	defer s.wg.Done()

	tic := time.NewTicker(time.Millisecond *
		time.Duration(s.cfg.Proc.UpdateLastBlockPause))

	for {
		select {
		case <-tic.C:
			block, err := s.eth.BlockByNumber(s.ctx, nil)
			if err != nil {
				log.Printf("failed to get last block: %s", err)
				continue
			}

			s.mtx.Lock()
			s.lastBlockNum = block.Number()
			s.mtx.Unlock()
		case <-s.ctx.Done():
			tic.Stop()
			return
		}
	}
}

func (s *Scheduler) lastBlockFromDB() (uint64, error) {
	lastBlockSetting := &data.Setting{}
	err := s.db.FindByPrimaryKeyTo(lastBlockSetting, "lastBlock")
	if err != nil {
		if err != reform.ErrNoRows {
			return 0, err
		}
		return 0, nil
	}
	return strconv.ParseUint(lastBlockSetting.Value, 10, 64)
}

func (s *Scheduler) updateLastBlockSetting(block *big.Int) error {
	setting := &data.Setting{
		Key:   "lastBlock",
		Value: block.String(),
	}

	return s.db.Save(setting)
}

func (s *Scheduler) collect() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		if err := s.collectData(); err != nil {
			log.Printf("failed to collect data: %s", err)
		}
		time.Sleep(time.Millisecond * time.Duration(s.cfg.Proc.CollectPause))
	}
}

func (s *Scheduler) collectData() error {
	startBlock, err := s.lastBlockFromDB()
	if err != nil {
		return err
	}

	if s.cfg.Eth.StartBlock > startBlock {
		startBlock = s.cfg.Eth.StartBlock
	}

	current := new(big.Int).SetUint64(startBlock)
	signer := types.NewEIP155Signer(s.netID)

	incCurrentBlockNum := func() {
		current = new(big.Int).Add(current, big.NewInt(1))
	}

	for {
		s.mtx.RLock()
		lastBlock := s.lastBlockNum
		s.mtx.RUnlock()

		if current.Cmp(lastBlock) > 0 {
			time.Sleep(time.Millisecond *
				time.Duration(s.cfg.Proc.CollectPause))
		}

		currentBlock, err := s.eth.BlockByNumber(s.ctx, current)
		if err != nil {
			return err
		}

		var result struct {
			txs []*data.Transaction
			acc []*data.Account
		}

		txs := currentBlock.Transactions()
		log.Printf("block: %d, transactions: %d",
			currentBlock.Number(), len(txs))

		if len(txs) == 0 {
			incCurrentBlockNum()
			if err := s.updateLastBlockSetting(current); err != nil {
				return err
			}
			continue
		}

		var confirm uint64

		if lastBlock.Cmp(currentBlock.Number()) >= 0 {
			confirm = new(big.Int).Sub(lastBlock,
				currentBlock.Number()).Uint64()
		}

		for k := range txs {
			hash := txs[k].Hash().String()
			from, err := signer.Sender(txs[k])
			if err != nil {
				log.Printf("invalid transaction %s: %s",
					txs[k].Hash().String(), err)
				continue
			}

			tx := &data.Transaction{
				ID:            gen.NewUUID(),
				Hash:          hash,
				From:          from.String(),
				Amount:        txs[k].Value().String(),
				ReceiptBlock:  pointer.ToUint64(currentBlock.Number().Uint64()),
				Timestamp:     pointer.ToUint64(currentBlock.Time().Uint64()),
				Confirmations: confirm,
			}

			tr, err := s.eth.TransactionReceipt(s.ctx, txs[k].Hash())
			if err != nil {
				return err
			}

			switch tr.Status {
			case types.ReceiptStatusFailed:
				tx.Status = pointer.ToString(data.TxFailed)
			case types.ReceiptStatusSuccessful:
				tx.Status = pointer.ToString(data.TxSuccessful)
			default:
				return errors.New("unknown status")
			}

			if txs[k].To() == nil {
				tx.To = tr.ContractAddress.String()
			} else {
				tx.Confirmations = confirm
				tx.To = txs[k].To().String()
			}

			result.txs = append(result.txs, tx)

			updateAccount := func(addr common.Address) error {
				balance, err := s.eth.BalanceAt(
					s.ctx, addr, currentBlock.Number())
				if err != nil {
					return err
				}

				acc := &data.Account{}
				if err := s.db.FindOneTo(acc, "public_key",
					addr.String()); err != nil {
					if err == reform.ErrNoRows {
						acc.ID = gen.NewUUID()
						acc.PublicKey = addr.String()
						acc.LastBlock = currentBlock.Number().Uint64()
					} else {
						return err
					}
				}

				if currentBlock.Number().Uint64() >= acc.LastBlock {
					acc.Balance = balance.String()
				}

				acc.LastUpdate = pointer.ToTime(time.Now())
				result.acc = append(result.acc, acc)
				return nil
			}

			if txs[k].To() != nil {
				if err := updateAccount(*txs[k].To()); err != nil {
					return err
				}
			}

			if err := updateAccount(from); err != nil {
				return err
			}
		}

		if err := s.db.InTransaction(func(t *reform.TX) error {
			for k := range result.txs {
				err := t.Insert(result.txs[k])
				if err != nil {
					return err
				}
			}

			for k := range result.txs {
				err := t.Save(result.acc[k])
				if err != nil {
					return err
				}
			}

			return nil
		}); err != nil {
			log.Printf("failed to processed currentBlock %s, error: %v",
				currentBlock.Number(), err)
			continue
		}

		incCurrentBlockNum()
		if err := s.updateLastBlockSetting(current); err != nil {
			return err
		}
	}
}

func (s *Scheduler) updateTransactions() {
	defer s.wg.Done()

	update := func() error {
		items, err := s.db.SelectAllFrom(data.TransactionTable,
			"WHERE confirmations <= $1", 6)
		if err != nil {
			return err
		}

		s.mtx.RLock()
		lastBlock := s.lastBlockNum.Uint64()
		s.mtx.RUnlock()

		for k := range items {
			tx := *items[k].(*data.Transaction)

			if tx.ReceiptBlock != nil && lastBlock > *tx.ReceiptBlock {
				confirm := lastBlock - *tx.ReceiptBlock

				if confirm > tx.Confirmations {
					tx.Confirmations = confirm
				}
			}

			if err := s.db.Save(&tx); err != nil {
				return err
			}
		}

		return nil
	}

	tic := time.NewTicker(time.Second *
		time.Duration(s.cfg.Proc.UpdateTransactionsPause))
	for {
		select {
		case <-tic.C:
			if err := update(); err != nil {
				log.Printf("failed to update transactions: %s", err)
			}
		case <-s.ctx.Done():
			tic.Stop()
			return
		}
	}
}
