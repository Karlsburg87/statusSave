package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	cockroach "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/karlsburg87/saveStatus/internal/shared"
)

func main() {
	//sort out db connection URL

	u, err := shared.GetURL()
	if err != nil {
		log.Fatalln(err)
	}
	ctx := context.Background()

	/***
	* do database setup if not exists
	* --- Requires single conn to defaultDB ---
	***/
	shared.RunDatabaseSetup(ctx, u) //all errors are fatal at startup so nothing returned

	/***
	* calculate max connection poolsize
	* --- for main pool connection
	***/
	poolSize := runtime.NumCPU() * 4
	if poolSize == 0 {
		poolSize = 8
	}
	//add query options for conn pool
	query := u.Query()
	query.Set("pool_max_conns", strconv.Itoa(poolSize)) //connections = (number of cores * 4) - https://www.cockroachlabs.com/docs/stable/connection-pooling.html?filters=go
	query.Set("sslmode", "verify-full")

	u.RawQuery = query.Encode()

	log.Printf("using main connection url for the worker pool: %s\n", u.String())

	// Set connection pool configuration, with maximum connection pool size.
	config, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		log.Fatal("error configuring the database: ", err)
	}

	// Create a connection pool to the database.
	dbPool, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}
	//defer dbPool.Close()

	//setup tables if not exists
	//create schema on Cockroach db if not exist
	if err := cockroach.ExecuteTx(ctx, dbPool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if err := shared.SetupTables(ctx, tx); err != nil {
			return fmt.Errorf("error creating tables on db : %v", err)
		}
		return nil
	}); err != nil {
		log.Fatalln(err)
	}

	//set up server
	p := 8080
	port := os.Getenv("PORT")
	if port != "" {
		p, err = strconv.Atoi(port)
		if err != nil {
			log.Fatalf("could not parse port value of %s", port)
		}
	}
	log.Printf("launching server on port %d", p)
	log.Fatalln(shared.NewServer(ctx, p, dbPool).ListenAndServe())
}
