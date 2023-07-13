package stackitprovider

import (
	"fmt"
	"net/http"

	stackitdnsclient "github.com/stackitcloud/stackit-dns-api-client-go"
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
	config Config,
	logger *zap.Logger,
	httpClient *http.Client,
) (*StackitDNSProvider, error) {
	configClient := stackitdnsclient.NewConfiguration()

	configClient.DefaultHeader["Authorization"] = fmt.Sprintf("Bearer %s", config.Token)
	configClient.BasePath = config.BasePath
	configClient.HTTPClient = httpClient
	apiClient := stackitdnsclient.NewAPIClient(configClient)

	provider := &StackitDNSProvider{
		apiClient:          apiClient,
		domainFilter:       config.DomainFilter,
		dryRun:             config.DryRun,
		projectId:          config.ProjectId,
		workers:            config.Workers,
		logger:             logger,
		zoneFetcherClient:  newZoneFetcher(apiClient, config.DomainFilter, config.ProjectId),
		rrSetFetcherClient: newRRSetFetcher(apiClient, config.DomainFilter, config.ProjectId, logger),
	}

	return provider, nil
}
