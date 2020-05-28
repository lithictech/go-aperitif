package sqlw

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type Interceptor func(context.Context, string, []interface{}) error

// WithInterceptor will call interceptor before each DB call.
// If interceptor returns an error, it will be returned.
// If the DB method does not return an error (like QueryRow), but Interceptor does,
// panic with the error.
// Usually this is used for mocking.
func WithInterceptor(db Interface, interceptor Interceptor) Interface {
	return &dbintercept{
		Interceptor: interceptor,
		DB:          db,
	}
}

type dbintercept struct {
	Interceptor Interceptor
	DB          Interface
}

func (p *dbintercept) DBX() *sqlx.DB {
	return p.DB.DBX()
}

func (p *dbintercept) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if err := p.Interceptor(ctx, query, args); err != nil {
		return nil, err
	}
	return p.DB.ExecContext(ctx, query, args...)
}

func (p *dbintercept) Exec(query string, args ...interface{}) (sql.Result, error) {
	if err := p.Interceptor(nil, query, args); err != nil {
		return nil, err
	}
	return p.DB.Exec(query, args...)
}

func (p *dbintercept) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if err := p.Interceptor(ctx, query, args); err != nil {
		return nil, err
	}
	return p.DB.QueryContext(ctx, query, args...)
}

func (p *dbintercept) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	if err := p.Interceptor(ctx, query, args); err != nil {
		return nil, err
	}
	return p.DB.QueryxContext(ctx, query, args...)
}

func (p *dbintercept) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	if err := p.Interceptor(ctx, query, args); err != nil {
		panic(err)
	}
	return p.DB.QueryRowxContext(ctx, query, args...)
}

func (p *dbintercept) Query(query string, args ...interface{}) (*sql.Rows, error) {
	if err := p.Interceptor(nil, query, args); err != nil {
		return nil, err
	}
	return p.DB.Query(query, args...)
}

func (p *dbintercept) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	if err := p.Interceptor(nil, query, args); err != nil {
		return nil, err
	}
	return p.DB.Queryx(query, args...)
}

func (p *dbintercept) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	if err := p.Interceptor(nil, query, args); err != nil {
		panic(err)
	}
	return p.DB.QueryRowx(query, args...)
}

var _ Interface = &dbintercept{}
