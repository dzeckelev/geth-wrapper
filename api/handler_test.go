package api_test

import (
	"fmt"
	"github.com/dzeckelev/geth-wrapper/db"
	"github.com/dzeckelev/geth-wrapper/eth"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"gopkg.in/reform.v1"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

const gasLimit uint64 = 4700000

func newDB(t *testing.T) (*reform.DB, sqlmock.Sqlmock) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	dbase, err := db.NewDB(database)
	if err != nil {
		t.Fatal(err)
	}

	return dbase, mock
}

func newEthClinet() *eth.MockClient {
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
		Acc:   map[string]*bind.TransactOpts{addr: opts},
		NetID: big.NewInt(4),
	}
}

func TestHandlerSendETH(t *testing.T) {
	dbase, mock := newDB(t)

}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
