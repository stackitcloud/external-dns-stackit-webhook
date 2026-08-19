package stackit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMissingBaseURL(t *testing.T) {
	t.Parallel()
	cfg := WebhookAuthConfig{}
	options, err := SetConfigOptions(&cfg)
	assert.ErrorContains(t, err, "base-url")
	assert.Nil(t, options)
}

func TestNoAuthOptionsSet_FallsBackToDefaultAuth(t *testing.T) {
	t.Parallel()
	cfg := WebhookAuthConfig{BaseURL: "https://example.com"}
	options, err := SetConfigOptions(&cfg)
	assert.NoError(t, err)
	assert.Len(t, options, 3)
}

func TestMultipleAuthOptionsSet_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := WebhookAuthConfig{
		BaseURL:    "https://example.com",
		KeyPath:    "key/path",
		WIFEnabled: true,
	}
	options, err := SetConfigOptions(&cfg)
	assert.ErrorContains(t, err, "ambiguous authentication configuration")
	assert.Nil(t, options)
}

func TestKeyPathSet(t *testing.T) {
	t.Parallel()
	cfg := WebhookAuthConfig{
		BaseURL: "https://example.com",
		KeyPath: "key/path",
	}
	options, err := SetConfigOptions(&cfg)
	assert.NoError(t, err)
	assert.Len(t, options, 4)
}

func TestWIFSet_WithoutTokenPath(t *testing.T) {
	t.Parallel()
	cfg := WebhookAuthConfig{
		BaseURL:    "https://example.com",
		WIFEnabled: true,
	}
	options, err := SetConfigOptions(&cfg)
	assert.NoError(t, err)
	assert.Len(t, options, 4)
}

func TestWIFSet_WithTokenPath(t *testing.T) {
	t.Parallel()
	cfg := WebhookAuthConfig{
		BaseURL:      "https://example.com",
		WIFTokenPath: "/var/run/secrets/tokens/stackit-token",
	}
	options, err := SetConfigOptions(&cfg)
	assert.NoError(t, err)
	assert.Len(t, options, 5)
}

func TestKeyPathAndURLSet(t *testing.T) {
	t.Parallel()
	cfg := WebhookAuthConfig{
		BaseURL:  "https://example.com",
		KeyPath:  "key/path",
		TokenURL: "https://alternative.url.stackit.cloud/token",
	}
	options, err := SetConfigOptions(&cfg)
	assert.NoError(t, err)
	assert.Len(t, options, 5)
}
