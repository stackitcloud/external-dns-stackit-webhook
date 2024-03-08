package stackitprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-json"
	stackitconfig "github.com/stackitcloud/stackit-sdk-go/core/config"
	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
)

func TestRecords(t *testing.T) {
	t.Parallel()

	server := getServerRecords(t)
	defer server.Close()

	stackitDnsProvider, err := getDefaultTestProvider(server)
	assert.NoError(t, err)

	endpoints, err := stackitDnsProvider.Records(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(endpoints))
	assert.Equal(t, "test.com", endpoints[0].DNSName)
	assert.Equal(t, "A", endpoints[0].RecordType)
	assert.Equal(t, "1.2.3.4", endpoints[0].Targets[0])
	assert.Equal(t, int64(300), int64(endpoints[0].RecordTTL))

	assert.Equal(t, "test2.com", endpoints[1].DNSName)
	assert.Equal(t, "A", endpoints[1].RecordType)
	assert.Equal(t, "5.6.7.8", endpoints[1].Targets[0])
	assert.Equal(t, int64(300), int64(endpoints[1].RecordTTL))
}

// TestWrongJsonResponseRecords tests the scenario where the server returns a wrong JSON response.
func TestWrongJsonResponseRecords(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	mux.HandleFunc("/v1/projects/1234/zones",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"invalid:"json"`)) // This is not a valid JSON.
		},
	)
	defer server.Close()

	stackitDnsProvider, err := getDefaultTestProvider(server)
	assert.NoError(t, err)

	endpoints, err := stackitDnsProvider.Records(context.Background())
	assert.Error(t, err)
	assert.Equal(t, 0, len(endpoints))
}

// TestPagedResponseRecords tests the scenario where the server returns a paged response.
func TestPagedResponseRecords(t *testing.T) {
	t.Parallel()

	server := getPagedRecordsServer(t)
	defer server.Close()

	stackitDnsProvider, err := getDefaultTestProvider(server)
	assert.NoError(t, err)

	endpoints, err := stackitDnsProvider.Records(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(endpoints))
	assert.Equal(t, "test.com", endpoints[0].DNSName)
	assert.Equal(t, "A", endpoints[0].RecordType)
	assert.Equal(t, "1.2.3.4", endpoints[0].Targets[0])
	assert.Equal(t, int64(300), int64(endpoints[0].RecordTTL))

	assert.Equal(t, "test2.com", endpoints[1].DNSName)
	assert.Equal(t, "A", endpoints[1].RecordType)
	assert.Equal(t, "5.6.7.8", endpoints[1].Targets[0])
	assert.Equal(t, int64(300), int64(endpoints[1].RecordTTL))
}

func TestEmptyZonesRouteRecords(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	stackitDnsProvider, err := getDefaultTestProvider(server)
	assert.NoError(t, err)

	_, err = stackitDnsProvider.Records(context.Background())
	assert.Error(t, err)
}

func TestEmptyRRSetRouteRecords(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/v1/projects/1234/zones",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			zones := stackitdnsclient.ListZonesResponse{
				ItemsPerPage: pointerTo(int64(1)),
				Message:      pointerTo("success"),
				TotalItems:   pointerTo(int64(2)),
				TotalPages:   pointerTo(int64(2)),
				Zones:        &[]stackitdnsclient.Zone{{Id: pointerTo("1234")}},
			}
			successResponseBytes, err := json.Marshal(zones)
			assert.NoError(t, err)

			w.WriteHeader(http.StatusOK)
			w.Write(successResponseBytes)
		},
	)
	defer server.Close()

	stackitDnsProvider, err := getDefaultTestProvider(server)
	assert.NoError(t, err)

	_, err = stackitDnsProvider.Records(context.Background())
	assert.Error(t, err)
}

