package stackitprovider

import (
	"context"
	"fmt"

	"github.com/antihax/optional"
	stackitdnsclient "github.com/stackitcloud/stackit-dns-api-client-go"
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

// fetchRecords fetches all []stackitdnsclient.DomainRrSet from STACKIT DNS API for given zone id.
func (r *rrSetFetcher) fetchRecords(
	ctx context.Context,
	zoneId string,
	nameFilter *string,
) ([]stackitdnsclient.DomainRrSet, error) {
	var result []stackitdnsclient.DomainRrSet
	queryParams := stackitdnsclient.RecordSetApiV1ProjectsProjectIdZonesZoneIdRrsetsGetOpts{
		Page:     optional.NewInt32(1),
		PageSize: optional.NewInt32(10000),
		ActiveEq: optional.NewBool(true),
	}

	if nameFilter != nil {
		queryParams.NameLike = optional.NewString(*nameFilter)
	}

	rrSetResponse, _, err := r.apiClient.RecordSetApi.V1ProjectsProjectIdZonesZoneIdRrsetsGet(
		ctx,
		r.projectId,
		zoneId,
		&queryParams,
	)
	if err != nil {
		return nil, err
	}
	result = append(result, rrSetResponse.RrSets...)

	queryParams.Page = optional.NewInt32(2)
	for queryParams.Page.Value() <= rrSetResponse.TotalPages {
		rrSetResponse, _, err := r.apiClient.RecordSetApi.V1ProjectsProjectIdZonesZoneIdRrsetsGet(
			ctx,
			r.projectId,
			zoneId,
			&queryParams,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, rrSetResponse.RrSets...)
		queryParams.Page = optional.NewInt32(queryParams.Page.Value() + 1)
	}

	return result, nil
}

// getRRSetForUpdateDeletion returns the record set to be deleted and the zone it belongs to.
func (r *rrSetFetcher) getRRSetForUpdateDeletion(
	ctx context.Context,
	change *endpoint.Endpoint,
	zones []stackitdnsclient.DomainZone,
) (*stackitdnsclient.DomainZone, *stackitdnsclient.DomainRrSet, error) {
	resultZone, found := findBestMatchingZone(change.DNSName, zones)
	if !found {
		r.logger.Error(
			"record set name contains no zone dns name",
			zap.String("name", change.DNSName),
		)

		return nil, nil, fmt.Errorf("record set name contains no zone dns name")
	}

	domainRrSets, err := r.fetchRecords(ctx, resultZone.Id, &change.DNSName)
	if err != nil {
		return nil, nil, err
	}

	resultRRSet, found := findRRSet(change.DNSName, change.RecordType, domainRrSets)
	if !found {
		r.logger.Info("record not found on record sets", zap.String("name", change.DNSName))

		return nil, nil, fmt.Errorf("record not found on record sets")
	}

	return resultZone, resultRRSet, nil
}
