package stackitprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

type ChangeType int

const (
	Create ChangeType = iota
	Update
	Delete
)

func TestApplyChanges(t *testing.T) {
	t.Parallel()

	testingData := []struct {
		changeType ChangeType
	}{
		{changeType: Create},
		{changeType: Update},
		{changeType: Delete},
	}

	for _, data := range testingData {
		testApplyChanges(t, data.changeType)
	}
}

func testApplyChanges(t *testing.T, changeType ChangeType) {
	t.Helper()
	ctx := context.Background()
	validZoneResponse := getValidResponseZoneAllBytes(t)
	validRRSetResponse := getValidResponseRRSetAllBytes(t)
	invalidZoneResponse := []byte(`{"invalid: "json"`)

	// Test cases
	tests := getApplyChangesBasicTestCases(validZoneResponse, validRRSetResponse, invalidZoneResponse)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()
			server := httptest.NewServer(mux)

			// Set up common endpoint for all types of changes
			setUpCommonEndpoints(mux, tt.responseZone, tt.responseZoneCode)

			// Set up change type-specific endpoints
			setUpChangeTypeEndpoints(t, mux, tt.responseRrset, tt.responseRrsetCode, changeType)

			defer server.Close()

			stackitDnsProvider, err := getDefaultTestProvider(server)
			assert.NoError(t, err)

			// Set up the changes according to the change type
			changes := getChangeTypeChanges(changeType)

			err = stackitDnsProvider.ApplyChanges(ctx, changes)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNoMatchingZoneFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	validZoneResponse := getValidResponseZoneAllBytes(t)

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	// Set up common endpoint for all types of changes
	setUpCommonEndpoints(mux, validZoneResponse, http.StatusOK)

	stackitDnsProvider, err := getDefaultTestProvider(server)
	assert.NoError(t, err)

	changes := &plan.Changes{
		Create: []*endpoint.Endpoint{
			{DNSName: "notfound.com", Targets: endpoint.Targets{"test.notfound.com"}},
		},
		UpdateNew: []*endpoint.Endpoint{},
		Delete:    []*endpoint.Endpoint{},
	}

	err = stackitDnsProvider.ApplyChanges(ctx, changes)
	assert.Error(t, err)
}

func TestNoRRSetFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	validZoneResponse := getValidResponseZoneAllBytes(t)
	rrSets := getValidResponseRRSetAll()
	rrSet := *rrSets.RrSets
	*rrSet[0].Name = "notfound.test.com"
	validRRSetResponse, err := json.Marshal(rrSets)
	assert.NoError(t, err)

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	// Set up common endpoint for all types of changes
	setUpCommonEndpoints(mux, validZoneResponse, http.StatusOK)

	mux.HandleFunc(
		"/v1/projects/1234/zones/1234/rrsets",
		responseHandler(validRRSetResponse, http.StatusOK),
	)

	stackitDnsProvider, err := getDefaultTestProvider(server)
	assert.NoError(t, err)

	changes := &plan.Changes{
		UpdateNew: []*endpoint.Endpoint{
			{DNSName: "test.com", Targets: endpoint.Targets{"notfound.test.com"}},
		},
	}

	err = stackitDnsProvider.ApplyChanges(ctx, changes)
	assert.Error(t, err)
}

func TestPartialUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	validZoneResponse := getValidResponseZoneAllBytes(t)

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	// Set up common endpoint for all types of changes
	setUpCommonEndpoints(mux, validZoneResponse, http.StatusOK)
	// Set up change type-specific endpoints
	// based on setUpChangeTypeEndpoints(t, mux, validRRSetResponse, http.StatusOK, Update)
	// but extended to check that the rrset is updated
	rrSetUpdated := false
	mux.HandleFunc(
		"/v1/projects/1234/zones/1234/rrsets/1234",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Println(r.Method)
			if r.Method == http.MethodPatch {
				rrSetUpdated = true
			}
		},
	)
	mux.HandleFunc(
		"/v1/projects/1234/zones/1234/rrsets",
		func(w http.ResponseWriter, r *http.Request) {
			getRrsetsResponseRecordsNonPaged(t, w, "1234")
		},
	)
	mux.HandleFunc(
		"/v1/projects/1234/zones/5678/rrsets",
		func(w http.ResponseWriter, r *http.Request) {
			getRrsetsResponseRecordsNonPaged(t, w, "5678")
		},
	)

	stackitDnsProvider, err := getDefaultTestProvider(server)
	assert.NoError(t, err)

	// Create update change
	changes := getChangeTypeChanges(Update)
	// Add task to create invalid endpoint
	changes.Create = []*endpoint.Endpoint{
		{DNSName: "notfound.com", Targets: endpoint.Targets{"test.notfound.com"}},
	}

	err = stackitDnsProvider.ApplyChanges(ctx, changes)
	assert.Error(t, err)
	assert.True(t, rrSetUpdated, "rrset was not updated")
}

// setUpCommonEndpoints for all change types.
func setUpCommonEndpoints(mux *http.ServeMux, responseZone []byte, responseZoneCode int) {
	mux.HandleFunc("/v1/projects/1234/zones", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(responseZoneCode)
		w.Write(responseZone)
	})
}

