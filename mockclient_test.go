package tk_test

import (
	"context"
	"time"

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

