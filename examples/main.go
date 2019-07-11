package main

import (
	_ "github.com/lib/pq"
	"database/sql"
	"log"
	"os"
	"fmt"
	"github.com/cinemast/dbolve"
)

func main() {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost/dbolve_test?sslmode=disable")
    if err != nil {
		panic(err)
	}

	migrations := []dbolve.Migration{
		dbolve.Migration{
			Name: "Add acccount table", 
				Code: func(tx dbolve.Transaction) error {
					return tx.Exec(`CREATE TABLE account(
						user_id serial PRIMARY KEY,
						username VARCHAR (50) UNIQUE NOT NULL,
						password VARCHAR (50) NOT NULL
					 );`)
				},
			},
		dbolve.Migration{
		Name: "Add acccount 2 table", 
			Code: func(tx dbolve.Transaction) error {
				return tx.Exec(`CREATE TABLE account2(
					user_id serial PRIMARY KEY,
					username VARCHAR (50) UNIQUE NOT NULL,
					password VARCHAR (50) NOT NULL
				 );`)
			},
		},
	}
	m,err := dbolve.NewMigrator(db, migrations)
	if err != nil {
		panic(err)
	}
	m.Log = log.New(os.Stdout, "", log.LstdFlags)
	if err := m.Migrate(); err != nil {
		panic(err)
	}
	fmt.Println("Finished migrations")
}