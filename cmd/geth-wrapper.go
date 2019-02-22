package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/dzeckelev/geth-wrapper/api"
	"github.com/dzeckelev/geth-wrapper/blockchain"
	"github.com/dzeckelev/geth-wrapper/config"
	"github.com/dzeckelev/geth-wrapper/db"
	"github.com/dzeckelev/geth-wrapper/proc"
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

	ethClient, err := blockchain.NewClient(ctx, cfg.Eth)
	if err != nil {
		log.Fatal(err)
	}
	defer blockchain.Close(ethClient)

	netID, err := ethClient.NetworkID(ctx)
	if err != nil {
		log.Fatal(err)
	}

	database, err := db.Connect(db.ConnectArgs(cfg.DB))
	if err != nil {
		log.Fatal(err)
	}
	defer db.CloseDB(database)

	pr, err := proc.NewScheduler(ctx, netID, cfg, database, ethClient)
	if err != nil {
		log.Fatal(err)
	}
	defer pr.Close()

	if err := pr.Start(); err != nil {
		log.Fatal(err)
	}

	handler := api.NewHandler(netID, database, ethClient)
	srv, err := api.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := srv.AddHandler(handler); err != nil {
		log.Fatal(err)

	}

	panicChan := make(chan error)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// TODO handle error
			panicChan <- err
		}
	}()
	defer srv.Close()

	log.Fatal(<-panicChan)
}
