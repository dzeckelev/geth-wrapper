## Geth-wrapper

### Geth

Method `unlockAccount` not implemented.
Tested with the following settings Geth.

```bash
geth --rinkeby --gcmode=archive --rpc --rpcapi "eth,net,personal" --unlock 0xd1dffc3c0537d46cd65b10019d4216f9dcd7e114
```

### Database preparation
```bash
psql -U postgres -f $GOPATH/src/github.com/dzeckelev/geth-wrapper/data/prepare.sql
psql -U postgres -d unionbase -f $GOPATH/src/github.com/dzeckelev/geth-wrapper/data/migrate.sql
```

### Build

Dependencies are already in the repository.

```bash
cd $GOPATH/src/github.com/dzeckelev
git clone https://github.com/dzeckelev/geth-wrapper.git
cd geth-wrapper
go get -u -v gopkg.in/reform.v1/reform
go generate ./...
go install
```

### Run

```bash
geth-wrapper -config config.json
```

### API methods

#### Get Last Transactions

Arguments:
- Limit: limits the number of transactions in a response.

Example: 
```bash
curl -X POST -H "Content-Type: application/json" --data '{"method": "api_getLast", "params": [100], "id": 100}' http://localhost:8081/http
```

#### Send 

Arguments:
- From: sender address.
- To: recipient address.
- Amount: The amount of Wei sent with this transaction. (1 ETH = 10^18 Wei)

```bash
curl -X POST -H "Content-Type: application/json" --data '{"method": "api_sendEth", "params": ["0xd1dffc3c0537d46cd65b10019d4216f9dcd7e114", "0xd6d39cd7672841789dc3afb97525984b6d31f796", "1000000000000"], "id": 100}' http://localhost:8081/http
```