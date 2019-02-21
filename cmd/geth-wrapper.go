package main

import (
	"context"
	"encoding/json"
	"flag"
	"github.com/dzeckelev/geth-wrapper/config"
	"github.com/dzeckelev/geth-wrapper/db"
	"github.com/dzeckelev/geth-wrapper/service"
	"log"
	"os"
)

func readConfig(name string, data interface{}) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(data)
}

func main() {
	cfg := config.NewConfig()
	fConfig := flag.String("config", "config.json", "Configuration file path.")
	if err := readConfig(*fConfig, cfg); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.Connect(db.ConnectArgs(cfg.DB))
	if err != nil {
		log.Fatal(err)
	}
	defer db.CloseDB(database)

	svc, err := service.NewService(ctx, cfg, database)
	if err != nil {
		log.Fatal(err)
	}

	if err := svc.Start(); err != nil {
		log.Fatal(err)
	}

}
