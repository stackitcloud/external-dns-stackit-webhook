package stackitprovider

import (
	"reflect"
	"strings"
	"testing"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns/v1api"
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
		Name: "test.",
		Ttl:  new(int32(300)),
		Type: "A",
		Records: []stackitdnsclient.RecordPayload{
			{
				Content: "192.0.2.1",
			},
			{
				Content: "192.0.2.2",
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
		{DnsName: "foo.com"},
		{DnsName: "bar.com"},
		{DnsName: "baz.com"},
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
		{Name: "www.foo.com", Type: "A"},
		{Name: "www.bar.com", Type: "A"},
		{Name: "www.baz.com", Type: "A"},
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
		Name: new("test."),
		Ttl:  new(int32(300)),
		Records: []stackitdnsclient.RecordPayload{
			{
				Content: "192.0.2.1",
			},
			{
				Content: "192.0.2.2",
			},
		},
	}

	got := getStackitPartialUpdateRecordSetPayload(change)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("getStackitRRSetRecordPatch() = %v, want %v", got, expected)
	}
}

func TestFormatTXTContent(t *testing.T) {
	t.Parallel()

	// Generate strings of exact lengths for testing
	string255 := strings.Repeat("a", 255)
	string256 := strings.Repeat("a", 256)
	string511 := strings.Repeat("a", 511)

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "Short string without quotes",
			content: "hello world",
			want:    `"hello world"`,
		},
		{
			name:    "Short string with existing quotes",
			content: `"hello world"`,
			want:    `"hello world"`,
		},
		{
			name:    "Exactly 255 characters",
			content: string255,
			want:    `"` + string255 + `"`,
		},
		{
			name:    "256 characters (requires 2 chunks)",
			content: string256,
			want:    `"` + string255 + `" "a"`,
		},
		{
			name:    "511 characters (requires 3 chunks)",
			content: string511,
			want:    `"` + string255 + `" "` + string255 + `" "a"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatTXTContent(tt.content); got != tt.want {
				t.Errorf("formatTXTContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnformatTXTContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "Unquoted short string",
			content: "hello world",
			want:    "hello world",
		},
		{
			name:    "Single chunk quoted string",
			content: `"hello world"`,
			want:    "hello world",
		},
		{
			name:    "Two chunk string",
			content: `"hello" "world"`,
			want:    "helloworld",
		},
		{
			name:    "Three chunk string",
			content: `"chunk1" "chunk2" "chunk3"`,
			want:    "chunk1chunk2chunk3",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := unformatTXTContent(tt.content); got != tt.want {
				t.Errorf("unformatTXTContent() = %v, want %v", got, tt.want)
			}
		})
	}
}
