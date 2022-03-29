package tk

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4/pgxpool"
	"math/rand"
)

const (
	DefaultTable = "token"
	Rows         = `val, expires, created`
	FieldCreated = `created`
)

type PgRepo struct {
	pool       *pgxpool.Pool
	table      string
	lockNumber int
}

func NewPgRepo(pool *pgxpool.Pool) (*PgRepo, error) {
	if pool == nil {
		return nil, errors.New("pool is nil")
	}

	return &PgRepo{
		pool:       pool,
		table:      DefaultTable,
		lockNumber: rand.Intn(100),
	}, nil
}

func (r *PgRepo) GetToken(ctx context.Context) (Token, error) {
	var (
		token  Token
		_query = `SELECT ` + Rows +
			` FROM ` + r.table +
			` ORDER BY ` + FieldCreated +
			` DESC LIMIT 1;`
	)
	if err := r.pool.QueryRow(ctx, _query).Scan(
		&token, //FIXME
	); err != nil {
		return nil, err
	}

	return token, nil
}

func (r *PgRepo) StoreToken(ctx context.Context, token Token) error {
	var _query = `INSERT INTO   ` + r.table +
		` (` + Rows + `) ` +
		`VALUES ($1, $2, $3);`
	if _, err := r.pool.Exec(
		ctx,
		_query,
		token.String(),
		token.Created(),
		token.Expires(),
	); err != nil {
		return err
	}

	return nil
}

func (r *PgRepo) Lock(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `SELECT pg_advisory_lock($1);`, r.lockNumber); err != nil {
		return err
	}

	return nil
}

func (r *PgRepo) Unlock(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `SELECT pg_advisory_unlock($1);`, r.lockNumber); err != nil {
		return err
	}

	return nil
}

func (r *PgRepo) withTable(table string) *PgRepo {
	r.table = table
	return r
}

func (r *PgRepo) withLockNumber(lockNumber int) *PgRepo {
	r.lockNumber = lockNumber
	return r
}
