package sqlw

import (
	"context"
	"database/sql"
	"github.com/lithictech/go-aperitif/logctx"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// WithLogging adds logging around all calls.
func WithLogging(db Interface, defaultLogger *logrus.Entry) Interface {
	if db == nil {
		panic("must provide db")
	}
	if defaultLogger == nil {
		panic("must provide logger")
	}
	return &dblogger{
		defaultLogger: defaultLogger,
		db:            db,
	}
}

type dblogger struct {
	defaultLogger *logrus.Entry
	db            Interface
}

func (p *dblogger) DBX() *sqlx.DB {
	return p.db.DBX()
}

func (p *dblogger) logger(ctx context.Context) *logrus.Entry {
	if ctx == nil {
		return p.defaultLogger
	}
	logger := logctx.LoggerOrNil(ctx)
	if logger != nil {
		return logger
	}
	return p.defaultLogger
}

func (p *dblogger) log(ctx context.Context, cmd, q string, args []interface{}) {
	logger := p.logger(ctx)
	logger.WithFields(logrus.Fields{
		"sql_statement": q,
		"sql_args":      args,
	}).Debug("sql_" + cmd)
}

func (p *dblogger) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	p.log(ctx, "exec", query, args)
	return p.db.ExecContext(ctx, query, args...)
}

func (p *dblogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	p.log(nil, "exec", query, args)
	return p.db.Exec(query, args...)
}

func (p *dblogger) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	p.log(ctx, "query", query, args)
	return p.db.QueryContext(ctx, query, args...)
}

func (p *dblogger) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	p.log(ctx, "queryx", query, args)
	return p.db.QueryxContext(ctx, query, args...)
}

func (p *dblogger) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	p.log(ctx, "queryxrow", query, args)
	return p.db.QueryRowxContext(ctx, query, args...)
}

func (p *dblogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	p.log(nil, "query", query, args)
	return p.db.Query(query, args...)
}

func (p *dblogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	p.log(nil, "queryx", query, args)
	return p.db.Queryx(query, args...)
}

func (p *dblogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	p.log(nil, "queryxrow", query, args)
	return p.db.QueryRowx(query, args...)
}
