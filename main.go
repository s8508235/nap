package main

import (
	"context"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/s8508235/nap/lib"
)

func main() {
	// The first DSN is assumed to be the master and all
	// other to be slaves
	dsns := "postgres://postgres:postgres@localhost:32780/postgres?sslmode=disable;"
	dsns += "postgres://repl_user:repl_password@localhost:32781/postgres?sslmode=disable;"
	dsns += "postgres://repl_user:repl_password@localhost:32782/postgres?sslmode=disable;"
	dsns += "postgres://repl_user:repl_password@localhost:32783/postgres?sslmode=disable"

	db, err := lib.Open("postgres", dsns)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Some physical database is unreachable: %s", err)
	}

	// Prepared statements are aggregates. If any of the underlying
	// physical databases fails to prepare the statement, the call will
	// return an error. On success, if Exec is called, then the
	// master is used, if Query or QueryRow are called, then a slave
	// is used.
	stmt, err := db.Prepare(`CREATE TABLE IF NOT EXISTS TEST_USER (id serial PRIMARY KEY,
		username VARCHAR(50) DEFAULT '')`)

	ctx, canc := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer canc()
	_, err = stmt.ExecContext(ctx)

	if err != nil {
		log.Fatal(err)
	}

	log.Println(`Create table success`)

	// Read queries are directed to slaves with Query and QueryRow.
	// Always use Query or QueryRow for SELECTS
	// Load distribution is round-robin only for now.
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM TEST_USER").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(count)

	// Transactions always use the master
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	// Write queries are directed to the master with Exec.
	// Always use Exec for INSERTS, UPDATES
	_, err = db.Exec("INSERT INTO TEST_USER (username) VALUES  ('test')")
	if err != nil {
		log.Fatal(err)
	}

	log.Println(`insert success`)

	_, err = db.Exec("UPDATE TEST_USER SET username = 'test123'")
	if err != nil {
		log.Fatal(err)
	}

	log.Println(`update success`)

	stmt, err = db.Prepare("SELECT * FROM TEST_USER WHERE id = 1")
	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec()

	if err != nil {
		log.Fatal(err)
	}
	log.Println(`select success`)

	// Do something transactional ...
	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}
}
