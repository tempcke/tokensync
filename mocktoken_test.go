package tk_test

import (
	"time"

	"github.com/google/uuid"
)

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

