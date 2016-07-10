package sq

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	sqlite3 "github.com/mattn/go-sqlite3"
)

// Getter is generic interface for getting single entity
type Getter interface {
	Get(dest interface{}, query string, args ...interface{}) error
}

// Selector is generic interface for getting multiple enties
type Selector interface {
	Select(dest interface{}, query string, args ...interface{}) error
}

// Execer is generic interface for executing SQL query with no result
type Execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type Database interface {
	Beginx() (Connection, error)
	Getter
	Selector
	Execer
}

type Connection interface {
	Getter
	Selector
	Execer
	Rollback() error
	Commit() error
}

// CastErr inspect given error and replace generic SQL error with easier to
// compare equivalent.
//
// See http://www.postgresql.org/docs/current/static/errcodes-appendix.html
func CastErr(err error) error {
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err, ok := err.(sqlite3.Error); ok {
		switch err.Code {
		case sqlite3.ErrNotFound:
			return ErrNotFound
		case sqlite3.ErrConstraint:
			return ErrConflict
		}
	}
	return err
}

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// sqlxdb wraps sqlx.DB structure and provides custom function notations that
// can be easily mocked. This wrapper is required, because of sqlx.DB's Beginx
// method notation
type sqlxdb struct {
	dbx *sqlx.DB
}

func (x *sqlxdb) Beginx() (Connection, error) {
	return x.dbx.Beginx()
}

func (x *sqlxdb) Get(dest interface{}, query string, args ...interface{}) error {
	return x.dbx.Get(dest, query, args...)
}

func (x *sqlxdb) Select(dest interface{}, query string, args ...interface{}) error {
	return x.dbx.Select(dest, query, args...)
}

func (x *sqlxdb) Exec(query string, args ...interface{}) (sql.Result, error) {
	return x.dbx.Exec(query, args...)
}
