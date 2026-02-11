package stackitprovider

import (
	"strings"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
)

// findBestMatchingZone finds the best matching zone for a given record set name. The criteria are
// that the zone name is contained in the record set name and that the zone name is the longest
// possible match. Eg foo.bar.com. would have precedence over bar.com. if rr set name is foo.bar.com.
func findBestMatchingZone(
	rrSetName string,
	zones []stackitdnsclient.Zone,
) (*stackitdnsclient.Zone, bool) {
	count := 0
	var domainZone stackitdnsclient.Zone
	for _, zone := range zones {
		if len(*zone.DnsName) > count && strings.Contains(rrSetName, *zone.DnsName) {
			count = len(*zone.DnsName)
			domainZone = zone
		}
	}

	if count == 0 {
		return nil, false
	}

	return &domainZone, true
}

// findRRSet finds a record set by name and type in a list of record sets.
func findRRSet(
	rrSetName, recordType string,
	rrSets []stackitdnsclient.RecordSet,
) (*stackitdnsclient.RecordSet, bool) {
	for _, rrSet := range rrSets {
		if *rrSet.Name == rrSetName && string(*rrSet.Type) == recordType {
			return &rrSet, true
		}
	}

	return nil, false
}

// appendDotIfNotExists appends a dot to the end of a string if it doesn't already end with a dot.
func appendDotIfNotExists(s string) string {
	if !strings.HasSuffix(s, ".") {
		return s + "."
	}

	return s
}

// modifyChange modifies a change to ensure it is valid for this stackitprovider.
func modifyChange(change *endpoint.Endpoint) {
	change.DNSName = appendDotIfNotExists(change.DNSName)

	if change.RecordTTL == 0 {
		change.RecordTTL = 300
	}
}

// getStackitRecordSetPayload returns a stackitdnsclient.RecordSetPayload from a change for the api client.
func getStackitRecordSetPayload(change *endpoint.Endpoint) stackitdnsclient.CreateRecordSetPayload {
	records := make([]stackitdnsclient.RecordPayload, len(change.Targets))
	for i := range change.Targets {
		records[i] = stackitdnsclient.RecordPayload{
			Content: &change.Targets[i],
		}
	}

	return stackitdnsclient.CreateRecordSetPayload{
		Name:    &change.DNSName,
		Records: &records,
		Ttl:     pointerTo(int64(change.RecordTTL)),
		Type:    (stackitdnsclient.CreateRecordSetPayloadGetTypeAttributeType)(&change.RecordType),
	}
}

// getStackitPartialUpdateRecordSetPayload returns a stackitdnsclient.PartialUpdateRecordSetPayload from a change for the api client.
func getStackitPartialUpdateRecordSetPayload(change *endpoint.Endpoint) stackitdnsclient.PartialUpdateRecordSetPayload {
	records := make([]stackitdnsclient.RecordPayload, len(change.Targets))
	for i := range change.Targets {
		records[i] = stackitdnsclient.RecordPayload{
			Content: &change.Targets[i],
		}
	}

	return stackitdnsclient.PartialUpdateRecordSetPayload{
		Name:    &change.DNSName,
		Records: &records,
		Ttl:     pointerTo(int64(change.RecordTTL)),
	}
}

// getLogFields returns a log.Fields object for a change.
func getLogFields(change *endpoint.Endpoint, action string, id string) []zap.Field {
	return []zap.Field{
		zap.String("record", change.DNSName),
		zap.String("content", strings.Join(change.Targets, ",")),
		zap.String("type", change.RecordType),
		zap.String("action", action),
		zap.String("id", id),
	}
}

// pointerTo returns a pointer to the given value.
func pointerTo[T any](v T) *T {
	return &v
}
