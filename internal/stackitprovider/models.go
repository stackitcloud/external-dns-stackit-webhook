package stackitprovider

import "sigs.k8s.io/external-dns/endpoint"

const (
	CREATE = "CREATE"
	UPDATE = "UPDATE"
	DELETE = "DELETE"
)

// ErrorMessage is the error message returned by the API.
type ErrorMessage struct {
	Message string `json:"message"`
}

// changeTask is a task that is passed to the worker.
type changeTask struct {
	change *endpoint.Endpoint
	action string
}

// endpointError is a list of endpoints and an error to pass to workers.
type endpointError struct {
	endpoints []*endpoint.Endpoint
	err       error
}
