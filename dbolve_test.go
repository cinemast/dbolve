package dbolve

import(
	_ "github.com/lib/pq"
	"testing"
	"fmt"
	"strings"
	"database/sql"
)

func CleanPostgresDB(t *testing.T) *sql.DB {
	dbhost := "localhost"
	dbuser := "postgres"
	dbpass := "postgres"
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s?sslmode=disable", dbuser, dbpass, dbhost))
    if err != nil {
		t.Error(err)
        t.Fail()
	}
	db.Exec("CREATE DATABASE dbolve_test;")
	db.Close()
	db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/dbolve_test?sslmode=disable", dbuser, dbpass, dbhost))
	if err != nil {
        t.Error(err)
        t.Fail()
	}
	db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	return db
}

func TestPostgres(t *testing.T) {
	db := CleanPostgresDB(t)
	defer db.Close()
	testEvolution(db, t)
	db.Close()
	db = CleanPostgresDB(t)
	testModifiedMigration(db, t)
	db.Close()
	db = CleanPostgresDB(t)
	testFailingMigration(db, t)
	db.Close()
}

func testEvolution(db *sql.DB, t *testing.T) {
	migrations := []Migration{
		Migration{
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

	m,err := NewMigrator(db, migrations)

	if len(m.Applied()) != 0 {
		t.Errorf("At the beginning, there should not be applied migrations")
	}

	if len(m.Pending()) != len(migrations) {
		t.Errorf("At the beginning, there should be all migrations pending")
	}
	
	if err != nil {
		t.Error(err)
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

	m,err = NewMigrator(db, migrations)
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

	m,err = NewMigrator(db, make([]Migration,0))
	if err := m.Migrate(); err.Error() != "Found more applied migrations than supplied" {
		t.Errorf("Should not accept unknown migrations")
	}
}

func testModifiedMigration(db *sql.DB, t *testing.T) {
	migrations := []Migration{
		Migration{
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
	m,_ := NewMigrator(db, migrations)
	if err := m.Migrate(); err != nil {
		t.Error(err)
	}

	if m.CountApplied() != 1 {
		t.Error("No more migrations should have been applied")
	}

	migrations[0].Name = "Fist migration"
	m,_ = NewMigrator(db, migrations)
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
		Migration{
			Name: "First migration", 
			Code: func(tx Transaction) error {
				return tx.Exec(`CREATE TABLE WITH SYNTAX ERROR);`)
			},
		},
	}
	m,_ := NewMigrator(db, migrations)
	if err := m.Migrate(); strings.Contains(err.Error(), "Migration (0) - \"First migration\" returned an error:") {
		t.Error("Error not thrown on invalid migration code")
	}
	if m.CountApplied() != 0 {
		t.Error("No migrations should have been applied")
	}
}

//Test failed migrator new