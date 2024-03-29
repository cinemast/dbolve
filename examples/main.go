package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/cinemast/dbolve"
	_ "github.com/lib/pq"
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
	m, err := dbolve.NewMigrator(db, migrations)
	if err != nil {
		panic(err)
	}
	m.Log = log.New(os.Stdout, "", 0)
	if err := m.DryRun(); err != nil {
		panic(err)
	}
	fmt.Println("Finished migrations")
}
