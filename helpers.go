package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	cockroach "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	sentry "github.com/karlsburg87/statusSentry/pkg/configuration"
)

//setupDb does the initial 'create if not exist...' setup
func setupDb(ctx context.Context, tx pgx.Tx) error {

	//setup database schema
	sql := `
	CREATE IF NOT EXIST statusUpdates(
		uid DEFAULT unique_rowid() PRIMARY KEY
		service_name STRING,
		domain STRING,
		message STRING,
		raw_message STRING,
		message_pub_timestamp TIMESTAMPTZ,
		status_page STRING,
		);

	CREATE IF NOT EXIST pingPolls(
		uid DEFAULT unique_rowid() PRIMARY KEY
		service_name STRING,
		domain STRING,
		status_page STRING,
		pinged_url STRING,
		status_code INT,
		error_text STRING,
		poll_timestamp TIMESTAMPTZ,
		dns_polltime INT,
		tls_handshake_polltime INT,
		connect_polltime INT,
		first_response_polltime INT,
		cert_verified BOOL,
		cert_valid_from DATE,
		cert_valid_until DATE,
		issuer STRING
	);	
		`
	info, err := tx.Exec(ctx, sql)
	log.Println(info.String())

	return err
}

//getURL constructs the database url from the envar config values
func getURL() (*url.URL, error) {
	var u *url.URL
	var err error
	if target := os.Getenv("DATABASE_URL"); target != "" {
		u, err = url.Parse(target)
		if err != nil {
			log.Fatalf("error parsing database_url: %v\n", err)
		}
	} else {
		//get configs from environment variables
		host := os.Getenv("HOST")                  //"free-tier5.gcp-europe-west1.cockroachlabs.cloud"
		username := os.Getenv("USERNAME")          //"username"
		password := os.Getenv("PASSWORD")          //"password"
		databaseName := os.Getenv("DATABASE_NAME") //"databaseName"
		routingID := os.Getenv("ROUTING_ID")       //"clusterName" // https://www.cockroachlabs.com/docs/stable/connect-to-the-database.html?filters=go#connection-parameters

		//parse DB_URL
		u := &url.URL{
			Scheme:   "postgresql",
			Host:     fmt.Sprintf("%s:%s", host, "port"),
			User:     url.UserPassword(username, password),
			Path:     databaseName,
			RawQuery: fmt.Sprint("options=--cluster%3D" + routingID),
		}
		fmt.Println(u.String())
		if u.String() == "" {
			return nil, fmt.Errorf("invalid db conn string - must provide DB_URL or the url elements HOST, USERNAME, PASSWORD, DATABASE_NAME, ROUTING_ID")
		}
	}
	return u, err
}

//insertRow adds a new row to the db from the received transporter object
func insertRow(ctx context.Context, tx pgx.Tx, t sentry.Transporter) error {
	var sql string
	var err error

	switch t.PingResponse == nil {
	case true:
		sql = `INSERT INTO statusUpdates (
			service_name, 
			domain, 
			message,
			raw_message,
			message_pub_timestamp,
			status_page) VALUES ($1,$2,$3,$4,$5,$6)`
		_, err = tx.Exec(ctx, sql, t.ServiceName, t.Domain, t.Message, t.RawMessage, t.MessagePublishedDateTime, t.StatusPage)

	default:
		sql = `INSERT INTO pingPolls (
			service_name,
			domain,
			status_page,
			pinged_url,
			status_code,
			error_text,
			poll_timestamp,
			dns_polltime,
			tls_handshake_polltime,
			connect_polltime,
			first_response_polltime,
			cert_valid_from,
			cert_valid_until,
			issuer,
			subject) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`

		//get primary cert
		var c sentry.PingCert
		for _, cert := range t.Certificates {
			if cert.ConnVerified {
				c = cert
				break
			}
		}

		_, err = tx.Exec(ctx, sql, t.ServiceName, t.Domain, t.StatusPage, t.PingResponse.URL, t.PingResponse.StatusCode, t.PingResponse.ErrorText, t.PingResponse.Time, t.ResponseTimes.DNS, t.ResponseTimes.TLSHandshake, t.ResponseTimes.Connect, t.ResponseTimes.FirstResponse, c.ValidFrom, c.ValidUntil, c.Issuer, c.Subject)
	}

	return err
}

//newServer is the server and mux that processes the Transport from the PubSub Chan and saves it to the server
func newServer(port int, dbConnPool *pgxpool.Pool) *http.Server {
	p := fmt.Sprintf(":%d", port)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		//parse to config from pubsub
		s := sentry.Transporter{}
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		//commit to database
		if err := cockroach.ExecuteTx(context.Background(), dbConnPool, pgx.TxOptions{}, func(tx pgx.Tx) error {
			if err := insertRow(context.Background(), tx, s); err != nil {
				return err
			}
			return nil
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	})

	return &http.Server{
		Addr:              p,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       20 * time.Second,
		Handler:           mux,
	}
}
