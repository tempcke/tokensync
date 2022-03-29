package tk_test

import (
	"github.com/stretchr/testify/assert"
	tokensync "github.com/tempcke/tk"
	"os"
	"testing"
)

func TestPgRepo(t *testing.T) {
	var token tokensync.Token

	err := os.Setenv(tokensync.DatabaseURL, "postgresql://actourex@localhost:5432/actourex")
	assert.Nil(t, err)

	pool, err := tokensync.GetPool(ctx)
	assert.Nil(t, err)

	repo, err := newFakePgRepo(pool)
	assert.Nil(t, err)

	defer repo.down()

	t.Run("get token from pg repo storage when there are no tokens", func(t *testing.T) {
		_, err = repo.GetToken(ctx)
		assert.NotNil(t, err)
	})

	t.Run("store token into pg repo storage", func(t *testing.T) {
		err = repo.StoreToken(ctx, newFakeToken())
		assert.Nil(t, err)
	})

	t.Run("get latest valid token from pg repo storage", func(t *testing.T) {
		newToken := newFakeToken()
		err = repo.StoreToken(ctx, newToken)
		assert.Nil(t, err)

		token, err = repo.GetToken(ctx)
		assert.Nil(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, newToken.String(), token.String())
		assert.Equal(t, newToken.Expires(), token.Expires())
		assert.Equal(t, newToken.Created(), token.Created())
	})

	t.Run("successfully lock pg repo storage", func(t *testing.T) {
		err = repo.Lock(ctx)
		assert.Nil(t, err)
	})

	t.Run("successfully unlock pg repo storage", func(t *testing.T) {
		err = repo.Unlock(ctx)
		assert.Nil(t, err)
	})
}
