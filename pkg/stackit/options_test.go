package stackit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMissingBaseURL(t *testing.T) {
	t.Parallel()

	options, err := SetConfigOptions("", "", "")
	assert.ErrorContains(t, err, "base-url")
	assert.Nil(t, options)
}

func TestBothAuthOptionsMissing(t *testing.T) {
	t.Parallel()

	options, err := SetConfigOptions("https://example.com", "", "")
	assert.ErrorContains(t, err, "auth-token or auth-key-path")
	assert.Nil(t, options)
}

func TestBothAuthOptionsSet(t *testing.T) {
	t.Parallel()

	options, err := SetConfigOptions("https://example.com", "token", "key/path")
	assert.ErrorContains(t, err, "auth-token or auth-key-path")
	assert.Nil(t, options)
}

func TestBearerTokenSet(t *testing.T) {
	t.Parallel()

	options, err := SetConfigOptions("https://example.com", "token", "")
	assert.NoError(t, err)
	assert.Len(t, options, 3)
}

func TestKeyPathSet(t *testing.T) {
	t.Parallel()

	options, err := SetConfigOptions("https://example.com", "", "key/path")
	assert.NoError(t, err)
	assert.Len(t, options, 3)
}
