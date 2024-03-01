package stackitprovider

import "sigs.k8s.io/external-dns/endpoint"

// Config is used to configure the creation of the StackitDNSProvider.
type Config struct {
	ProjectId    string
	DomainFilter endpoint.DomainFilter
	DryRun       bool
	Workers      int
}
