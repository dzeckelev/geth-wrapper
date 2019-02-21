package config

import "time"

type Config struct {
	Eth      *Eth
	DB       *DB
	DemoMode bool
}

type Eth struct {
	NodeURL    string
	QueryPause time.Duration // In milliseconds.
}

type DB struct {
	DBName   string
	Host     string
	Port     uint16
	User     string
	Password string
	// TODO: ssl mode is disabled.
}

func NewConfig() *Config {
	return &Config{
		Eth: &Eth{
			QueryPause: 15000, // 15 seconds
		},
		DB: &DB{
			DBName: "unionbase",
			Host:   "localhost",
			Port:   5433,
			User:   "postgres",
		},
	}
}
