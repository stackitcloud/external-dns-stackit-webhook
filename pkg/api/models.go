package api

const (
	mediaTypeFormat        = "application/external.dns.webhook+json;version=1"
	contentTypeHeader      = "Content-Type"
	contentTypePlaintext   = "text/plain"
	acceptHeader           = "Accept"
	varyHeader             = "Vary"
	supportedMediaVersions = "1"
	healthPath             = "/health"
	logFieldRequestPath    = "requestPath"
	logFieldRequestMethod  = "requestMethod"
	logFieldError          = "err"
)

var mediaTypeVersion1 = mediaTypeVersion("1")

type mediaType string

func mediaTypeVersion(v string) mediaType {
	return mediaType(mediaTypeFormat + "version=" + v)
}

func (m mediaType) Is(headerValue string) bool {
	return string(m) == headerValue
}

type Message struct {
	Message string `json:"message"`
}
