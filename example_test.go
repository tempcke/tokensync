package tokensync_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tempcke/tokensync"
)

func TestTokenKeeper(t *testing.T) {

	client := &fakeClient{}
	t.Run("should return same token each time", func(t *testing.T) {
		keeper := tokensync.NewTokenKeeper(client)
		tok := keeper.Token()
		assert.NotEmpty(t, tok)

		assert.Equal(t, tok, keeper.Token())
	})

	t.Run("should get new token when expired", func(t *testing.T) {
		keeper := tokensync.NewTokenKeeper(client)
		tokA := keeper.Token()
		a := tokA.String()
		assert.NoError(t, tokA.Validate())
		client.expireToken()
		tokB := keeper.Token()
		assert.NotEqual(t, a, tokB.String())
		assert.NoError(t, tokB.Validate())
	})

	t.Run("should refresh token when token not valid", func(t *testing.T) {
		keeper := tokensync.NewTokenKeeper(client)
		tokA := keeper.Token() // valid token
		client.expireToken()

		tokB := keeper.Token()
		assert.NotEqual(t, tokA, tokB)
		assert.NoError(t, tokB.Validate())
	})
}

func TestTokenKeeper_concurrent(t *testing.T) {
	// we want to ensure that if many calls are made to get a token when the token is expired
	// that only 1 call will be made to refresh it and other calls wait for the one to get the token

	// how many concurrent token calls? make it high enough to be super confident but not insane :)
	var numCalls = 100000

	// 20ms is enough to prove this, but manually change to 2000 to check it if you want
	var lag = 20 * time.Millisecond

	t.Run("first token", func(t *testing.T) {
		client := &fakeClient{}
		client.lag = lag // slow it down

		keeper := tokensync.NewTokenKeeper(client)

		var wg sync.WaitGroup
		var tokens = make([]tokensync.Token, numCalls, numCalls)
		for i := 0; i < numCalls; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				tokens[i] = keeper.Token()
			}(i) // pass i in or else bad things happen :)
		}
		wg.Wait()
		for i := 1; i < numCalls; i++ {
			require.Equal(t, tokens[0].String(), tokens[i].String())
		}

		// one for first token
		require.Equal(t, 1, client.reqCount)
	})

	t.Run("refresh tokens", func(t *testing.T) {
		client := &fakeClient{}
		keeper := tokensync.NewTokenKeeper(client)

		// get first token, expire it, then slow down the client
		firstToken := keeper.Token()
		require.NoError(t, firstToken.Validate())
		client.expireToken()
		client.lag = lag // slow it down

		var wg sync.WaitGroup
		var tokens = make([]tokensync.Token, numCalls, numCalls)
		for i := 0; i < numCalls; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				tokens[i] = keeper.Token()
			}(i) // pass i in or else bad things happen :)
		}
		wg.Wait()
		for i := 1; i < numCalls; i++ {
			require.Equal(t, tokens[0].String(), tokens[i].String())
		}

		// one for first token, one for refresh token
		require.Equal(t, 2, client.reqCount)
	})
}

func TestTokenKeeper_multiProcess(t *testing.T) {
	// now suppose you have 4 pods running and each time you get a token the service invalidates the previous
	// this means each pod needs the current token and must avoid race conditions over many processes which
	// do not share memory...

	// FIXME: implement and remove skip
	t.Skip() // not yet implemented

	// 20ms is enough to prove this, but manually change to 2000 to check it if you want
	var lag = 20 * time.Millisecond

	var processCount = 4 // 4 pods in cluster

	var pods = make(cluster, processCount)
	for i, _ := range pods {
		client := &fakeClient{}
		client.lag = lag
		pods[i] = &pod{
			client: client,
			keeper: tokensync.NewTokenKeeper(client),
		}
	}

	// require all tokens equal by asserting that all tokens equal the first
	tokens := pods.Token()
	for i := 1; i < len(tokens); i++ {
		require.Equal(t, tokens[0].String(), tokens[i].String())
	}

	// the sum of all client calls from all pods should be 1
	callCount := 0
	for _, p := range pods {
		callCount += p.client.reqCount
	}
	require.Equal(t, 1, callCount)
}

func TestTokenKeeper_errors(t *testing.T) {
	t.Run("cant get token from client", func(t *testing.T) {
		client := &fakeClient{}
		client.err = errors.New("unknown error " + uuid.NewString())

		keeper := tokensync.NewTokenKeeper(client)
		tok := keeper.Token()
		assert.ErrorIs(t, tok.Validate(), tokensync.ErrClientNewTokenFailed)
		assert.ErrorContains(t, tok.Validate(), client.err.Error())
	})
	t.Run("cant refresh token", func(t *testing.T) {
		client := &fakeClient{}
		keeper := tokensync.NewTokenKeeper(client)
		_ = keeper.Token() // valid token

		// expire current token
		client.expireToken()

		// force client to return error on RefreshToken call
		client.err = errors.New("unknown error " + uuid.NewString())

		tokB := keeper.Token()
		assert.ErrorIs(t, tokB.Validate(), tokensync.ErrClientRefreshTokenFailed)
		assert.ErrorContains(t, tokB.Validate(), client.err.Error())
	})
}

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
	c.reqCount++
	if c.lag > 0 {
		time.Sleep(c.lag)
	}
}

func (c *fakeClient) newFakeToken() *fakeToken {
	c.curToken = newFakeToken()
	return c.curToken
}

func (c *fakeClient) expireToken() {
	c.curToken.expires = time.Now().Add(-2 * time.Minute)
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

type pod = struct {
	client *fakeClient
	keeper tokensync.TokenKeeper
}
type cluster []*pod

// Token has every pod in the cluster get a token all at the same time
func (c cluster) Token() []tokensync.Token {
	// have ever pod request a new token all at the same time, and expect them all to be equal
	var wg sync.WaitGroup
	tokens := make([]tokensync.Token, len(c))
	for i, p := range c {
		wg.Add(1)
		go func(i int, p *pod) {
			defer wg.Done()
			tokens[i] = p.keeper.Token()
		}(i, p)
	}
	wg.Wait()
	return tokens
}
