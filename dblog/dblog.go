// Package dblog adds a DB logging driver for sqlx.
package dblog

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"github.com/lithictech/go-aperitif/logctx"
	"github.com/sirupsen/logrus"
)

func New(db *sqlx.DB, defaultLogger *logrus.Entry) *DBLogger {
	if db == nil {
		panic("must provide db")
	}
	if defaultLogger == nil {
		panic("must provide logger")
	}
	return &DBLogger{
		defaultLogger: defaultLogger,
		DB:            db,
	}
}

type DBLogger struct {
	defaultLogger *logrus.Entry
	DB            *sqlx.DB
}

func (p *DBLogger) logger(ctx context.Context) *logrus.Entry {
	if ctx == nil {
		return p.defaultLogger
	}
	logger := logctx.LoggerOrNil(ctx)
	if logger != nil {
		return logger
	}
	return p.defaultLogger
}

func (p *DBLogger) log(ctx context.Context, cmd, q string, args []interface{}) {
	logger := p.logger(ctx)
	logger.WithFields(logrus.Fields{
		"sql_statement": q,
		"sql_args":      args,
	}).Debug("sql_" + cmd)
}

func (p *DBLogger) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	p.log(ctx, "exec", query, args)
	return p.DB.ExecContext(ctx, query, args...)
}

func (p *DBLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	p.log(nil, "exec", query, args)
	return p.DB.Exec(query, args...)
}

func (p *DBLogger) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	p.log(ctx, "query", query, args)
	return p.DB.QueryContext(ctx, query, args...)
}

func (p *DBLogger) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	p.log(ctx, "queryx", query, args)
	return p.DB.QueryxContext(ctx, query, args...)
}

func (p *DBLogger) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	p.log(ctx, "queryxrow", query, args)
	return p.DB.QueryRowxContext(ctx, query, args...)
}

func (p *DBLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	p.log(nil, "query", query, args)
	return p.DB.Query(query, args...)
}

func (p *DBLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	p.log(nil, "queryx", query, args)
	return p.DB.Queryx(query, args...)
}

func (p *DBLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	p.log(nil, "queryxrow", query, args)
	return p.DB.QueryRowx(query, args...)
}

var _ sqlx.Queryer = &DBLogger{}
var _ sqlx.QueryerContext = &DBLogger{}
var _ sqlx.Execer = &DBLogger{}
var _ sqlx.ExecerContext = &DBLogger{}

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
