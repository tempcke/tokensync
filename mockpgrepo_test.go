package tk_test

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4/pgxpool"
	tokensync "github.com/tempcke/tk"
	"math/rand"
)

type fakePgRepo struct {
	pool       *pgxpool.Pool
	table      string
	lockNumber int
}

func newFakePgRepo(pool *pgxpool.Pool) (*fakePgRepo, error) {
	if pool == nil {
		return nil, errors.New("pool is nil")
	}
	repo := &fakePgRepo{
		pool:       pool,
		table:      tokensync.DefaultTable,
		lockNumber: rand.Intn(100),
	}

	return repo, repo.up()
}

func (r *fakePgRepo) GetToken(ctx context.Context) (tokensync.Token, error) {
	var (
		token  fakeToken
		_query = `SELECT ` + tokensync.Rows +
			` FROM ` + r.table +
			` ORDER BY ` + tokensync.FieldCreated +
			` DESC LIMIT 1;`
	)
	if err := r.pool.QueryRow(ctx, _query).Scan(
		&token.val,
		&token.created,
		&token.expires,
	); err != nil {
		return nil, err
	}

	return token, nil
}

func (r *fakePgRepo) StoreToken(ctx context.Context, token tokensync.Token) error {
	var _query = `INSERT INTO   ` + r.table +
		` (` + tokensync.Rows + `) ` +
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

func (r *fakePgRepo) Lock(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `SELECT pg_advisory_lock($1);`, r.lockNumber); err != nil {
		return err
	}

	return nil
}

func (r *fakePgRepo) Unlock(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `SELECT pg_advisory_unlock($1);`, r.lockNumber); err != nil {
		return err
	}

	return nil
}

func (r *fakePgRepo) withTable(table string) *fakePgRepo {
	_ = r.down()
	r.table = table
	_ = r.up()
	return r
}

func (r *fakePgRepo) withLockNumber(lockNumber int) *fakePgRepo {
	r.lockNumber = lockNumber
	return r
}

func (r *fakePgRepo) up() error {
	if _, err := r.pool.Exec(
		ctx,
		`CREATE TABLE IF NOT EXISTS `+r.table+` (
					val					varchar												NOT NULL,
					expires				timestamp											NOT NULL,
					created				timestamp		default now()						NOT NULL,
					PRIMARY KEY (val)
				);`,
	); err != nil {
		return err
	}
	return nil
}

func (r *fakePgRepo) down() error {
	if _, err := r.pool.Exec(ctx, `DROP TABLE IF EXISTS `+r.table+`;`); err != nil {
		return err
	}
	return nil
}
