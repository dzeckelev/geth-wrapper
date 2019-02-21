package service

import (
	"context"
	"github.com/AlekSi/pointer"
	"github.com/dzeckelev/geth-wrapper/common"
	"github.com/dzeckelev/geth-wrapper/config"
	"github.com/dzeckelev/geth-wrapper/data"
	"github.com/dzeckelev/geth-wrapper/eth"
	common2 "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gopkg.in/reform.v1"
	"log"
	"math/big"
	"strconv"
	"time"
)

type Service struct {
	ethBack *eth.Backend
	db      *reform.DB
}

func NewService(ctx context.Context, cfg *config.Config,
	database *reform.DB) (*Service, error) {
	ethBackend, err := eth.NewBackend(ctx, cfg.Eth)
	if err != nil {
		return nil, err
	}

	return &Service{
		db:      database,
		ethBack: ethBackend,
	}, nil
}

func (s *Service) Start() error {
	if err := s.ethBack.WaitSync(); err != nil {
		return err
	}

	go func() {
		for {
			if err := s.getTransactions(); err != nil {
				log.Printf("failed to get transactions: %s", err)
			}
			// TODO hardcoded pause.
			time.Sleep(time.Second * 15)
		}
	}()

	// TODO
	time.Sleep(time.Second * 100)

	return nil
}

func (s *Service) Stop() {
	s.ethBack.Stop()
}

func (s *Service) lastBlock() (uint64, error) {
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

func (s *Service) updateLastBlock(block *big.Int) error {
	setting := &data.Setting{
		Key:   "lastBlock",
		Value: block.String(),
	}

	return s.db.Save(setting)
}

func (s *Service) getTransactions() error {
	startBlock, err := s.lastBlock()
	if err != nil {
		return err
	}

	current := new(big.Int).SetUint64(startBlock)
	signer := types.NewEIP155Signer(s.ethBack.NetID())

	incBlockNum := func() {
		current = new(big.Int).Add(current, big.NewInt(1))
	}

	for {
		time.Sleep(time.Second / 50)

		block, err := s.ethBack.BlockByNumber(current)
		if err != nil {
			return err
		}

		var result struct {
			txs []*data.Transaction
			acc []*data.Account
		}

		txs := block.Transactions()
		log.Printf("block: %d, transactions: %d", block.Number(), len(txs))

		if len(txs) == 0 {
			incBlockNum()
			if err := s.updateLastBlock(current); err != nil {
				return err
			}
			continue
		}

		for k := range txs {
			hash := txs[k].Hash().String()
			from, err := signer.Sender(txs[k])
			if err != nil {
				log.Printf("invalid transaction %s", txs[k].Hash().String())
				continue
			}

			tx := &data.Transaction{
				ID:           common.NewUUID(),
				Hash:         hash,
				From:         from.String(),
				Amount:       txs[k].Value().String(),
				ReceiptBlock: pointer.ToUint64(block.Number().Uint64()),
				Timestamp:    pointer.ToUint64(block.Time().Uint64()),
			}

			if txs[k].To() != nil {
				tx.To = pointer.ToString(txs[k].To().String())
			}

			tr, err := s.ethBack.TransactionReceipt(txs[k].Hash())
			if err != nil {
				return err
			}

			tx.Status = pointer.ToUint64(tr.Status)
			result.txs = append(result.txs, tx)

			updateAccount := func(addr common2.Address) error {
				balance, err := s.ethBack.Balance(addr, block.Number())
				if err != nil {
					return err
				}

				acc := &data.Account{}
				if err := s.db.FindOneTo(acc, "public_key",
					addr.String()); err != nil {
					if err == reform.ErrNoRows {
						acc.ID = common.NewUUID()
						acc.PublicKey = addr.String()
					} else {
						return err
					}
				}

				acc.Balance = balance.String()
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
			log.Printf("failed to processed block %s, error: %v",
				block.Number(), err)
			continue
		}

		incBlockNum()
		if err := s.updateLastBlock(current); err != nil {
			return err
		}
	}
}
