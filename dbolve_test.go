package dbolve

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type dbCredentials struct {
	driver   string
	user     string
	password string
	host     string
	dbname   string
}

func (c *dbCredentials) connectDB(t *testing.T) *sql.DB {
	var url string
	switch c.driver {
	case "postgres":
		url = fmt.Sprintf("%s://%s:%s@%s/%s?sslmode=disable", c.driver, c.user, c.password, c.host, c.dbname)
	case "mysql":
		url = fmt.Sprintf("%s:%s@tcp(%s)/%s?autocommit=false", c.user, c.password, c.host, c.dbname)
	case "sqlite3":
		url = fmt.Sprintf("file:%s", c.dbname)
	default:
		t.Fail()
	}

	db, err := sql.Open(c.driver, url)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	return db
}

func CleanDB(t *testing.T, creds dbCredentials) *sql.DB {

	switch creds.driver {
	case "sqlite3":
		os.Remove(creds.dbname + ".db")
		return creds.connectDB(t)
	default:
		db := creds.connectDB(t)
		_, err := db.Exec("DROP DATABASE IF EXISTS dbolve_test;")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
		_, err = db.Exec("CREATE DATABASE dbolve_test;")
		if err != nil {
			t.Error(err)
			t.Fail()
		}
		db.Close()
		creds.dbname = "dbolve_test"
		db = creds.connectDB(t)
		return db
	}
}

func TestPostgres(t *testing.T) {
	testWithDB(t, dbCredentials{driver: "postgres", user: "postgres", password: "postgres", host: "localhost"})
}

func TestMySQL(t *testing.T) {
	testWithDB(t, dbCredentials{driver: "mysql", user: "root", password: "mysql", host: "localhost"})
}

func TestSQLite(t *testing.T) {
	testWithDB(t, dbCredentials{driver: "sqlite3"})
}

func testWithDB(t *testing.T, creds dbCredentials) {
	db := CleanDB(t, creds)
	_ = db.Close()
	testEvolution(db, t)
	_ = db.Close()
	db = CleanDB(t, creds)
	testModifiedMigration(db, t)
	_ = db.Close()
	db = CleanDB(t, creds)
	testFailingMigration(db, t)
	_ = db.Close()

	if creds.driver != "sqlite3" {
		db, _ = sql.Open(creds.driver, "some invalid")
		if m, err := NewMigrator(db, make([]Migration, 0)); err == nil || m != nil {
			t.Errorf("Invalid database should throw an error")
		}
	}
}

func testEvolution(db *sql.DB, t *testing.T) {
	migrations := []Migration{
		{
			Name: "First migration",
			Code: func(tx Transaction) error {
				return tx.Exec(`CREATE TABLE account(
					user_id serial PRIMARY KEY,
					username VARCHAR (50) UNIQUE NOT NULL,
					password VARCHAR (50) NOT NULL,
					email VARCHAR (355) UNIQUE NOT NULL,
					created_on TIMESTAMP NOT NULL,
					last_login TIMESTAMP
				 );`)
			},
		},
	}

	m, err := NewMigrator(db, migrations)
	if err != nil {
		t.Error(err)
	}

	if err := m.DryRun(); err != nil {
		t.Error(err)
	}

	if len(m.Applied()) != 0 {
		t.Errorf("At the beginning, there should not be applied migrations")
	}

	if len(m.Pending()) != len(migrations) {
		t.Errorf("At the beginning, there should be all migrations pending")
	}

	err = m.Migrate()
	if err != nil {
		t.Error(err)
	}

	if len(m.Applied()) != len(migrations) {
		t.Errorf("After migration, all migrations should be applied")
	}

	if len(m.Pending()) != 0 {
		t.Errorf("At migration, there should be no pending migrations")
	}

	migrations = append(migrations, Migration{
		Name: "Second migration",
		Code: func(tx Transaction) error {
			return tx.Exec(`CREATE TABLE account2(
					user_id serial PRIMARY KEY,
					username VARCHAR (50) UNIQUE NOT NULL,
					password VARCHAR (50) NOT NULL
				 );`)
		},
	})

	m, _ = NewMigrator(db, migrations)
	if len(m.Applied()) != len(migrations)-1 {
		t.Errorf("Old migratinos should be applied")
	}

	if len(m.Pending()) != 1 {
		t.Errorf("There should be one more pending migration")
	}

	err = m.Migrate()
	if err != nil {
		t.Error(err)
	}

	if len(m.Applied()) != len(migrations) {
		t.Errorf("After migration, all migrations should be applied")
	}

	if len(m.Pending()) != 0 {
		t.Errorf("At migration, there should be no pending migrations")
	}

	m, _ = NewMigrator(db, make([]Migration, 0))
	if err := m.Migrate(); err.Error() != "Found more applied migrations than supplied" {
		t.Errorf("Should not accept unknown migrations")
	}
}

func testModifiedMigration(db *sql.DB, t *testing.T) {
	migrations := []Migration{
		{
			Name: "First migration",
			Code: func(tx Transaction) error {
				return tx.Exec(`CREATE TABLE account(
					user_id serial PRIMARY KEY,
					username VARCHAR (50) UNIQUE NOT NULL,
					password VARCHAR (50) NOT NULL,
					email VARCHAR (355) UNIQUE NOT NULL,
					created_on TIMESTAMP NOT NULL,
					last_login TIMESTAMP
				 );`)
			},
		},
	}
	m, _ := NewMigrator(db, migrations)
	if err := m.Migrate(); err != nil {
		t.Error(err)
	}

	if m.CountApplied() != 1 {
		t.Error("No more migrations should have been applied")
	}

	migrations[0].Name = "Fist migration"
	m, _ = NewMigrator(db, migrations)
	if err := m.Migrate(); err.Error() != fmt.Sprintf("Migration \"%s\" names changed: current:\"%s\" != applied:\"%s\"", "Fist migration", "Fist migration", "First migration") {
		t.Error("name change should have been detected")
	}

	migrations[0].Name = "First migration"
	migrations[0].Code = func(tx Transaction) error {
		return tx.Exec(`CREATE TABLE account(
			email VARCHAR (355) UNIQUE NOT NULL,
			created_on TIMESTAMP NOT NULL,
			last_login TIMESTAMP
		 );`)
	}

	if err := m.Migrate(); err.Error() != fmt.Sprintf("Migration \"%s\" hash changed", "First migration") {
		t.Error("Name hash change should have been detected")
	}

	if m.CountApplied() != 1 {
		t.Error("No more migrations should have been applied")
	}
}

func testFailingMigration(db *sql.DB, t *testing.T) {
	migrations := []Migration{
		{
			Name: "First migration",
			Code: func(tx Transaction) error {
				return tx.Exec(`CREATE TABLE WITH SYNTAX ERROR);`)
			},
		},
	}
	m, _ := NewMigrator(db, migrations)
	if err := m.Migrate(); strings.Contains(err.Error(), "Migration (0) - \"First migration\" returned an error:") {
		t.Error("Error not thrown on invalid migration code")
	}
	if m.CountApplied() != 0 {
		t.Error("No migrations should have been applied")
	}
}
