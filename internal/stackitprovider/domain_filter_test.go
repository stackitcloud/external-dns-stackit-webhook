package stackitprovider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
)

func TestGetDomainFilter(t *testing.T) {
	t.Parallel()

	server := getServerRecords(t)
	defer server.Close()

	stackitDnsProvider, err := getDefaultTestProvider(server)
	assert.NoError(t, err)

	domainFilter := stackitDnsProvider.GetDomainFilter()
	assert.Equal(t, domainFilter, endpoint.DomainFilter{})
}
