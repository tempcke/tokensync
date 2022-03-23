package tokensync

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	ErrClientNewTokenFailed     = errors.New("client.NewToken failed")
	ErrClientRefreshTokenFailed = errors.New("client.RefreshToken failed")
)

type Token interface {
	String() string
	Created() time.Time
	Expires() time.Time
	Validate() error
}

type Client interface {
	NewToken(ctx context.Context) (Token, error)
	RefreshToken(ctx context.Context, t Token) (Token, error)
}

type TokenKeeper struct {
	ctx    context.Context
	client Client
	token  Token
	logger logrus.FieldLogger
	lock   sync.Mutex
}

func NewTokenKeeper(client Client) TokenKeeper {
	log := logrus.New()
	log.Out = ioutil.Discard
	return TokenKeeper{
		client: client,
		logger: log,
	}
}

func (k TokenKeeper) WithLogger(logger logrus.FieldLogger) TokenKeeper {
	k.logger = logger.WithField("TokenKeeper", "TokenKeeper")
	return k
}

func (k *TokenKeeper) Token() Token {
	k.lock.Lock()
	defer k.lock.Unlock()
	if k.token == nil {
		tok, err := k.client.NewToken(k.ctx)
		if err != nil {
			k.logError(err, ErrClientNewTokenFailed.Error())
			err = fmt.Errorf("%w: %s", ErrClientNewTokenFailed, err)
			return newInvalidToken(err)
		}
		k.token = tok
	}
	if err := k.validateToken(); err != nil {
		k.logger.WithField("token", k.token).
			WithError(err).Warn("token invalid")
		tok, err := k.client.RefreshToken(k.ctx, k.token)
		if err != nil {
			k.logError(err, ErrClientRefreshTokenFailed.Error())
			err = fmt.Errorf("%w: %s", ErrClientRefreshTokenFailed, err)
			return newInvalidToken(err)
		}
		k.token = tok
	}
	return k.token
}

func (k TokenKeeper) validateToken() error {
	if err := k.token.Validate(); err != nil {
		return err
	}
	if k.token.Expires().Before(time.Now()) {
		return errors.New("token is expired")
	}
	return nil
}

func (k TokenKeeper) logError(err error, msg string) {
	if k.logger != nil {
		k.logger.WithError(err).Error(msg)
	}
}

type invalidToken struct {
	err     error
	created time.Time
}

func newInvalidToken(err error) Token {
	return invalidToken{err: err, created: time.Now()}
}

func (t invalidToken) String() string     { return "" }
func (t invalidToken) Created() time.Time { return t.created }
func (t invalidToken) Expires() time.Time { return t.created }
func (t invalidToken) Validate() error    { return t.err }
