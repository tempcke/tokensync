package tk

import (
	"context"
)

type PgRepo struct {}

func (r PgRepo) GetToken(ctx context.Context) (Token, error) {
	panic("implement me")
}

func (r PgRepo) StoreToken(ctx context.Context, token Token) error {
	panic("implement me")
}

func (r PgRepo) Lock() {
	panic("implement me")
}

func (r PgRepo) Unlock() {
	panic("implement me")
}
