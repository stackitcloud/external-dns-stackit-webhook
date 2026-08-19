package stackit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	stackitconfig "github.com/stackitcloud/stackit-sdk-go/core/config"
)

type AuthType int

const (
	AuthTypeDefault AuthType = iota
	AuthTypeExplicitKey
	AuthTypeExplicitWIF
)

type WebhookAuthConfig struct {
	BaseURL      string
	TokenURL     string
	KeyPath      string
	WIFEnabled   bool
	WIFTokenPath string
}

func determineAuthType(cfg *WebhookAuthConfig) (AuthType, error) {
	var activeTypes []AuthType

	if len(cfg.KeyPath) > 0 {
		activeTypes = append(activeTypes, AuthTypeExplicitKey)
	}
	if cfg.WIFEnabled || len(cfg.WIFTokenPath) > 0 {
		activeTypes = append(activeTypes, AuthTypeExplicitWIF)
	}

	if len(activeTypes) > 1 {
		return AuthTypeDefault, fmt.Errorf("ambiguous authentication configuration: specify at most one of auth-key-path or auth-wif/auth-wif-token-path")
	}

	if len(activeTypes) == 1 {
		return activeTypes[0], nil
	}

	return AuthTypeDefault, nil
}

func SetConfigOptions(cfg *WebhookAuthConfig) ([]stackitconfig.ConfigurationOption, error) {
	if len(cfg.BaseURL) == 0 {
		return nil, fmt.Errorf("base-url is required")
	}

	authType, err := determineAuthType(cfg)
	if err != nil {
		return nil, err
	}

	options := []stackitconfig.ConfigurationOption{
		stackitconfig.WithHTTPClient(&http.Client{
			Timeout: 10 * time.Second,
		}),
		stackitconfig.WithEndpoint(cfg.BaseURL),
		stackitconfig.WithBackgroundTokenRefresh(context.Background()),
	}

	if len(cfg.TokenURL) > 0 {
		options = append(options, stackitconfig.WithTokenEndpoint(cfg.TokenURL))
	}

	switch authType {
	case AuthTypeExplicitKey:
		options = append(options, stackitconfig.WithServiceAccountKeyPath(cfg.KeyPath))
	case AuthTypeExplicitWIF:
		options = append(options, stackitconfig.WithWorkloadIdentityFederationAuth())
		if len(cfg.WIFTokenPath) > 0 {
			options = append(options, stackitconfig.WithWorkloadIdentityFederationPath(cfg.WIFTokenPath))
		}
	case AuthTypeDefault:
	}

	return options, nil
}
