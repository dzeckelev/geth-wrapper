package proc

import (
	"context"
	"log"
	"math/big"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gopkg.in/reform.v1"

	"github.com/dzeckelev/geth-wrapper/config"
	"github.com/dzeckelev/geth-wrapper/data"
	"github.com/dzeckelev/geth-wrapper/eth"
	"github.com/dzeckelev/geth-wrapper/gen"
)

// Scheduler is a task scheduler.
type Scheduler struct {
	netID    *big.Int
	cfg      *config.Config
	cancel   context.CancelFunc
	ctx      context.Context
	eth      *eth.Client
	db       *reform.DB
	updBalCh chan []string

	mtx          sync.RWMutex
	lastBlockNum *big.Int

	wg sync.WaitGroup
}

// NewScheduler creates a new task scheduler.
func NewScheduler(ctx context.Context, networkID *big.Int, cfg *config.Config,
	database *reform.DB, ethClient *eth.Client) (*Scheduler, error) {
	ctx, cancel := context.WithCancel(ctx)

	return &Scheduler{
		netID:    networkID,
		cfg:      cfg,
		cancel:   cancel,
		ctx:      ctx,
		db:       database,
		eth:      ethClient,
		updBalCh: make(chan []string, 1000),
	}, nil
}

// Start starts a task scheduler.
func (s *Scheduler) Start() error {
	last, err := s.eth.BlockByNumber(s.ctx, nil)
	if err != nil {
		return err
	}

	s.lastBlockNum = last.Number()
	s.wg.Add(4)

	go s.updateLastBlock()
	go s.updateTransactions()
	go s.collect()
	go s.updateAccounts()

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

		if err := s.collectTxs(); err != nil {
			log.Printf("failed to collect data: %s", err)
		}
		time.Sleep(time.Millisecond * time.Duration(s.cfg.Proc.CollectPause))
	}
}

func (s *Scheduler) getAccounts() (map[common.Address]struct{}, error) {
	acs, err := s.eth.Accounts(s.ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[common.Address]struct{})
	for k := range acs {
		m[common.HexToAddress(acs[k])] = struct{}{}
	}
	return m, nil
}

func targetAccounts(accounts map[common.Address]struct{},
	from, to common.Address) (result []string) {
	if _, ok := accounts[from]; ok {
		result = append(result, strings.ToLower(from.String()))
	}

	if _, ok := accounts[to]; ok {
		result = append(result, strings.ToLower(to.String()))
	}

	return result
}

func (s *Scheduler) startBlock() (*big.Int, error) {
	startBlock, err := s.lastBlockFromDB()
	if err != nil {
		return nil, err
	}

	if s.cfg.Eth.StartBlock > startBlock {
		startBlock = s.cfg.Eth.StartBlock
	}

	return new(big.Int).SetUint64(startBlock), nil
}

