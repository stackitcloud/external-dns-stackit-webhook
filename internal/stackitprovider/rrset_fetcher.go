package stackitprovider

import (
	"context"
	"fmt"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
)

type rrSetFetcher struct {
	apiClient    *stackitdnsclient.APIClient
	domainFilter endpoint.DomainFilter
	projectId    string
	logger       *zap.Logger
}

func newRRSetFetcher(
	apiClient *stackitdnsclient.APIClient,
	domainFilter endpoint.DomainFilter,
	projectId string,
	logger *zap.Logger,
) *rrSetFetcher {
	return &rrSetFetcher{
		apiClient:    apiClient,
		domainFilter: domainFilter,
		projectId:    projectId,
		logger:       logger,
	}
}

// fetchRecords fetches all []stackitdnsclient.RecordSet from STACKIT DNS API for given zone id.
func (r *rrSetFetcher) fetchRecords(
	ctx context.Context,
	zoneId string,
	nameFilter *string,
) ([]stackitdnsclient.RecordSet, error) {
	var result []stackitdnsclient.RecordSet
	var pager int32 = 1

	listRequest := r.apiClient.ListRecordSets(ctx, r.projectId, zoneId).Page(pager).PageSize(10000).ActiveEq(true)

	if nameFilter != nil {
		listRequest = listRequest.NameLike(*nameFilter)
	}

	rrSetResponse, err := listRequest.Execute()
	if err != nil {
		return nil, err
	}

	result = append(result, *rrSetResponse.RrSets...)

	// if there is more than one page, we need to loop over the other pages and
	// issue another API request for each one of them
	pager++
	for int64(pager) <= *rrSetResponse.TotalPages {
		rrSetResponse, err := listRequest.Page(pager).Execute()
		if err != nil {
			return nil, err
		}
		result = append(result, *rrSetResponse.RrSets...)
		pager++
	}

	return result, nil
}

// getRRSetForUpdateDeletion returns the record set to be deleted and the zone it belongs to.
func (r *rrSetFetcher) getRRSetForUpdateDeletion(
	ctx context.Context,
	change *endpoint.Endpoint,
	zones []stackitdnsclient.Zone,
) (*stackitdnsclient.Zone, *stackitdnsclient.RecordSet, error) {
	resultZone, found := findBestMatchingZone(change.DNSName, zones)
	if !found {
		r.logger.Error(
			"record set name contains no zone dns name",
			zap.String("name", change.DNSName),
		)

		return nil, nil, fmt.Errorf("record set name contains no zone dns name")
	}

	domainRRSets, err := r.fetchRecords(ctx, *resultZone.Id, &change.DNSName)
	if err != nil {
		return nil, nil, err
	}

	resultRRSet, found := findRRSet(change.DNSName, change.RecordType, domainRRSets)
	if !found {
		r.logger.Info("record not found on record sets", zap.String("name", change.DNSName))

		return nil, nil, fmt.Errorf("record not found on record sets")
	}

	return resultZone, resultRRSet, nil
}
