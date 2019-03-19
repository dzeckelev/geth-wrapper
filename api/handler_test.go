package api_test

import (
	"database/sql"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"gopkg.in/reform.v1"

	"github.com/dzeckelev/geth-wrapper/api"
	"github.com/dzeckelev/geth-wrapper/data"
	"github.com/dzeckelev/geth-wrapper/db"
	"github.com/dzeckelev/geth-wrapper/eth"
	"github.com/dzeckelev/geth-wrapper/gen"
)

const gasLimit uint64 = 4700000

var (
	handler   *api.Handler
	dataBase  *reform.DB
	sqlMock   sqlmock.Sqlmock
	ethClient *eth.MockClient
)

func newEthClient() *eth.MockClient {
	key, _ := crypto.GenerateKey()
	opts := bind.NewKeyedTransactor(key)
	addr := strings.ToLower(opts.From.String())

	balance := "1000000000000000000000000000000000000000000000000000"
	b1 := new(big.Int)
	_, _ = fmt.Sscan(balance, b1)

	alloc := make(core.GenesisAlloc)
	alloc[opts.From] = core.GenesisAccount{Balance: b1}
	sim := backends.NewSimulatedBackend(alloc, gasLimit)

	return &eth.MockClient{
		Acc:     map[string]*bind.TransactOpts{addr: opts},
		NetID:   big.NewInt(4),
		Block:   nil,
		Backend: sim,
	}
}

func TestGetLast(t *testing.T) {
	limitArg := uint64(100)
	confirmationsArg := uint64(3)
	hash := "0x64e604787cbf194841e7b68d7cd28786f6c9a0a3ab9f8b0a0e87cb4387ab0107"
	from := "0xe7dc9fe68da458b54f648146a817126053eeef66"
	to := "0xa7dba6053a0d631177340e8061bc12f5009ba453"
	amount := strconv.Itoa(10000)
	status := pointer.ToString(data.TxSuccessful)
	block := pointer.ToUint64(123456)
	timestamp := pointer.ToUint64(777777)
	confirmations := uint64(1)

	tx := &data.Transaction{
		ID:            gen.NewUUID(),
		Hash:          hash,
		From:          from,
		To:            to,
		Amount:        amount,
		Status:        status,
		Block:         block,
		Timestamp:     timestamp,
		Confirmations: confirmations,
	}

	tx2 := &data.Transaction{
		ID:            gen.NewUUID(),
		Hash:          hash,
		From:          from,
		To:            to,
		Amount:        amount,
		Status:        status,
		Block:         block,
		Timestamp:     timestamp,
		Confirmations: confirmations,
	}

	txs := []*data.Transaction{tx, tx2}

	columns := []string{"o_id", "o_hash", "o_from", "o_to", "o_amount",
		"o_status", "o_block", "o_timestamp", "o_marked", "o_confirmations"}

	// Empty result.
	sqlMock.ExpectQuery(`SELECT (.+) FROM "transactions"`).
		WithArgs(confirmationsArg, limitArg).WillReturnRows(sqlmock.NewRows(columns))

	result, err := handler.GetLast(limitArg)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 0 {
		t.Fatal("result must be empty")
	}

	// Normal test.
	sqlMock.ExpectQuery(`SELECT (.+) FROM "transactions"`).
		WithArgs(confirmationsArg, limitArg).WillReturnRows(
		sqlmock.NewRows(columns).AddRow(
			tx.ID, tx.Hash, tx.From, tx.To, tx.Amount, tx.Status,
			tx.Block, tx.Timestamp, tx.Marked, tx.Confirmations).AddRow(
			tx2.ID, tx2.Hash, tx2.From, tx2.To, tx2.Amount, tx2.Status,
			tx2.Block, tx2.Timestamp, tx2.Marked, tx2.Confirmations))

	for range txs {
		sqlMock.ExpectExec(`UPDATE "transactions"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	result, err = handler.GetLast(limitArg)
	if err != nil {
		t.Fatal(err)
	}

	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	checkFiled := func(exp, got interface{}) {
		if !reflect.DeepEqual(exp, got) {
			t.Fatalf("expected %v, got %v", exp, got)
		}
	}

	checkDate := func(str string, timestamp *uint64) {
		if timestamp != nil {
			tm := time.Unix(int64(*timestamp), 0)
			if !reflect.DeepEqual(tm.Format(time.RFC3339), str) {
				t.Fatalf("expected %v, got %v", tm.Format(time.RFC3339), str)
			}
		}
	}

	for k := range result {
		checkFiled(result[k].Confirmations, txs[k].Confirmations)
		checkFiled(result[k].Amount, txs[k].Amount)
		checkFiled(result[k].Hash, txs[k].Hash)
		checkFiled(result[k].Address, txs[k].To)
		checkDate(result[k].Date, txs[k].Timestamp)
	}
}

func testMain(m *testing.M) int {
	var err error
	var sqlDB *sql.DB

	sqlDB, sqlMock, err = sqlmock.New()
	if err != nil {
		return 1
	}

	dataBase, err = db.NewDB(sqlDB)
	if err != nil {
		return 1
	}

	defer db.CloseDB(dataBase)

	ethClient = newEthClient()

	handler = api.NewHandler(big.NewInt(4), dataBase, ethClient)

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}
