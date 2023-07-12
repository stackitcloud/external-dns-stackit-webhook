package stackitprovider

import (
	"context"

	"github.com/antihax/optional"
	stackitdnsclient "github.com/stackitcloud/stackit-dns-api-client-go"
	"sigs.k8s.io/external-dns/endpoint"
)

type zoneFetcher struct {
	apiClient    *stackitdnsclient.APIClient
	domainFilter endpoint.DomainFilter
	projectId    string
}

func newZoneFetcher(
	apiClient *stackitdnsclient.APIClient,
	domainFilter endpoint.DomainFilter,
	projectId string,
) *zoneFetcher {
	return &zoneFetcher{
		apiClient:    apiClient,
		domainFilter: domainFilter,
		projectId:    projectId,
	}
}

// zones returns filtered list of stackitdnsclient.DomainZone if filter is set.
func (z *zoneFetcher) zones(ctx context.Context) ([]stackitdnsclient.DomainZone, error) {
	if len(z.domainFilter.Filters) == 0 {
		// no filters, return all zones
		queryParams := stackitdnsclient.ZoneApiV1ProjectsProjectIdZonesGetOpts{
			ActiveEq: optional.NewBool(true),
		}
		zones, err := z.fetchZones(ctx, queryParams)
		if err != nil {
			return nil, err
		}

		return zones, nil
	}

	var result []stackitdnsclient.DomainZone
	// send one request per filter
	for _, filter := range z.domainFilter.Filters {
		queryParams := stackitdnsclient.ZoneApiV1ProjectsProjectIdZonesGetOpts{
			DnsNameLike: optional.NewString(filter),
			ActiveEq:    optional.NewBool(true),
		}
		zones, err := z.fetchZones(ctx, queryParams)
		if err != nil {
			return nil, err
		}
		result = append(result, zones...)
	}

	return result, nil
}

// fetchZones fetches all []stackitdnsclient.DomainZone from STACKIT DNS API.
func (z *zoneFetcher) fetchZones(
	ctx context.Context,
	queryParams stackitdnsclient.ZoneApiV1ProjectsProjectIdZonesGetOpts,
) ([]stackitdnsclient.DomainZone, error) {
	var result []stackitdnsclient.DomainZone
	queryParams.Page = optional.NewInt32(1)
	queryParams.PageSize = optional.NewInt32(10000)

	zoneResponse, _, err := z.apiClient.ZoneApi.V1ProjectsProjectIdZonesGet(
		ctx,
		z.projectId,
		&queryParams,
	)
	if err != nil {
		return nil, err
	}
	result = append(result, zoneResponse.Zones...)

	page := int32(2)
	for page <= zoneResponse.TotalPages {
		queryParams.Page = optional.NewInt32(page)
		zoneResponse, _, err := z.apiClient.ZoneApi.V1ProjectsProjectIdZonesGet(
			ctx,
			z.projectId,
			&queryParams,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, zoneResponse.Zones...)
		page++
	}

	return result, nil
}
