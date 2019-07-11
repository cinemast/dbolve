package dbolve

import (
	"crypto/sha256"
	"bytes"
	"fmt"
	"errors"
	"log"
	"io/ioutil"
	"database/sql"
)

const(
	logPrefix = "dbolve: "
	tableName = "dbolve_migrations"
)

//Migration struct
type Migration struct {
	Name string
	Code func(t Transaction) error
	Timestamp string
	idx int
	hash string
}

//Migrator type
type Migrator struct {
	db *sql.DB
	Migrations []Migration
	Log *log.Logger
}

//Transaction exposes allowed database operations for migrations
type Transaction interface {
	Exec(sql string) error
}

//NewMigrator creates a new instance of Migrator
func NewMigrator(db *sql.DB, migrations []Migration) (*Migrator,error) {
	err := db.Ping()
	if err != nil {
		return nil, errors.New(logPrefix + "Could not connect to db, "+ err.Error())
	}
	_, err = db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INT NOT NULL, name VARCHAR(255) NOT NULL, hash VARCHAR(64) NOT NULL, timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (id));", tableName))
	if err != nil {
		return nil, errors.New(logPrefix + "Could not create migration table: " + err.Error())
	}
	return &Migrator{db, migrations, log.New(ioutil.Discard, logPrefix, log.Ldate)}, nil
}

//Pending returns a slice of not yet applied migrations
func (m *Migrator) Pending() []Migration {
	return m.Migrations[m.CountApplied():len(m.Migrations)]
}

//Applied returns a slice of already applied migrations
func (m *Migrator) Applied() []Migration {
	return readAppliedMigrations(m.db)
}

//CountApplied returns the number of already applied migrations
func (m *Migrator) CountApplied() int {
	row := m.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s;",tableName))
	count := 0
	_ = row.Scan(&count)
	return count
}

//Migrate run's all missing migrations
func (m *Migrator) Migrate() error {
	return m.migrate(false)
}

//DryRun tries to run the migrations but rollbacks each transaction
func (m *Migrator) DryRun() error {
	return m.migrate(true)
}

func (m *Migrator) migrate(dryRun bool) error {
	appliedMigrations := m.Applied()
	if len(appliedMigrations) > len(m.Migrations) {
		return errors.New("Found more applied migrations than supplied")
	}
	for idx,applied := range m.Applied() {
		if err := verifyMigration(applied, m.Migrations[idx]); err != nil {
			m.Log.Printf("%s☓ Verification failed (%d) \"%s\" -> %s", logPrefix, idx, applied.Name, err.Error())
			return err
		}
		m.Log.Printf("%s✔  Verified migration (%d) \"%s\"",logPrefix, idx, applied.Name)
	}
	for idx,pending := range m.Migrations[len(appliedMigrations):len(m.Migrations)] {
		if dryRun {
			m.Log.Printf("%sWould apply migration (%d) \"%s\"", logPrefix, idx+len(appliedMigrations), pending.Name)
		}
		if err := applyMigration(m.db, idx+len(appliedMigrations), &pending, dryRun, m.Log); err != nil {
			m.Log.Printf("%s: ☓ Migration failed (%d) \"%s\" -> %s", logPrefix, idx+len(appliedMigrations), pending.Name, err.Error())
			return err
		}
		if !dryRun {
			m.Log.Printf("%s★  Applied migration (%d) \"%s\"", logPrefix, idx+len(appliedMigrations), pending.Name)
		}
	}
 	return nil
}

func readAppliedMigrations(db *sql.DB) ([]Migration) {
	rows, _ := db.Query(fmt.Sprintf("SELECT * FROM %s;",tableName))
	defer rows.Close()
	migrations := make([]Migration,0)
	for rows.Next() {
		migration := Migration{}
		_ = rows.Scan(&migration.idx, &migration.Name, &migration.hash, &migration.Timestamp)	
		migrations = append(migrations,migration)
	}
	return migrations
}

func applyMigration(db *sql.DB, idx int, migration *Migration, dryRun bool, logger *log.Logger) error {
	tx,err := db.Begin()
	if err != nil {
		return errors.New("Could not start transaction: " + err.Error())
	}
	exec := &executor{tx, verifier{}, dryRun, logger}
	err = migration.Code(exec)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("Migration (%d) - %s returned an error: %s",idx, migration.Name, err.Error())
	}
	_,err = tx.Exec(fmt.Sprintf("INSERT INTO %s (id,name,hash) VALUES (%d,'%s','%s');",tableName, idx, migration.Name, exec.verifier.Hash()))
	if err != nil || dryRun {
		tx.Rollback()
		return err
	} 
	tx.Commit()
	return nil
}

func verifyMigration(applied Migration, pending Migration) error {
	if applied.Name != pending.Name {
		return fmt.Errorf("Migration \"%s\" names changed: current:\"%s\" != applied:\"%s\"", pending.Name, pending.Name, applied.Name)
	}
	v := &verifier{}
	pending.Code(v)
	if v.Hash() != applied.hash {
		return fmt.Errorf("Migration \"%s\" hash changed", pending.Name)
	}
	return nil
}

type executor struct {
	tx *sql.Tx
	verifier verifier
	dryrun bool
	log *log.Logger
}

func (e *executor) Exec(sql string) error {
	e.verifier.Exec(sql)
	if !e.dryrun {
		_, err := e.tx.Exec(sql)
		if err != nil {
			e.tx.Rollback()
		}
		return err
	} 
	e.log.Println("   -> " + sql)
	return nil
}

type verifier struct {
	buffer bytes.Buffer
}

func (v *verifier)Exec(sql string) error {
	v.buffer.WriteString(sql)
	return nil
}

func (v *verifier) Hash() string {
	sum := sha256.Sum256([]byte(v.buffer.String()))
	return fmt.Sprintf("%x", sum)
}