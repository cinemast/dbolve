package main

import (
	"database/sql"
	"fmt"
	"github.com/cinemast/dbolve"
	_ "github.com/lib/pq"
	"log/slog"
)

func main() {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost/dbolve_test?sslmode=disable")
	if err != nil {
		panic(err)
	}

	migrations := []dbolve.Migration{
		{
			Name: "Add account table",
			Code: func(tx dbolve.Transaction) error {
				return tx.Exec(`CREATE TABLE account(
						user_id serial PRIMARY KEY,
						username VARCHAR (50) UNIQUE NOT NULL,
						password VARCHAR (50) NOT NULL
					 );`)
			},
		},
		{
			Name: "Add account 2 table",
			Code: func(tx dbolve.Transaction) error {
				return tx.Exec(`CREATE TABLE account2(
					user_id serial PRIMARY KEY,
					username VARCHAR (50) UNIQUE NOT NULL,
					password VARCHAR (50) NOT NULL
				 );`)
			},
		},
	}
	m, err := dbolve.NewMigrator(db, migrations, slog.Default())
	if err != nil {
		panic(err)
	}
	if err := m.DryRun(); err != nil {
		panic(err)
	}
	fmt.Println("Finished migrations")
}
