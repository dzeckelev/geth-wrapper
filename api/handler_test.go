package api_test

import (
	"context"
	"database/sql/driver"
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
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	ethClient *eth.MockClient
	columns   = []string{"o_id", "o_hash", "o_from", "o_to", "o_amount",
		"o_status", "o_block", "o_timestamp", "o_marked", "o_confirmations"}
	network = big.NewInt(4)
)

func genNewAccount() *bind.TransactOpts {
	key, _ := crypto.GenerateKey()
	return bind.NewKeyedTransactor(key)
}

func newEthClient() *eth.MockClient {
	opts := genNewAccount()
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

func newDB(t *testing.T) (*reform.DB, sqlmock.Sqlmock) {
	sqlDB, sqlMock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	dataBase, err := db.NewDB(sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	return dataBase, sqlMock
}

func newTestTx() *data.Transaction {
	hash := "0x64e604787cbf194841e7b68d7cd28786f6c9a0a3ab9f8b0a0e87cb4387ab0107"
	from := "0xe7dc9fe68da458b54f648146a817126053eeef66"
	to := "0xa7dba6053a0d631177340e8061bc12f5009ba453"
	amount := strconv.Itoa(10000)
	status := pointer.ToString(data.TxSuccessful)
	block := pointer.ToUint64(123456)
	timestamp := pointer.ToUint64(777777)
	confirmations := uint64(1)

	return &data.Transaction{
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
}

func createTestTxs(num int) (result []*data.Transaction) {
	for i := 0; i < num; i++ {
		result = append(result, newTestTx())
	}
	return result
}

func toRow(tx *data.Transaction) (result []driver.Value) {
	result = make([]driver.Value, len(tx.Values()))

	for k, v := range tx.Values() {
		result[k] = v
	}

	return result
}

func checkFiled(t *testing.T, exp, got interface{}) {
	if !reflect.DeepEqual(exp, got) {
		t.Fatalf("expected %v, got %v", exp, got)
	}
}

func checkDate(t *testing.T, str string, timestamp *uint64) {
	if timestamp != nil {
		tm := time.Unix(int64(*timestamp), 0)
		if !reflect.DeepEqual(tm.Format(time.RFC3339), str) {
			t.Fatalf("expected %v, got %v", tm.Format(time.RFC3339), str)
		}
	}
}

func checkGetLast(t *testing.T, handler *api.Handler,
	limit uint64, expected int) []api.GetLastResult {
	result, err := handler.GetLast(limit)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != expected {
		t.Fatalf("expected %v, got %v", expected, len(result))
	}

	return result
}

func TestGetLast(t *testing.T) {
	dataBase, sqlMock := newDB(t)
	handler := api.NewHandler(network, dataBase, ethClient, nil)

	limit := uint64(100)
	confirmations := uint64(3)

	txs := createTestTxs(2)

	expSelectSQL := `SELECT (.+) FROM "transactions"`
	expUpdateSQL := `UPDATE "transactions"`

	// Empty result.
	sqlMock.ExpectQuery(expSelectSQL).
		WithArgs(confirmations, limit).WillReturnRows(sqlmock.NewRows(columns))

	checkGetLast(t, handler, limit, 0)

	// Normal test.
	sqlMock.ExpectQuery(expSelectSQL).
		WithArgs(confirmations, limit).WillReturnRows(sqlmock.NewRows(columns).
		AddRow(toRow(txs[0])...).AddRow(toRow(txs[1])...))

	for range txs {
		sqlMock.ExpectExec(expUpdateSQL).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	result := checkGetLast(t, handler, limit, 2)

	for k := range result {
		checkFiled(t, result[k].Confirmations, txs[k].Confirmations)
		checkFiled(t, result[k].Amount, txs[k].Amount)
		checkFiled(t, result[k].Hash, txs[k].Hash)
		checkFiled(t, result[k].Address, txs[k].To)
		checkDate(t, result[k].Date, txs[k].Timestamp)
	}

	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestHandlerSendETH(t *testing.T) {
	dataBase, sqlMock := newDB(t)
	handler := api.NewHandler(network, dataBase, ethClient, gen.NewUUID)

	accounts, err := ethClient.Accounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	opts := genNewAccount()
	to := strings.ToLower(opts.From.String())

	expUpdateSQL := `UPDATE "outputs"`

	sqlMock.ExpectExec(expUpdateSQL).
		WillReturnResult(sqlmock.NewResult(1, 1))

	hash, err := handler.SendETH(accounts[0], to, "10000")
	if err != nil {
		t.Fatal(err)
	}

	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	_, err = hexutil.Decode(*hash)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	ethClient = newEthClient()
	os.Exit(m.Run())
}
