package api

const (
	mediaTypeFormat      = "application/external.dns.webhook+json;version=1"
	contentTypeHeader    = "Content-Type"
	contentTypePlaintext = "text/plain"
	varyHeader           = "Vary"
	logFieldError        = "err"
)

type Message struct {
	Message string `json:"message"`
}
