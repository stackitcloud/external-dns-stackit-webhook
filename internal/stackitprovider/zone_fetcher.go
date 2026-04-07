package stackitprovider

import (
	"context"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns/v1api"
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

// zones returns filtered list of stackitdnsclient.Zone if filter is set.
func (z *zoneFetcher) zones(ctx context.Context) ([]stackitdnsclient.Zone, error) {
	if len(z.domainFilter.Filters) == 0 {
		// no filters, return all zones
		zones, err := z.fetchZones(new(z.apiClient.DefaultAPI.ListZones(ctx, z.projectId).ActiveEq(true)))
		if err != nil {
			return nil, err
		}

		return zones, nil
	}

	var result []stackitdnsclient.Zone
	// send one request per filter
	for _, filter := range z.domainFilter.Filters {
		zones, err := z.fetchZones(new(z.apiClient.DefaultAPI.ListZones(ctx, z.projectId).ActiveEq(true).DnsNameLike(filter)))
		if err != nil {
			return nil, err
		}
		result = append(result, zones...)
	}

	return result, nil
}

// fetchZones fetches all []stackitdnsclient.Zone from STACKIT DNS API.
func (z *zoneFetcher) fetchZones(
	listRequest *stackitdnsclient.ApiListZonesRequest,
) ([]stackitdnsclient.Zone, error) {
	var result []stackitdnsclient.Zone
	var pager int32 = 1

	req := listRequest.Page(1).PageSize(10000)

	zoneResponse, err := req.Execute()
	if err != nil {
		return nil, err
	}

	result = append(result, zoneResponse.Zones...)

	// if there is more than one page, we need to loop over the other pages and
	// issue another API request for each one of them
	pager++
	for pager <= zoneResponse.TotalPages {
		zoneResponse, err := req.Page(pager).Execute()
		if err != nil {
			return nil, err
		}
		result = append(result, zoneResponse.Zones...)
		pager++
	}

	return result, nil
}
