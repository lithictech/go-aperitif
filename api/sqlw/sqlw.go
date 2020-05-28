package sqlw

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
)

// Interface is a common wrapper over sqlx so we can compose functionality.
type Interface interface {
	sqlx.Queryer
	sqlx.QueryerContext
	sqlx.Execer
	sqlx.ExecerContext
	DBX() *sqlx.DB
}

type AddRow func([]interface{})

func CopyFrom(ctx context.Context, db *sql.DB, copyIn string, rowAdder func(cb AddRow)) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := txn.Prepare(copyIn)
	if err != nil {
		return err
	}
	rowAdder(func(i []interface{}) {
		if _, e := stmt.ExecContext(ctx, i...); e != nil {
			err = e
		}
	})
	if err != nil {
		return err
	}
	if _, err = stmt.ExecContext(ctx); err != nil {
		return err
	}
	if err := stmt.Close(); err != nil {
		return err
	}
	if err := txn.Commit(); err != nil {
		return err
	}
	return nil
}

// Wrap wraps a real sqlx.DB connection into one that can be composed.
func Wrap(db *sqlx.DB) Interface {
	return &sqlxWrapper{db: db}
}

type sqlxWrapper struct {
	db *sqlx.DB
}

func (s *sqlxWrapper) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

func (s *sqlxWrapper) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	return s.db.Queryx(query, args...)
}

func (s *sqlxWrapper) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	return s.db.QueryRowx(query, args...)
}

func (s *sqlxWrapper) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

func (s *sqlxWrapper) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	return s.db.QueryxContext(ctx, query, args...)
}

func (s *sqlxWrapper) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	return s.db.QueryRowxContext(ctx, query, args...)
}

func (s *sqlxWrapper) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

func (s *sqlxWrapper) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s *sqlxWrapper) DBX() *sqlx.DB {
	return s.db
}
