package stackitprovider

import (
	stackitconfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
)

// StackitDNSProvider implements the DNS stackitprovider for STACKIT DNS.
type StackitDNSProvider struct {
	provider.BaseProvider
	projectId          string
	domainFilter       endpoint.DomainFilter
	dryRun             bool
	workers            int
	logger             *zap.Logger
	apiClient          *stackitdnsclient.APIClient
	zoneFetcherClient  *zoneFetcher
	rrSetFetcherClient *rrSetFetcher
}

// NewStackitDNSProvider creates a new STACKIT DNS stackitprovider.
func NewStackitDNSProvider(
	logger *zap.Logger,
	providerConfig Config,
	stackitConfig ...stackitconfig.ConfigurationOption,
) (*StackitDNSProvider, error) {
	apiClient, err := stackitdnsclient.NewAPIClient(stackitConfig...)
	if err != nil {
		return nil, err
	}

	provider := &StackitDNSProvider{
		apiClient:          apiClient,
		domainFilter:       providerConfig.DomainFilter,
		dryRun:             providerConfig.DryRun,
		projectId:          providerConfig.ProjectId,
		workers:            providerConfig.Workers,
		logger:             logger,
		zoneFetcherClient:  newZoneFetcher(apiClient, providerConfig.DomainFilter, providerConfig.ProjectId),
		rrSetFetcherClient: newRRSetFetcher(apiClient, providerConfig.DomainFilter, providerConfig.ProjectId, logger),
	}

	return provider, nil
}
