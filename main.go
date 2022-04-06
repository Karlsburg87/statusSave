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
)

func main() {
	//sort out db connection URL

	u, err := getURL()
	if err != nil {
		log.Fatalln(err)
	}

	//calculate max connection poolsize
	poolSize := runtime.NumCPU() * 4
	if poolSize == 0 {
		poolSize = 8
	}

	//add query options for conn pool
	query := u.Query()
	query.Set("pool_max_conns", strconv.Itoa(poolSize)) //connections = (number of cores * 4) - https://www.cockroachlabs.com/docs/stable/connection-pooling.html?filters=go
	query.Set("sslmode", "verify-full")

	u.RawQuery = query.Encode()

	// Set connection pool configuration, with maximum connection pool size.
	config, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		log.Fatal("error configuring the database: ", err)
	}

	// Create a connection pool to the database.
	dbPool, err := pgxpool.ConnectConfig(context.Background(), config)
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}
	defer dbPool.Close()

	//create schema on Cockroach db if not exist
	if err := cockroach.ExecuteTx(context.Background(), dbPool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if err := setupDb(context.Background(), tx); err != nil {
			return fmt.Errorf("error creating schema on db : %v", err)
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
	log.Fatalln(newServer(p, dbPool).ListenAndServe())
}
