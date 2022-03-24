package tk_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	tokensync "github.com/tempcke/tk"
)

type fakeClient struct {
	err      error
	curToken *fakeToken
	lag      time.Duration
	reqCount int
}

func (c *fakeClient) NewToken(_ context.Context) (tokensync.Token, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.sleep()
	return c.newFakeToken(), nil
}

func (c *fakeClient) RefreshToken(_ context.Context, _ tokensync.Token) (tokensync.Token, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.sleep()
	return c.newFakeToken(), nil
}

func (c *fakeClient) sleep() {
	if c.lag > 0 {
		time.Sleep(c.lag)
	}
}

func (c *fakeClient) newFakeToken() *fakeToken {
	c.reqCount++
	c.curToken = newFakeToken()
	return c.curToken
}

func (c *fakeClient) expireToken() {
	c.curToken.expireToken()
}

func (c *fakeClient) withLag(lag time.Duration) *fakeClient {
	c.lag = lag
	return c
}

type fakeToken struct {
	val     string
	created time.Time
	expires time.Time
	err     error
}

func newFakeToken() *fakeToken {
	return &fakeToken{
		val:     uuid.NewString(),
		created: time.Now(),
		expires: time.Now().Add(time.Minute),
	}
}

func (t fakeToken) String() string {
	return t.val
}

func (t fakeToken) Created() time.Time {
	return t.created
}

func (t fakeToken) Expires() time.Time {
	return t.expires
}

func (t fakeToken) Validate() error {
	return t.err
}

func (t *fakeToken) expireToken() {
	t.expires = time.Now().Add(-2 * time.Minute)
}

type pod = struct {
	client *fakeClient
	keeper *tokensync.TokenKeeper
}

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

func (r fakeRepo) sleep() {
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