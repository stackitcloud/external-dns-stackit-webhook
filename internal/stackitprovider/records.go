package stackitprovider

import (
	"context"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/provider"
)

// Records returns resource records.
func (d *StackitDNSProvider) Records(ctx context.Context) ([]*endpoint.Endpoint, error) {
	zones, err := d.zoneFetcherClient.zones(ctx)
	if err != nil {
		return nil, err
	}

	var endpoints []*endpoint.Endpoint
	endpointsErrorChannel := make(chan endpointError, len(zones))
	zoneIdsChannel := make(chan string, len(zones))

	for i := 0; i < d.workers; i++ {
		go d.fetchRecordsWorker(ctx, zoneIdsChannel, endpointsErrorChannel)
	}

	for i := range zones {
		zone := &zones[i]
		if zone.Id == nil {
			continue
		}
		zoneIdsChannel <- *zone.Id
	}

	for i := 0; i < len(zones); i++ {
		endpointsErrorList := <-endpointsErrorChannel
		if endpointsErrorList.err != nil {
			close(zoneIdsChannel)

			return nil, endpointsErrorList.err
		}
		endpoints = append(endpoints, endpointsErrorList.endpoints...)
	}

	close(zoneIdsChannel)

	return endpoints, nil
}

// fetchRecordsWorker fetches all records from a given zone.
func (d *StackitDNSProvider) fetchRecordsWorker(
	ctx context.Context,
	zoneIdChannel chan string,
	endpointsErrorChannel chan<- endpointError,
) {
	for zoneId := range zoneIdChannel {
		d.processZoneRRSets(ctx, zoneId, endpointsErrorChannel)
	}

	d.logger.Debug("fetch record set worker finished")
}

// processZoneRRSets fetches and processes DNS records for a given zone.
func (d *StackitDNSProvider) processZoneRRSets(
	ctx context.Context,
	zoneId string,
	endpointsErrorChannel chan<- endpointError,
) {
	var endpoints []*endpoint.Endpoint
	rrSets, err := d.rrSetFetcherClient.fetchRecords(ctx, zoneId, nil)
	if err != nil {
		endpointsErrorChannel <- endpointError{
			endpoints: nil,
			err:       err,
		}

		return
	}

	endpoints = d.collectEndPoints(rrSets)
	endpointsErrorChannel <- endpointError{
		endpoints: endpoints,
		err:       nil,
	}
}

// collectEndPoints creates a list of Endpoints from the provided rrSets.
func (d *StackitDNSProvider) collectEndPoints(
	rrSets []stackitdnsclient.RecordSet,
) []*endpoint.Endpoint {
	var endpoints []*endpoint.Endpoint

	for i := range rrSets {
		r := &rrSets[i]

		name, recordType, ttl, records, ok := recordSetCoreFields(r)
		if !ok || !provider.SupportedRecordType(recordType) {
			continue
		}

		endpoints = append(endpoints, endpointsFromRecords(name, recordType, ttl, records)...)
	}

	return endpoints
}

func recordSetCoreFields(r *stackitdnsclient.RecordSet) (name string, recordType string, ttl endpoint.TTL, records []stackitdnsclient.Record, ok bool) {
	if r == nil || r.Type == nil || r.Name == nil || r.Ttl == nil || r.Records == nil {
		return "", "", 0, nil, false
	}

	return *r.Name, string(*r.Type), endpoint.TTL(*r.Ttl), *r.Records, true
}

func endpointsFromRecords(name, recordType string, ttl endpoint.TTL, records []stackitdnsclient.Record) []*endpoint.Endpoint {
	endpoints := make([]*endpoint.Endpoint, 0, len(records))

	for i := range records {
		rec := &records[i]
		if rec.Content == nil {
			continue
		}

		endpoints = append(endpoints, endpoint.NewEndpointWithTTL(name, recordType, ttl, *rec.Content))
	}

	return endpoints
}
