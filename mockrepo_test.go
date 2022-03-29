package tk_test

import (
	"context"
	"errors"
	"sync"
	"time"

	tokensync "github.com/tempcke/tk"
)

type fakeRepo struct {
	dataStore *storage
	lag       time.Duration
}

func (r *fakeRepo) GetToken(_ context.Context) (tokensync.Token, error) {
	r.sleep()
	if r.storage().token == nil {
		return nil, errors.New("no token")
	}
	return r.storage().token, nil
}

func (r *fakeRepo) StoreToken(_ context.Context, token tokensync.Token) error {
	r.sleep()
	r.storage().token = &fakeToken{
		val:     token.String(),
		created: token.Created(),
		expires: token.Expires(),
		err:     token.Validate(),
	}
	return nil
}

func (r *fakeRepo) Lock(ctx context.Context) error   { r.sleep(); return r.storage().Lock(ctx) }
func (r *fakeRepo) Unlock(ctx context.Context) error { r.sleep(); return r.storage().Unlock(ctx) }

func (r *fakeRepo) storage() *storage {
	if r.dataStore == nil {
		r.dataStore = &storage{}
	}
	return r.dataStore
}

func (r *fakeRepo) sleep() {
	if r.lag > 0 {
		time.Sleep(r.lag)
	}
}

func (r *fakeRepo) withLag(lag time.Duration) *fakeRepo {
	r.lag = lag
	return r
}

func (r *fakeRepo) withStorage(s *storage) *fakeRepo {
	r.dataStore = s
	return r
}

func (r *fakeRepo) token() *fakeToken {
	return r.storage().token
}

type storage struct {
	token *fakeToken
	lock  sync.Mutex
}

func (s *storage) Lock(_ context.Context) error   { s.lock.Lock(); return nil }
func (s *storage) Unlock(_ context.Context) error { s.lock.Unlock(); return nil }
