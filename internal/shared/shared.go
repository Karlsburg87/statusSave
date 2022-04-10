package shared

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	cockroach "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgx"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	sqlstatements "github.com/karlsburg87/statusSave/internal/sqlStatements"
	sentry "github.com/karlsburg87/statusSentry/pkg/configuration"
)

//setupDb does the initial 'create if not exist...' setup
func setupDb(ctx context.Context, tx pgx.Tx) error {
	//setup database schemas

	info, err := tx.Exec(ctx, sqlstatements.CreateDatabase)
	log.Println("setup info " + info.String())

	return err
}

//SetupTables does the initial 'create if not exist...' setup for the tables
func SetupTables(ctx context.Context, tx pgx.Tx) error {
	info, err := tx.Exec(ctx, sqlstatements.CreateTables)
	log.Println("execution info : " + info.String())

	return err
}

//RunDatabaseSetup does the initial setup of a database in the CockroachDB cluster
func RunDatabaseSetup(ctx context.Context, targetURL *url.URL) {
	//create single thread connection
	setupURL := *targetURL
	setupURL.Path = "defaultdb"
	suq := setupURL.Query()
	suq.Set("sslmode", "verify-full")
	setupURL.RawQuery = suq.Encode()

	log.Printf("using defaultdb connection for setup: %s\n", setupURL.String())

	suConn, err := pgx.Connect(ctx, setupURL.String())
	if err != nil {
		log.Fatalln(err)
	}
	defer suConn.Close(ctx)

	//create schema on Cockroach db if not exist
	if err := cockroach.ExecuteTx(ctx, suConn, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if err := setupDb(ctx, tx); err != nil {
			return fmt.Errorf("error creating schema on db : %v", err)
		}
		return nil
	}); err != nil {
		log.Fatalln(err)
	}
}

//GetURL constructs the database url from the envar config values
func GetURL() (*url.URL, error) {
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
		port := os.Getenv("DB_PORT")

		//parse DB_URL
		u := &url.URL{
			Scheme:   "postgresql",
			Host:     fmt.Sprintf("%s:%s", host, port),
			User:     url.UserPassword(username, password),
			Path:     strings.ToLower(databaseName), //db name is always lower case in cockroachDB
			RawQuery: fmt.Sprint("options=--cluster%3D" + routingID),
		}
		fmt.Println(u.String())
		if u.String() == "" {
			return nil, fmt.Errorf("invalid db conn string - must provide DB_URL or the url elements HOST, USERNAME, PASSWORD, DATABASE_NAME, ROUTING_ID")
		}
	}
	return u, err
}

//nullableString creates strings that db drivers can handle that may be null
func nullableString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

//nullableInt64 creates int64 that db drivers can handle that may be null
func nullableInt64(d int64) sql.NullInt64 {
	if d == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{
		Int64: d,
		Valid: true,
	}
}

//nullableInt creates int64 that db drivers can handle that may be null
func nullableInt(d int) sql.NullInt64 {
	return nullableInt64(int64(d))
}

//ISO8601Encode encodes a time from Transporter to ISO8601 that Cockroachdb understands in UTC
func ISO8601Encode(rfc3339 string) (iso8601 string, err error) {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		log.Printf("could not parse time string as RFC3339 in ISO8601Encode : %v", err)
		return
	}
	return t.UTC().Format(time.RFC3339), nil
}

//ISO8601Decode decodes UTC ISO8601 to local time RFC3339
func ISO8601Decode(iso8601 string) (rfc3339 string, err error) {
	t, err := time.Parse(time.RFC3339, iso8601)
	if err != nil {
		log.Printf("could not parse time string as RFC3339 in ISO8601Decode : %v", err)
		return
	}
	return t.Local().Format(time.RFC3339), nil
}

//insertRow adds a new row to the db from the received transporter object
func insertRow(ctx context.Context, tx pgx.Tx, t *sentry.Transporter) error {
	//pointers deals with null issue: https://www.manniwood.com/2016_08_21/pgxfiles_08.html
	fmt.Printf("transport in insertRow func : %+v\n", *t)
	fmt.Printf("transport in insertRow nullable string : %+v\n", t.DisplayServiceName)
	switch t.PingResponse == nil {
	case true:
		pubtime, err := ISO8601Encode(t.MessagePublishedDateTime)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, sqlstatements.InsertStatusUpdates, nullableString(t.Message), nullableString(t.RawMessage), nullableString(pubtime), nullableString(t.MetaStatusPage), nullableString(t.DisplayServiceName), nullableString(t.DisplayDomain), nullableString(t.MetaStatusPage)); err != nil {
			return err
		}

	default:

		//get primary cert
		var c sentry.PingCert
		for _, cert := range t.Certificates {
			if cert.ConnVerified {
				c = cert
				break
			}
		}
		time.Now().UTC().Format(time.RFC3339)
		timestamp, err := ISO8601Encode(t.PingResponse.Time)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, sqlstatements.InsertPingPolls, nullableString(t.PingResponse.URL), nullableInt(t.PingResponse.StatusCode), nullableString(t.PingResponse.ErrorText), nullableString(timestamp), nullableInt64(t.ResponseTimes.DNS), nullableInt64(t.ResponseTimes.TLSHandshake), nullableInt64(t.ResponseTimes.Connect), nullableInt64(t.ResponseTimes.FirstResponse), nullableString(c.ValidFrom), nullableString(c.ValidUntil), nullableString(c.Issuer), nullableString(c.Subject), nullableString(t.DisplayServiceName), nullableString(t.DisplayDomain), nullableString(t.MetaStatusPage)); err != nil {
			return err
		}
	}

	return nil
}

//NewServer is the server and mux that processes the Transport from the PubSub Chan and saves it to the server
func NewServer(ctx context.Context, port int, dbConnPool *pgxpool.Pool) *http.Server {
	p := fmt.Sprintf(":%d", port)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "note": "were you looking for the /receive endpoint?"}); err != nil {
			log.Panicln(err)
		}
	})
	mux.HandleFunc("/receive", func(w http.ResponseWriter, r *http.Request) {

		//parse to config from pubsub
		transport := &sentry.Transporter{}
		if err := json.NewDecoder(r.Body).Decode(transport); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"error": err, "note": "issue decoding json body"}); err != nil {
				log.Panicln(err)
			}
			return
		}
		fmt.Printf("transporter : %+v", transport)
		//check message is valid to store
		if transport.Message == "" && transport.PingResponse == nil {
			log.Printf("no message: %+v\n", transport)
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("no message: %+v", transport), "note": "empty transport"}); err != nil {
				log.Panicln(err)
			}
			return
		}

		//commit to database
		if err := cockroach.ExecuteTx(ctx, dbConnPool, pgx.TxOptions{}, func(tx pgx.Tx) error {
			fmt.Printf("tx : %+v\n", tx)
			return insertRow(ctx, tx, transport)
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{"error": err, "note": "issue executing insert row"}); err != nil {
				log.Panicln(err)
			}
			return
		}

		fmt.Fprintln(w, "Done.")
	})

	return &http.Server{
		Addr:              p,
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       20 * time.Second,
		Handler:           mux,
	}
}
