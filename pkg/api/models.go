package api

const (
	mediaTypeFormat        = "application/external.dns.webhook+json;"
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

// PropertyValuesEqualsRequest holds params for property values equals request.
type PropertyValuesEqualsRequest struct {
	Name     string `json:"name"`
	Previous string `json:"previous"`
	Current  string `json:"current"`
}

// PropertiesValuesEqualsResponse holds params for property values equals response.
type PropertiesValuesEqualsResponse struct {
	Equals bool `json:"equals"`
}
