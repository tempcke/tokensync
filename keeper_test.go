package tk_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tokensync "github.com/tempcke/tk"
)

var ctx = context.Background()

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
		client.curToken.err = errors.New("anything " + uuid.NewString())

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
		var tokens = make([]tokensync.Token, numCalls)
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
		var tokens = make([]tokensync.Token, numCalls)
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

func TestTokenKeeper_SharedToken(t *testing.T) {
	// suppose we have an external api which when you request a new token it makes all others for you invalid
	// if we have 2 instances of our app running and both need to use that api then they
	// will need to share the same access token
	// to accomplish this the token needs to exist in a shared location such as postgres or redis or... etc

	t.Run("get token from repo", func(t *testing.T) {
		// when the keeper doesn't have a token, before telling the client to make a new one
		// it should check the repo for a valid token

		repo := &fakeRepo{}
		_ = repo.StoreToken(ctx, newFakeToken())

		client := &fakeClient{}
		keeper := tokensync.NewTokenKeeper(client).WithRepo(repo)
		assert.Equal(t, repo.token().String(), keeper.Token().String())
		assert.Equal(t, 0, client.reqCount)
	})

	t.Run("keeper stores new tokens into repo", func(t *testing.T) {
		var (
			repo   = new(fakeRepo)
			client = new(fakeClient)
			keeper = tokensync.NewTokenKeeper(client).WithRepo(repo)
		)
		tok := keeper.Token() // new token from client
		require.NotNil(t, tok)
		require.NotNil(t, repo.token)
		assert.Equal(t, tok.String(), repo.token().String())
	})

	t.Run("repo token is expired", func(t *testing.T) {
		// if the repo token is expired or invalid then keeper should
		// get a new token from the client, then update the repo with new token

		origRepoToken := newFakeToken()
		origRepoToken.expireToken()
		repo := &fakeRepo{}
		_ = repo.StoreToken(ctx, origRepoToken)

		client := &fakeClient{}
		keeper := tokensync.NewTokenKeeper(client).WithRepo(repo)

		fetchedToken := keeper.Token()

		// assert we didn't get the expired token
		assert.NotEqual(t, origRepoToken.String(), fetchedToken.String())

		// assert new token from client stored into repo
		assert.Equal(t, repo.token().String(), fetchedToken.String())
		require.NoError(t, fetchedToken.Validate())
	})

	t.Run("repo lock blocks keeper until token ready", func(t *testing.T) {
		// suppose we deploy the app in 2 pods
		// neither will have a token, but will need one, and they do not share memory
		// only ONE may fetch a token from the client...

		var (
			ctx    = context.Background()
			lag    = 50 * time.Millisecond
			client = new(fakeClient).withLag(lag)
			repo   = new(fakeRepo).withLag(lag)
			token  = newFakeToken()

			fetchedToken tokensync.Token
			wg           sync.WaitGroup
		)

		// lock the repo as though another process is updating the token
		repo.Lock()

		keeper := tokensync.NewTokenKeeper(client).WithRepo(repo)

		wg.Add(1)
		go func() {
			defer wg.Done()
			fetchedToken = keeper.Token()
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			fetchedToken = keeper.Token()
		}()

		// store the token in repo with lag after the keeper has requested it
		require.NoError(t, repo.StoreToken(ctx, token))
		// unlock the repo, this should now allow the keeper.Token() call to get this token
		repo.Unlock()
		wg.Wait()

		// assert token not fetched from client and that the fetchedToken is correct
		require.Equal(t, 0, client.reqCount)
		require.Equal(t, token.String(), fetchedToken.String())

		// assert repo token was not changed
		require.Equal(t, token.String(), repo.token().String())
	})

	t.Run("two processes startup at the same time", func(t *testing.T) {
		// suppose we deploy the app in 2 pods
		// neither will have a token, but will need one, and they do not share memory
		// only ONE may fetch a token from the client...

		var (
			lag = 50 * time.Millisecond

			dataStore = new(storage)

			client1 = new(fakeClient).withLag(lag)
			client2 = new(fakeClient).withLag(lag)

			repo1 = new(fakeRepo).withLag(lag).withStorage(dataStore)
			repo2 = new(fakeRepo).withLag(lag).withStorage(dataStore)

			keeper1 = tokensync.NewTokenKeeper(client1).WithRepo(repo1)
			keeper2 = tokensync.NewTokenKeeper(client2).WithRepo(repo2)

			wg sync.WaitGroup
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = keeper1.Token()
		}()
		go func() {
			defer wg.Done()
			_ = keeper2.Token()
		}()
		wg.Wait()

		// both keepers should have the same token
		require.Equal(t, keeper1.Token().String(), keeper2.Token().String())

		// repo token and keeper token should match
		require.Equal(t, keeper1.Token().String(), repo1.token().String())

		// only 1 request for a token should be made total
		assert.Equal(t, 1, client1.reqCount+client2.reqCount)
	})
}

func TestTokenKeeper_multiProcess(t *testing.T) {
	type pod = struct {
		client *fakeClient
		keeper *tokensync.TokenKeeper
	}

	// now suppose you have 4 pods running and each time you get a token the service invalidates the previous
	// this means each pod needs the current token and must avoid race conditions over many processes which
	// do not share memory...

	// 20ms is enough to prove this, but manually change to 2000 to check it if you want
	var lag = 20 * time.Millisecond

	var processCount = 4 // 4 pods in cluster

	var dataStore = new(storage)

	// init the pods with their own client and keeper
	var pods = make([]*pod, processCount)
	for i := range pods {
		client := &fakeClient{}
		client.lag = lag
		repo := new(fakeRepo).withLag(lag).withStorage(dataStore)
		pods[i] = &pod{
			client: client,
			keeper: tokensync.NewTokenKeeper(client).WithRepo(repo),
		}
	}

	// get a token from each pod at the same time
	var wg sync.WaitGroup
	tokens := make([]tokensync.Token, len(pods))
	for i, p := range pods {
		wg.Add(1)
		go func(i int, p *pod) {
			defer wg.Done()
			tokens[i] = p.keeper.Token()
		}(i, p)
	}
	wg.Wait()

	// assert all tokens from each pod match
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

func TestTokenKeeper_fallback(t *testing.T) {
	// test lock TTL, what happens if a process locks the repo and then dies?
	// the keeper should be able to realize the lock is held for to long and take over

	// test keep-alive, rather than waiting until a token fails to replace it
	// establish rules for replacing it before it expires
	// ensure that multiple pods do not try to replace it at the same time
}
