package dbx

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/gtlang/gt/lib/x/goql"
)

var ErrReadOnly = errors.New("error 1299. Can't write changes in Read-Only mode")

type DB struct {
	*sql.DB
	Driver   string
	DSN      string // used only with sqlite3
	Database string

	Namespace         string
	WriteAnyNamespace bool
	OpenAnyDatabase   bool
	ReadOnly          bool

	mut      *sync.Mutex
	tx       *sql.Tx
	nestedTx int // to keep track of the number of nested transactions
}

func Open(driver, dsn string) (*DB, error) {
	return OpenDatabase("", driver, dsn)
}

// OpenDatabase opens a new database handle.
func OpenDatabase(database, driver, dsn string) (*DB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	dx := &DB{
		Driver:   driver,
		Database: database,
		DB:       db,
		mut:      &sync.Mutex{},
	}

	// save the dns in sqlite3 for schema queries.
	if driver == "sqlite3" {
		dx.DSN = dsn
		dx.Database = ""
	}

	return dx, nil
}

// Open returns a new db handler for the database.
func (db *DB) Open(database string) *DB {
	return &DB{
		Driver:            db.Driver,
		DSN:               db.DSN,
		Database:          database,
		Namespace:         db.Namespace,
		WriteAnyNamespace: db.WriteAnyNamespace,
		OpenAnyDatabase:   db.OpenAnyDatabase,
		ReadOnly:          db.ReadOnly,
		DB:                db.DB,
		mut:               &sync.Mutex{},
	}
}

// Copies the database but without transaction information
func (db *DB) Clone() *DB {
	return &DB{
		Driver:            db.Driver,
		DSN:               db.DSN,
		Database:          db.Database,
		Namespace:         db.Namespace,
		WriteAnyNamespace: db.WriteAnyNamespace,
		OpenAnyDatabase:   db.OpenAnyDatabase,
		ReadOnly:          db.ReadOnly,
		DB:                db.DB,
		mut:               &sync.Mutex{},
	}
}

func (db *DB) HasTransaction() bool {
	db.mut.Lock()
	v := db.tx != nil
	db.mut.Unlock()
	return v
}

func (db *DB) NestedTransactions() int {
	db.mut.Lock()
	v := db.nestedTx
	db.mut.Unlock()
	return v
}

type queryable interface {
	Prepare(query string) (*sql.Stmt, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

func (db *DB) queryable() queryable {
	db.mut.Lock()
	defer db.mut.Unlock()

	if db.tx != nil {
		return db.tx
	}
	return db.DB
}

func (db *DB) TransactionNestLevel() int {
	return db.nestedTx
}

func (db *DB) Begin() error {
	db.mut.Lock()
	defer db.mut.Unlock()

	if db.tx == nil && db.nestedTx > 0 {
		return fmt.Errorf("gOQL: Previous transaction still open")
	}

	if db.tx != nil {
		db.nestedTx++
		return nil
	}

	t, err := db.DB.Begin()
	if err != nil {
		return err
	}

	db.nestedTx++
	db.tx = t
	return nil
}

func (db *DB) Rollback() error {
	db.mut.Lock()
	defer db.mut.Unlock()

	if db.tx != nil {
		err := db.tx.Rollback()
		db.tx = nil
		if err != nil {
			return err
		}
	}

	if db.nestedTx == 0 {
		// there is an error. It should have an nested value
		return fmt.Errorf("no transaction open.3")
	}

	db.nestedTx--
	return nil
}

func (db *DB) CommitForce() error {
	db.mut.Lock()
	defer db.mut.Unlock()

	if db.nestedTx == 0 {
		return fmt.Errorf("no transaction open")
	}

	if db.tx == nil {
		return fmt.Errorf("no transaction open.2")
	}

	err := db.tx.Commit()
	if err != nil {
		return err
	}

	db.tx = nil
	db.nestedTx = 0
	return nil
}

func (db *DB) Commit() error {
	db.mut.Lock()
	defer db.mut.Unlock()

	if db.nestedTx == 0 {
		return fmt.Errorf("no transaction open")
	}

	db.nestedTx--

	if db.nestedTx > 0 {
		return nil
	}

	if db.tx == nil {
		return fmt.Errorf("no transaction open.2")
	}

	err := db.tx.Commit()
	if err != nil {
		return err
	}

	db.tx = nil
	return nil
}

func (db *DB) Prepare(query string) (*Stmt, error) {
	stmt, err := db.queryable().Prepare(query)
	if err != nil {
		return nil, err
	}

	return &Stmt{DB: db, Stmt: stmt, query: query}, nil
}

func (db *DB) ToSql(q goql.Query, params []interface{}) (string, []interface{}, error) {
	s, params, err := goql.ToSql(q, params, db.Database, db.Driver)
	if err != nil {
		return "", nil, err
	}
	return s, params, err
}

func (db *DB) ExecRaw(query string, args ...interface{}) (sql.Result, error) {
	if db.ReadOnly {
		return nil, ErrReadOnly
	}

	q := db.queryable()
	r, err := q.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (db *DB) QueryRaw(query string, args ...interface{}) (*sql.Rows, error) {
	r, err := db.queryable().Query(query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}

func (db *DB) QueryRowRaw(query string, args ...interface{}) *sql.Row {
	return db.queryable().QueryRow(query, args...)
}

func (db *DB) ScanValueRaw(v interface{}, query string, args ...interface{}) error {
	return db.queryable().QueryRow(query, args...).Scan(v)
}
