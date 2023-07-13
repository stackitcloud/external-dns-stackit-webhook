package stackitprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	stackitdnsclient "github.com/stackitcloud/stackit-dns-api-client-go"
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
	validRespJson := getValidResponseZoneALlBytes(t)
	validRRSetResponse := getValidResponseRRSetAllBytes(t)
	invalidRespJson := []byte(`{"invalid: "json"`)

	// Test cases
	tests := getApplyChangesBasicTestCases(validRespJson, validRRSetResponse, invalidRespJson)

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

func getApplyChangesBasicTestCases( //nolint:funlen // Test cases are long
	validRespJson []byte,
	validRRSetResponse []byte,
	invalidRespJson []byte,
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
			validRespJson,
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
			invalidRespJson,
			http.StatusOK,
			validRRSetResponse,
			http.StatusAccepted,
			true,
			"",
		},
		{
			"Zone response, Rrset response 403",
			validRespJson,
			http.StatusOK,
			nil,
			http.StatusForbidden,
			true,
			http.MethodPost,
		},
		{
			"Zone response, Rrset response 500",
			validRespJson,
			http.StatusOK,
			nil,
			http.StatusInternalServerError,
			true,
			http.MethodPost,
		},
		// swagger client does not return an error when the response is invalid json
		{
			"Zone response, Rrset response Invalid JSON",
			validRespJson,
			http.StatusOK,
			invalidRespJson,
			http.StatusAccepted,
			false,
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

func getValidResponseZoneALlBytes(t *testing.T) []byte {
	t.Helper()

	zones := getValidZoneResponseAll()
	validRespJson, err := json.Marshal(zones)
	assert.NoError(t, err)

	return validRespJson
}

func getValidZoneResponseAll() stackitdnsclient.ZoneResponseZoneAll {
	return stackitdnsclient.ZoneResponseZoneAll{
		ItemsPerPage: 10,
		Message:      "success",
		TotalItems:   2,
		TotalPages:   1,
		Zones: []stackitdnsclient.DomainZone{
			{Id: "1234", DnsName: "test.com"},
			{Id: "5678", DnsName: "test2.com"},
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

func getValidResponseRRSetAll() stackitdnsclient.RrsetResponseRrSetAll {
	return stackitdnsclient.RrsetResponseRrSetAll{
		ItemsPerPage: 20,
		Message:      "success",
		RrSets: []stackitdnsclient.DomainRrSet{
			{
				Name:  "test.com",
				Type_: "A",
				Ttl:   300,
				Records: []stackitdnsclient.DomainRecord{
					{Content: "1.2.3.4"},
				},
			},
		},
		TotalItems: 2,
		TotalPages: 1,
	}
}
