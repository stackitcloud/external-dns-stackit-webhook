package stackit

import (
	"fmt"
	"net/http"
	"time"

	stackitconfig "github.com/stackitcloud/stackit-sdk-go/core/config"
)

// SetConfigOptions sets the default config options for the STACKIT
// client and determines which type of authorization to use, depending on the
// passed bearerToken and keyPath parameters. If no baseURL or an invalid
// combination of auth options is given (neither or both), the function returns
// an error.
func SetConfigOptions(baseURL, bearerToken, keyPath string) ([]stackitconfig.ConfigurationOption, error) {
	if len(baseURL) == 0 {
		return nil, fmt.Errorf("base-url is required")
	}

	options := []stackitconfig.ConfigurationOption{
		stackitconfig.WithHTTPClient(&http.Client{
			Timeout: 10 * time.Second,
		}),
		stackitconfig.WithEndpoint(baseURL),
	}

	bearerTokenSet := len(bearerToken) > 0
	keyPathSet := len(keyPath) > 0

	if (!bearerTokenSet && !keyPathSet) || (bearerTokenSet && keyPathSet) {
		return nil, fmt.Errorf("exactly only one of auth-token or auth-key-path is required")
	}

	if bearerTokenSet {
		return append(options, stackitconfig.WithToken(bearerToken)), nil
	}

	return append(options, stackitconfig.WithServiceAccountKeyPath(keyPath)), nil
}
