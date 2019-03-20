package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"github.com/dzeckelev/geth-wrapper/api"
	"github.com/dzeckelev/geth-wrapper/config"
	"github.com/dzeckelev/geth-wrapper/db"
	"github.com/dzeckelev/geth-wrapper/eth"
	"github.com/dzeckelev/geth-wrapper/gen"
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

	flag.Parse()

	if err := readConfig(*fConfig, cfg); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ethClient, err := eth.NewClient(ctx, cfg.Eth.NodeURL)
	if err != nil {
		log.Fatal(err)
	}
	defer ethClient.Close()

	syncPause := time.Duration(cfg.Proc.SyncPause) * time.Millisecond
	if err := eth.WaitSync(ctx, ethClient, syncPause); err != nil {
		log.Fatal(err)
	}

	netID, err := ethClient.NetworkID(ctx)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := sql.Open("postgres", db.ConnectArgs(cfg.DB))
	if err == nil {
		err = conn.Ping()
	}
	if err != nil {
		log.Fatal(err)
	}

	database, err := db.NewDB(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.CloseDB(database)

	scheduler, err := proc.NewScheduler(ctx, netID, cfg, database, ethClient)
	if err != nil {
		log.Fatal(err)
	}
	defer scheduler.Close()

	if err := scheduler.Start(); err != nil {
		log.Fatal(err)
	}

	handler := api.NewHandler(netID, database, ethClient, gen.NewUUID)
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