func (s *Scheduler) collectTxs() error {
	current, err := s.startBlock()
	if err != nil {
		return err
	}

	increaseBlockNum := func() {
		current = new(big.Int).Add(current, big.NewInt(1))
	}

	signer := types.NewEIP155Signer(s.netID)

	for {
		s.mtx.RLock()
		lastBlock := s.lastBlockNum
		s.mtx.RUnlock()

		if current.Cmp(lastBlock) > 0 {
			time.Sleep(time.Millisecond *
				time.Duration(s.cfg.Proc.CollectPause))

			log.Printf("current block %s, lastBlock block %s",
				current.String(), lastBlock.String())
			continue
		}

		accounts, err := s.getAccounts()
		if err != nil {
			return err
		}

		block, err := s.eth.BlockByNumber(s.ctx, current)
		if err != nil {
			return err
		}

		txs := block.Transactions()

		log.Printf("block: %d, transactions: %d",
			block.Number(), len(txs))

		if len(txs) == 0 {
			increaseBlockNum()
			if err := s.updateLastBlockSetting(current); err != nil {
				return err
			}
			continue
		}

		var confirm uint64
		if lastBlock.Cmp(block.Number()) >= 0 {
			confirm = new(big.Int).Sub(lastBlock,
				block.Number()).Uint64()
		}

		var accountsToUpd []string

		concurrency := runtime.NumCPU()
		sem := make(chan struct{}, concurrency)

		resultMtx := sync.Mutex{}
		var result []*data.Transaction

		for k := range txs {
			sem <- struct{}{}

			go func(k int) {
				defer func() { <-sem }()

				hash := txs[k].Hash().String()
				from, err := signer.Sender(txs[k])
				if err != nil {
					log.Printf("invalid transaction %s: %s",
						txs[k].Hash().String(), err)
					return
				}

				tr, err := s.eth.TransactionReceipt(s.ctx,
					txs[k].Hash())
				if err != nil {
					log.Printf("failed to get transaction receipt: %s", err)
					return
				}

				var to common.Address

				if txs[k].To() != nil {
					to = *txs[k].To()
				} else {
					to = tr.ContractAddress
				}

				acc := targetAccounts(accounts, from, to)

				if len(acc) == 0 {
					return
				}

				tx := &data.Transaction{
					ID:     gen.NewUUID(),
					Hash:   hash,
					From:   strings.ToLower(from.String()),
					To:     strings.ToLower(to.String()),
					Amount: txs[k].Value().String(),
					Block: pointer.ToUint64(
						block.Number().Uint64()),
					Timestamp: pointer.ToUint64(
						block.Time().Uint64()),
					Confirmations: confirm,
				}

				switch tr.Status {
				case types.ReceiptStatusFailed:
					tx.Status = pointer.ToString(data.TxFailed)
				case types.ReceiptStatusSuccessful:
					tx.Status = pointer.ToString(data.TxSuccessful)
				default:
					log.Printf("unknown status transaction status: %s", hash)
					return
				}

				resultMtx.Lock()
				accountsToUpd = append(accountsToUpd, acc...)
				result = append(result, tx)
				resultMtx.Unlock()
			}(k)
		}

		for i := 0; i < cap(sem); i++ {
			sem <- struct{}{}
		}

		select {
		case s.updBalCh <- accountsToUpd:
		// TODO: hardcoded timeout
		case <-time.After(time.Second):
		}

		if err := s.db.InTransaction(func(t *reform.TX) error {
			for k := range result {
				err := t.Insert(result[k])
				if err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			log.Printf("failed to processed block %s, error: %v",
				block.Number(), err)
			continue
		}

		increaseBlockNum()
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

			if tx.Block != nil && lastBlock > *tx.Block {
				confirm := lastBlock - *tx.Block

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

	tic := time.NewTicker(time.Millisecond *
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

func (s *Scheduler) updateAccounts() {
	defer s.wg.Done()

	update := func(accounts []string) {
		for k := range accounts {
			balance, err := s.eth.BalanceAt(
				s.ctx, common.HexToAddress(accounts[k]), nil)
			if err != nil {
				log.Printf("failed to get account balance: %s", err)
				return
			}

			account := &data.Account{}
			if err := s.db.FindOneTo(account,
				"public_key", accounts[k]); err != nil {
				if err != reform.ErrNoRows {
					log.Printf("failed to find account: %s", err)
					return
				}

				account.ID = gen.NewUUID()
				account.PublicKey = accounts[k]
			}

			account.Balance = balance.String()

			if err := s.db.Save(account); err != nil {
				log.Printf("failed to save account: %s", err)
				return
			}
		}
	}

	accounts, err := s.eth.Accounts(s.ctx)
	if err != nil {
		log.Printf("failed to get accounts: %s", err)
		return
	}
	update(accounts)

	for {
		select {
		case accounts, ok := <-s.updBalCh:
			if !ok {
				return
			}
			update(accounts)
		case <-s.ctx.Done():
			return
		}
	}
}
