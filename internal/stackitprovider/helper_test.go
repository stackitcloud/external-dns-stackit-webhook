package stackitprovider

import (
	"reflect"
	"testing"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
)

func TestAppendDotIfNotExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		want string
	}{
		{"No dot at end", "test", "test."},
		{"Dot at end", "test.", "test."},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := appendDotIfNotExists(tt.s); got != tt.want {
				t.Errorf("appendDotIfNotExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModifyChange(t *testing.T) {
	t.Parallel()

	endpointWithTTL := &endpoint.Endpoint{
		DNSName:   "test",
		RecordTTL: endpoint.TTL(400),
	}
	modifyChange(endpointWithTTL)
	if endpointWithTTL.DNSName != "test." {
		t.Errorf("modifyChange() did not append dot to DNSName = %v, want test.", endpointWithTTL.DNSName)
	}
	if endpointWithTTL.RecordTTL != 400 {
		t.Errorf("modifyChange() changed existing RecordTTL = %v, want 400", endpointWithTTL.RecordTTL)
	}

	endpointWithoutTTL := &endpoint.Endpoint{
		DNSName: "test",
	}
	modifyChange(endpointWithoutTTL)
	if endpointWithoutTTL.DNSName != "test." {
		t.Errorf("modifyChange() did not append dot to DNSName = %v, want test.", endpointWithoutTTL.DNSName)
	}
	if endpointWithoutTTL.RecordTTL != 300 {
		t.Errorf("modifyChange() did not set default RecordTTL = %v, want 300", endpointWithoutTTL.RecordTTL)
	}
}

func TestGetStackitRRSetRecordPost(t *testing.T) {
	t.Parallel()

	change := &endpoint.Endpoint{
		DNSName:    "test.",
		RecordTTL:  endpoint.TTL(300),
		RecordType: "A",
		Targets: endpoint.Targets{
			"192.0.2.1",
			"192.0.2.2",
		},
	}
	expected := stackitdnsclient.CreateRecordSetPayload{
		Name: pointerTo("test."),
		Ttl:  pointerTo(int64(300)),
		Type: pointerTo("A"),
		Records: &[]stackitdnsclient.RecordPayload{
			{
				Content: pointerTo("192.0.2.1"),
			},
			{
				Content: pointerTo("192.0.2.2"),
			},
		},
	}
	got := getStackitRecordSetPayload(change)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("getStackitRRSetRecordPost() = %v, want %v", got, expected)
	}
}

func TestFindBestMatchingZone(t *testing.T) {
	t.Parallel()

	zones := []stackitdnsclient.Zone{
		{DnsName: pointerTo("foo.com")},
		{DnsName: pointerTo("bar.com")},
		{DnsName: pointerTo("baz.com")},
	}

	tests := []struct {
		name      string
		rrSetName string
		want      *stackitdnsclient.Zone
		wantFound bool
	}{
		{"Matching Zone", "www.foo.com", &zones[0], true},
		{"No Matching Zone", "www.test.com", nil, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, found := findBestMatchingZone(tt.rrSetName, zones)
			if !reflect.DeepEqual(got, tt.want) || found != tt.wantFound {
				t.Errorf("findBestMatchingZone() = %v, %v, want %v, %v", got, found, tt.want, tt.wantFound)
			}
		})
	}
}

func TestFindRRSet(t *testing.T) {
	t.Parallel()

	rrSets := []stackitdnsclient.RecordSet{
		{Name: pointerTo("www.foo.com"), Type: pointerTo("A")},
		{Name: pointerTo("www.bar.com"), Type: pointerTo("A")},
		{Name: pointerTo("www.baz.com"), Type: pointerTo("A")},
	}

	tests := []struct {
		name       string
		rrSetName  string
		recordType string
		want       *stackitdnsclient.RecordSet
		wantFound  bool
	}{
		{"Matching RRSet", "www.foo.com", "A", &rrSets[0], true},
		{"No Matching RRSet", "www.test.com", "A", nil, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, found := findRRSet(tt.rrSetName, tt.recordType, rrSets)
			if !reflect.DeepEqual(got, tt.want) || found != tt.wantFound {
				t.Errorf("findRRSet() = %v, %v, want %v, %v", got, found, tt.want, tt.wantFound)
			}
		})
	}
}

func TestGetLogFields(t *testing.T) {
	t.Parallel()

	change := &endpoint.Endpoint{
		DNSName:    "test.",
		RecordTTL:  endpoint.TTL(300),
		RecordType: "A",
		Targets: endpoint.Targets{
			"192.0.2.1",
			"192.0.2.2",
		},
	}

	expected := []zap.Field{
		zap.String("record", "test."),
		zap.String("content", "192.0.2.1,192.0.2.2"),
		zap.String("type", "A"),
		zap.String("action", "create"),
		zap.String("id", "123"),
	}

	got := getLogFields(change, "create", "123")

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("getLogFields() = %v, want %v", got, expected)
	}
}

func TestGetStackitRRSetRecordPatch(t *testing.T) {
	t.Parallel()

	change := &endpoint.Endpoint{
		DNSName:    "test.",
		RecordTTL:  endpoint.TTL(300),
		RecordType: "A",
		Targets: endpoint.Targets{
			"192.0.2.1",
			"192.0.2.2",
		},
	}

	expected := stackitdnsclient.PartialUpdateRecordSetPayload{
		Name: pointerTo("test."),
		Ttl:  pointerTo(int64(300)),
		Records: &[]stackitdnsclient.RecordPayload{
			{
				Content: pointerTo("192.0.2.1"),
			},
			{
				Content: pointerTo("192.0.2.2"),
			},
		},
	}

	got := getStackitPartialUpdateRecordSetPayload(change)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("getStackitRRSetRecordPatch() = %v, want %v", got, expected)
	}
}
