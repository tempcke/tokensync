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
	lag   time.Duration
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

func (r *fakeRepo) Lock()   { r.sleep(); r.storage().Lock() }
func (r *fakeRepo) Unlock() { r.sleep(); r.storage().Unlock() }

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
	lock sync.Mutex
}

func (s *storage) Lock() { s.lock.Lock() }
func (s *storage) Unlock() { s.lock.Unlock() }
