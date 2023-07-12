package stackitprovider

import (
	"errors"
	"strings"

	stackitdnsclient "github.com/stackitcloud/stackit-dns-api-client-go"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
)

// findBestMatchingZone finds the best matching zone for a given record set name. The criteria are
// that the zone name is contained in the record set name and that the zone name is the longest
// possible match. Eg foo.bar.com. would have prejudice over bar.com. if rr set name is foo.bar.com.
func findBestMatchingZone(
	rrSetName string,
	zones []stackitdnsclient.DomainZone,
) (*stackitdnsclient.DomainZone, bool) {
	count := 0
	var domainZone stackitdnsclient.DomainZone
	for _, zone := range zones {
		if len(zone.DnsName) > count && strings.Contains(rrSetName, zone.DnsName) {
			count = len(zone.DnsName)
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
	rrSets []stackitdnsclient.DomainRrSet,
) (*stackitdnsclient.DomainRrSet, bool) {
	for _, rrSet := range rrSets {
		if rrSet.Name == rrSetName && rrSet.Type_ == recordType {
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

// getStackitRRSetRecordPost returns a stackitdnsclient.RrsetRrSetPost from a change for the api client.
func getStackitRRSetRecordPost(change *endpoint.Endpoint) stackitdnsclient.RrsetRrSetPost {
	records := make([]stackitdnsclient.RrsetRecordPost, len(change.Targets))
	for i, target := range change.Targets {
		records[i] = stackitdnsclient.RrsetRecordPost{
			Content: target,
		}
	}

	return stackitdnsclient.RrsetRrSetPost{
		Name:    change.DNSName,
		Records: records,
		Ttl:     int32(change.RecordTTL),
		Type_:   change.RecordType,
	}
}

// getStackitRRSetRecordPatch returns a stackitdnsclient.RrsetRrSetPatch from a change for the api client.
func getStackitRRSetRecordPatch(change *endpoint.Endpoint) stackitdnsclient.RrsetRrSetPatch {
	records := make([]stackitdnsclient.RrsetRecordPost, len(change.Targets))
	for i, target := range change.Targets {
		records[i] = stackitdnsclient.RrsetRecordPost{
			Content: target,
		}
	}

	return stackitdnsclient.RrsetRrSetPatch{
		Name:    change.DNSName,
		Records: records,
		Ttl:     int32(change.RecordTTL),
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

// getSwaggerErrorMessage returns the error message from a swagger error.
func getSwaggerErrorMessage(err error) string {
	message := err.Error()

	var swaggerError stackitdnsclient.GenericSwaggerError
	if errors.As(err, &swaggerError) {
		if v2, ok := swaggerError.Model().(stackitdnsclient.SerializerMessage); ok {
			message = v2.Message
		}
	}

	return message
}