// setUpChangeTypeEndpoints for type-specific endpoints.
func setUpChangeTypeEndpoints(
	t *testing.T,
	mux *http.ServeMux,
	responseRrset []byte,
	responseRrsetCode int,
	changeType ChangeType,
) {
	t.Helper()

	switch changeType {
	case Create:
		mux.HandleFunc(
			"/v1/projects/1234/zones/1234/rrsets",
			responseHandler(responseRrset, responseRrsetCode),
		)
	case Update, Delete:
		mux.HandleFunc(
			"/v1/projects/1234/zones/1234/rrsets/1234",
			responseHandler(nil, responseRrsetCode),
		)
		mux.HandleFunc(
			"/v1/projects/1234/zones/1234/rrsets",
			func(w http.ResponseWriter, r *http.Request) {
				getRrsetsResponseRecordsNonPaged(t, w, "1234")
			},
		)
		mux.HandleFunc(
			"/v1/projects/1234/zones/5678/rrsets",
			func(w http.ResponseWriter, r *http.Request) {
				getRrsetsResponseRecordsNonPaged(t, w, "5678")
			},
		)
	}
}

// getChangeTypeChanges according to the change type.
func getChangeTypeChanges(changeType ChangeType) *plan.Changes {
	switch changeType {
	case Create:
		return &plan.Changes{
			Create: []*endpoint.Endpoint{
				{DNSName: "test.com", Targets: endpoint.Targets{"test.test.com"}},
			},
			UpdateNew: []*endpoint.Endpoint{},
			Delete:    []*endpoint.Endpoint{},
		}
	case Update:
		return &plan.Changes{
			UpdateNew: []*endpoint.Endpoint{
				{DNSName: "test.com", Targets: endpoint.Targets{"test.com"}, RecordType: "A"},
			},
		}
	case Delete:
		return &plan.Changes{
			Delete: []*endpoint.Endpoint{
				{DNSName: "test.com", Targets: endpoint.Targets{"test.com"}, RecordType: "A"},
			},
		}
	default:
		return nil
	}
}

func getApplyChangesBasicTestCases(
	validZoneResponse []byte,
	validRRSetResponse []byte,
	invalidZoneResponse []byte,
) []struct {
	name                string
	responseZone        []byte
	responseZoneCode    int
	responseRrset       []byte
	responseRrsetCode   int
	expectErr           bool
	expectedRrsetMethod string
} {
	tests := []struct {
		name                string
		responseZone        []byte
		responseZoneCode    int
		responseRrset       []byte
		responseRrsetCode   int
		expectErr           bool
		expectedRrsetMethod string
	}{
		{
			"Valid response",
			validZoneResponse,
			http.StatusOK,
			validRRSetResponse,
			http.StatusAccepted,
			false,
			http.MethodPost,
		},
		{
			"Zone response 403",
			nil,
			http.StatusForbidden,
			validRRSetResponse,
			http.StatusAccepted,
			true,
			"",
		},
		{
			"Zone response 500",
			nil,
			http.StatusInternalServerError,
			validRRSetResponse,
			http.StatusAccepted,
			true,
			"",
		},
		{
			"Zone response Invalid JSON",
			invalidZoneResponse,
			http.StatusOK,
			validRRSetResponse,
			http.StatusAccepted,
			true,
			"",
		},
		{
			"Zone response, Rrset response 403",
			validZoneResponse,
			http.StatusOK,
			nil,
			http.StatusForbidden,
			true,
			http.MethodPost,
		},
		{
			"Zone response, Rrset response 500",
			validZoneResponse,
			http.StatusOK,
			nil,
			http.StatusInternalServerError,
			true,
			http.MethodPost,
		},
	}

	return tests
}

func responseHandler(responseBody []byte, statusCode int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if responseBody != nil {
			w.Write(responseBody)
		}
	}
}

func getValidResponseZoneAllBytes(t *testing.T) []byte {
	t.Helper()

	zones := getValidZoneResponseAll()
	validZoneResponse, err := json.Marshal(zones)
	assert.NoError(t, err)

	return validZoneResponse
}

func getValidZoneResponseAll() stackitdnsclient.ListZonesResponse {
	return stackitdnsclient.ListZonesResponse{
		ItemsPerPage: pointerTo(int64(10)),
		Message:      pointerTo("success"),
		TotalItems:   pointerTo(int64(2)),
		TotalPages:   pointerTo(int64(1)),
		Zones: &[]stackitdnsclient.Zone{
			{Id: pointerTo("1234"), DnsName: pointerTo("test.com")},
			{Id: pointerTo("5678"), DnsName: pointerTo("test2.com")},
		},
	}
}

func getValidResponseRRSetAllBytes(t *testing.T) []byte {
	t.Helper()

	rrSets := getValidResponseRRSetAll()
	validRRSetResponse, err := json.Marshal(rrSets)
	assert.NoError(t, err)

	return validRRSetResponse
}

func getValidResponseRRSetAll() stackitdnsclient.ListRecordSetsResponse {
	return stackitdnsclient.ListRecordSetsResponse{
		ItemsPerPage: pointerTo(int64(20)),
		Message:      pointerTo("success"),
		RrSets: &[]stackitdnsclient.RecordSet{
			{
				Name: pointerTo("test.com"),
				Type: pointerTo(stackitdnsclient.RECORDSETTYPE_A),
				Ttl:  pointerTo(int64(300)),
				Records: &[]stackitdnsclient.Record{
					{Content: pointerTo("1.2.3.4")},
				},
			},
		},
		TotalItems: pointerTo(int64(2)),
		TotalPages: pointerTo(int64(1)),
	}
}
