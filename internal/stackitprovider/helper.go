package stackitprovider

import (
	"math"
	"strings"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns/v1api"
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
	var domainZone *stackitdnsclient.Zone

	for i := range zones {
		zone := &zones[i]
		if l := len(zone.DnsName); l > count && strings.Contains(rrSetName, zone.DnsName) {
			count = l
			domainZone = zone
		}
	}

	if domainZone == nil {
		return nil, false
	}

	return domainZone, true
}

// findRRSet finds a record set by name and type in a list of record sets.
func findRRSet(
	rrSetName, recordType string,
	rrSets []stackitdnsclient.RecordSet,
) (*stackitdnsclient.RecordSet, bool) {
	for i := range rrSets {
		rrSet := &rrSets[i]
		if rrSet.Name == rrSetName && rrSet.Type == recordType {
			return rrSet, true
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
		content := change.Targets[i]

		if change.RecordType == txtRecord {
			content = formatTXTContent(content)
		}

		records[i] = stackitdnsclient.RecordPayload{
			Content: content,
		}
	}

	return stackitdnsclient.CreateRecordSetPayload{
		Name:    change.DNSName,
		Records: records,
		Ttl:     safeTTLToInt32(change.RecordTTL),
		Type:    change.RecordType,
	}
}

// getStackitPartialUpdateRecordSetPayload returns a stackitdnsclient.PartialUpdateRecordSetPayload from a change for the api client.
func getStackitPartialUpdateRecordSetPayload(change *endpoint.Endpoint) stackitdnsclient.PartialUpdateRecordSetPayload {
	records := make([]stackitdnsclient.RecordPayload, len(change.Targets))
	for i := range change.Targets {
		content := change.Targets[i]

		if change.RecordType == txtRecord {
			content = formatTXTContent(content)
		}

		records[i] = stackitdnsclient.RecordPayload{
			Content: content,
		}
	}

	return stackitdnsclient.PartialUpdateRecordSetPayload{
		Name:    &change.DNSName,
		Records: records,
		Ttl:     safeTTLToInt32(change.RecordTTL),
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

// safeTTLToInt32 safely converts an endpoint.TTL (int64) to *int32, clamping to valid bounds.
func safeTTLToInt32(ttl endpoint.TTL) *int32 {
	var v int32

	switch {
	case int64(ttl) > math.MaxInt32:
		v = math.MaxInt32
	case int64(ttl) < 0:
		v = 0
	default:
		v = int32(ttl) // #nosec G115 -- bounds checked above
	}

	return &v
}

// formatTXTContent splits long TXT records into 255-character chunks separated by spaces.
func formatTXTContent(content string) string {
	cleanContent := strings.Trim(content, `"`)

	if len(cleanContent) <= 255 {
		return content
	}

	var chunks []string
	for i := 0; i < len(cleanContent); i += 255 {
		end := i + 255
		if end > len(cleanContent) {
			end = len(cleanContent)
		}
		chunks = append(chunks, `"`+cleanContent[i:end]+`"`)
	}

	return strings.Join(chunks, " ")
}

// unformatTXTContent reverses the DNS chunking and quoting process.
func unformatTXTContent(content string) string {
	if strings.Contains(content, `" "`) {
		return strings.ReplaceAll(content, `" "`, ``)
	}

	return content
}
