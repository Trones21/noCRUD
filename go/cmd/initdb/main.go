// Command initdb creates a database named by the DB_NAME environment variable —
// the Go counterpart of python/init_db.py.
//
// This isn't really part of the test runner; it's just a quick way to create a
// database without reaching for psql.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Trones21/noCRUD/go/utils/dbclient"
)

func main() {
	name := os.Getenv("DB_NAME")
	if name == "" {
		fmt.Fprintln(os.Stderr, "DB_NAME is not set")
		os.Exit(2)
	}

	ctx := context.Background()
	client, err := dbclient.New(ctx, dbclient.AdminConfig(), os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer client.Close(ctx)

	if err := client.CreateDB(ctx, name); err != nil {
		os.Exit(1)
	}
}