func TestZoneEndpoint500Records(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/v1/projects/1234/zones",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			w.WriteHeader(http.StatusInternalServerError)
		},
	)
	defer server.Close()

	stackitDnsProvider, err := getDefaultTestProvider(server)
	assert.NoError(t, err)

	_, err = stackitDnsProvider.Records(context.Background())
	assert.Error(t, err)
}

func TestZoneEndpoint403Records(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/v1/projects/1234/zones",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			w.WriteHeader(http.StatusForbidden)
		},
	)
	defer server.Close()

	stackitDnsProvider, err := NewStackitDNSProvider(
		zap.NewNop(),
		Config{
			ProjectId:    "1234",
			DomainFilter: endpoint.DomainFilter{},
			DryRun:       false,
			Workers:      10,
		},
		stackitconfig.WithHTTPClient(server.Client()),
		stackitconfig.WithEndpoint(server.URL),
		// we need a non-empty token for the bootstrapping not to fail
		stackitconfig.WithToken("token"),
	)
	assert.NoError(t, err)

	_, err = stackitDnsProvider.Records(context.Background())
	assert.Error(t, err)
}

func getDefaultTestProvider(server *httptest.Server) (*StackitDNSProvider, error) {
	stackitDnsProvider, err := NewStackitDNSProvider(
		zap.NewNop(),
		Config{
			ProjectId:    "1234",
			DomainFilter: endpoint.DomainFilter{},
			DryRun:       false,
			Workers:      1,
		},
		stackitconfig.WithHTTPClient(server.Client()),
		stackitconfig.WithEndpoint(server.URL),
		// we need a non-empty token for the bootstrapping not to fail
		stackitconfig.WithToken("token"))

	return stackitDnsProvider, err
}

func getZonesHandlerRecordsPaged(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		zones := stackitdnsclient.ListZonesResponse{}
		if r.URL.Query().Get("page") == "1" {
			zones = stackitdnsclient.ListZonesResponse{
				ItemsPerPage: pointerTo(int64(1)),
				Message:      pointerTo("success"),
				TotalItems:   pointerTo(int64(2)),
				TotalPages:   pointerTo(int64(2)),
				Zones:        &[]stackitdnsclient.Zone{{Id: pointerTo("1234")}},
			}
		}
		if r.URL.Query().Get("page") == "2" {
			zones = stackitdnsclient.ListZonesResponse{
				ItemsPerPage: pointerTo(int64(1)),
				Message:      pointerTo("success"),
				TotalItems:   pointerTo(int64(2)),
				TotalPages:   pointerTo(int64(2)),
				Zones:        &[]stackitdnsclient.Zone{{Id: pointerTo("5678")}},
			}
		}
		successResponseBytes, err := json.Marshal(zones)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write(successResponseBytes)
	}
}

func getRrsetsHandlerReecodsPaged(t *testing.T, domain string) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		rrSets := stackitdnsclient.ListRecordSetsResponse{}
		if domain == "1234" {
			rrSets = stackitdnsclient.ListRecordSetsResponse{
				ItemsPerPage: pointerTo(int64(1)),
				Message:      pointerTo("success"),
				RrSets: &[]stackitdnsclient.RecordSet{
					{
						Name: pointerTo("test.com."),
						Type: pointerTo("A"),
						Ttl:  pointerTo(int64(300)),
						Records: &[]stackitdnsclient.Record{
							{Content: pointerTo("1.2.3.4")},
						},
					},
				},
				TotalItems: pointerTo(int64(1)),
				TotalPages: pointerTo(int64(1)),
			}
		}
		if domain == "5678" {
			rrSets = stackitdnsclient.ListRecordSetsResponse{
				ItemsPerPage: pointerTo(int64(1)),
				Message:      pointerTo("success"),
				RrSets: &[]stackitdnsclient.RecordSet{
					{
						Name: pointerTo("test2.com."),
						Type: pointerTo("A"),
						Ttl:  pointerTo(int64(300)),
						Records: &[]stackitdnsclient.Record{
							{Content: pointerTo("5.6.7.8")},
						},
					},
				},
				TotalItems: pointerTo(int64(1)),
				TotalPages: pointerTo(int64(1)),
			}
		}

		successResponseBytes, err := json.Marshal(rrSets)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		w.Write(successResponseBytes)
	}
}

func getPagedRecordsServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	mux.HandleFunc("/v1/projects/1234/zones", getZonesHandlerRecordsPaged(t))
	mux.HandleFunc("/v1/projects/1234/zones/1234/rrsets", getRrsetsHandlerReecodsPaged(t, "1234"))
	mux.HandleFunc("/v1/projects/1234/zones/5678/rrsets", getRrsetsHandlerReecodsPaged(t, "5678"))

	return server
}

func getZonesResponseRecordsNonPaged(t *testing.T, w http.ResponseWriter) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")

	zones := stackitdnsclient.ListZonesResponse{
		ItemsPerPage: pointerTo(int64(10)),
		Message:      pointerTo("success"),
		TotalItems:   pointerTo(int64(2)),
		TotalPages:   pointerTo(int64(1)),
		Zones: &[]stackitdnsclient.Zone{
			{Id: pointerTo("1234"), DnsName: pointerTo("test.com")},
			{Id: pointerTo("5678"), DnsName: pointerTo("test2.com")},
		},
	}
	successResponseBytes, err := json.Marshal(zones)
	assert.NoError(t, err)

	w.WriteHeader(http.StatusOK)
	w.Write(successResponseBytes)
}

func getRrsetsResponseRecordsNonPaged(t *testing.T, w http.ResponseWriter, domain string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")

	var rrSets stackitdnsclient.ListRecordSetsResponse

	if domain == "1234" {
		rrSets = stackitdnsclient.ListRecordSetsResponse{
			ItemsPerPage: pointerTo(int64(20)),
			Message:      pointerTo("success"),
			RrSets: &[]stackitdnsclient.RecordSet{
				{
					Name: pointerTo("test.com."),
					Type: pointerTo("A"),
					Ttl:  pointerTo(int64(300)),
					Records: &[]stackitdnsclient.Record{
						{Content: pointerTo("1.2.3.4")},
					},
					Id: pointerTo("1234"),
				},
			},
			TotalItems: pointerTo(int64(2)),
			TotalPages: pointerTo(int64(1)),
		}
	} else if domain == "5678" {
		rrSets = stackitdnsclient.ListRecordSetsResponse{
			ItemsPerPage: pointerTo(int64(20)),
			Message:      pointerTo("success"),
			RrSets: &[]stackitdnsclient.RecordSet{
				{
					Name: pointerTo("test2.com."),
					Type: pointerTo("A"),
					Ttl:  pointerTo(int64(300)),
					Records: &[]stackitdnsclient.Record{
						{Content: pointerTo("5.6.7.8")},
					},
					Id: pointerTo("5678"),
				},
			},
			TotalItems: pointerTo(int64(2)),
			TotalPages: pointerTo(int64(1)),
		}
	}

	successResponseBytes, err := json.Marshal(rrSets)
	assert.NoError(t, err)

	w.WriteHeader(http.StatusOK)
	w.Write(successResponseBytes)
}

func getServerRecords(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	mux.HandleFunc("/v1/projects/1234/zones", func(w http.ResponseWriter, r *http.Request) {
		getZonesResponseRecordsNonPaged(t, w)
	})
	mux.HandleFunc("/v1/projects/1234/zones/1234/rrsets", func(w http.ResponseWriter, r *http.Request) {
		getRrsetsResponseRecordsNonPaged(t, w, "1234")
	})
	mux.HandleFunc("/v1/projects/1234/zones/5678/rrsets", func(w http.ResponseWriter, r *http.Request) {
		getRrsetsResponseRecordsNonPaged(t, w, "5678")
	})

	return server
}
