package config

// Config is an application configuration.
type Config struct {
	API  *API
	DB   *DB
	Eth  *Eth
	Proc *Proc
}

// Eth is a communication configuration with Ethereum.
type Eth struct {
	NodeURL    string
	StartBlock uint64
}

// DB is a database configuration.
type DB struct {
	DBName   string
	Host     string
	Port     uint16
	User     string
	Password string
	// TODO: ssl mode is disabled.
}

// API is a API configuration.
type API struct {
	Addr string
}

// Proc is a processing configuration.
type Proc struct {
	UpdateLastBlockPause    uint64 // In milliseconds.
	CollectPause            uint64 // In milliseconds.
	UpdateTransactionsPause uint64 // In milliseconds.
}

// NewConfig creates a default application configuration.
func NewConfig() *Config {
	return &Config{
		API: &API{
			Addr: "localhost:80",
		},
		Eth: &Eth{
			StartBlock: 0,
		},
		DB: &DB{
			DBName: "unionbase",
			Host:   "localhost",
			Port:   5433,
			User:   "postgres",
		},
		Proc: &Proc{
			UpdateLastBlockPause:    15000,
			CollectPause:            15000,
			UpdateTransactionsPause: 20000,
		},
	}
}
