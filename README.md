[![CircleCI](https://circleci.com/gh/cinemast/dbolve.svg?style=svg)](https://circleci.com/gh/cinemast/dbolve)
[![codecov](https://codecov.io/gh/cinemast/dbolve/branch/master/graph/badge.svg)](https://codecov.io/gh/cinemast/dbolve)
[![GoDoc](https://godoc.org/github.com/cinemast/dbolve/go?status.svg)](https://godoc.org/github.com/cinemast/dbolve)
[![Go Report Card](https://goreportcard.com/badge/github.com/cinemast/dbolve)](https://goreportcard.com/report/github.com/cinemast/dbolve)

# dbolve

Very simple code only migration library for go

## Features
- Very simple and readable code (< 200 lines of code)
- Easy to use interface
- Transaction safety for each migration
- Verifies that already applied transactions haven't changed

## Usage
```
go get -u github.com/cinemast/dbolve
```

### Quickstart

```go
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
					return tx.Exec(`CREATE TABLE account(user_id serial PRIMARY KEY, username VARCHAR (50) UNIQUE NOT NULL, password VARCHAR (50) NOT NULL);`)
				},
			},
		dbolve.Migration{
		Name: "Add acccount 2 table",
			Code: func(tx dbolve.Transaction) error {
				return tx.Exec(`CREATE TABLE account2(user_id serial PRIMARY KEY, username VARCHAR (50) UNIQUE NOT NULL, password VARCHAR (50) NOT NULL);`)
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
```

## Motivation

Heavily inspired by [lopezator/migrator](https://github.com/lopezator/migrator).

I was missing two features:
  - Allow to list already applied and pending migrations
  - Verification that already applied migrations match the current migration code
