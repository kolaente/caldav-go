package caldav

import (
	"net/http"

	"github.com/samedi/caldav-go/data"
	"github.com/samedi/caldav-go/handlers"
)

// RequestHandler handles the given CALDAV request and writes the reponse righ away. This function is to be
// used by passing it directly as the handle func to the `http` lib. Example: http.HandleFunc("/", caldav.RequestHandler).
func RequestHandler(writer http.ResponseWriter, request *http.Request) {
	response := HandleRequest(request)
	response.Write(writer)
}

// HandleRequest handles the given CALDAV request and returns the response. Useful when the caller
// wants to do something else with the response before writing it to the response stream.
func HandleRequest(request *http.Request) *handlers.Response {
	handler := handlers.NewHandler(request)
	return handler.Handle()
}

// HandleRequestWithStorage handles the request the same way as `HandleRequest` does, but before,
// it sets the given storage that will be used throughout the request handling flow.
//
// Deprecated: this mutates the global storage, so concurrent requests using different storages
// race with each other and can be answered with the wrong storage's data. Use
// `HandleRequestWithConfig` instead, which keeps all the request state local to the request.
func HandleRequestWithStorage(request *http.Request, stg data.Storage) *handlers.Response {
	SetupStorage(stg)
	return HandleRequest(request)
}

// Config carries the per-request state (storage, user and supported components) used to handle
// a single CalDAV request. Fields left empty fall back to the corresponding global setup.
type Config = handlers.Config

// HandleRequestWithConfig handles the given CALDAV request using only the state given in `config`,
// without reading nor writing any of the global setup (except as a fallback for the config fields
// left empty). This is the entry point to use when the same server handles concurrent requests for
// different users and/or storages.
func HandleRequestWithConfig(request *http.Request, config Config) *handlers.Response {
	return handlers.NewHandlerWithConfig(request, config).Handle()
}
