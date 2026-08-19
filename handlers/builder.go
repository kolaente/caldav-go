package handlers

import (
	"net/http"

	"github.com/samedi/caldav-go/data"
	"github.com/samedi/caldav-go/global"
)

// HandlerInterface represents a CalDAV request handler. It has only one function `Handle`,
// which is used to handle the CalDAV request and returns the response.
type HandlerInterface interface {
	Handle() *Response
}

// Config carries all the state a single request needs to be handled: the storage the resources
// are read from/written to, the user currently interacting with the calendar and the components
// supported by that storage. Passing it per request is what allows a server to handle concurrent
// requests for different users without them stepping on each other.
// Any field left empty falls back to its counterpart in the `global` package.
type Config struct {
	Storage             data.Storage
	User                *data.CalUser
	SupportedComponents []string
}

// Fills in the fields that were left empty with the global (server-wide) configuration.
func (c Config) withGlobalFallbacks() Config {
	if c.Storage == nil {
		c.Storage = global.Storage
	}

	if c.User == nil {
		c.User = global.User
	}

	if len(c.SupportedComponents) == 0 {
		c.SupportedComponents = global.SupportedComponents
	}

	return c
}

// Common data shared across the specific handlers. Defined here to
// easily make available, in a single place, all the basic data possibly needed by the handlers.
type handlerData struct {
	request             *http.Request
	requestBody         string
	requestPath         string
	headers             headers
	response            *Response
	storage             data.Storage
	user                *data.CalUser
	supportedComponents []string
}

// NewHandler returns a new CalDAV request handler object based on the provided request.
// With the returned request handler, you can call `Handle()` to handle the request.
// The handler uses the global configuration (see the `global` package).
func NewHandler(request *http.Request) HandlerInterface {
	return NewHandlerWithConfig(request, Config{})
}

// NewHandlerWithConfig returns a new CalDAV request handler object based on the provided request,
// handled with the provided per-request `config`. Empty config fields fall back to the global configuration.
func NewHandlerWithConfig(request *http.Request, config Config) HandlerInterface {
	config = config.withGlobalFallbacks()

	hData := handlerData{
		request:             request,
		requestBody:         readRequestBody(request),
		requestPath:         request.URL.Path,
		headers:             headers{request.Header},
		response:            NewResponse(),
		storage:             config.Storage,
		user:                config.User,
		supportedComponents: config.SupportedComponents,
	}

	switch request.Method {
	case "GET":
		return getHandler{handlerData: hData, onlyHeaders: false}
	case "HEAD":
		return getHandler{handlerData: hData, onlyHeaders: true}
	case "PUT":
		return putHandler{hData}
	case "DELETE":
		return deleteHandler{hData}
	case "PROPFIND":
		return propfindHandler{hData}
	case "OPTIONS":
		return optionsHandler{hData}
	case "REPORT":
		return reportHandler{hData}
	default:
		return notImplementedHandler{hData}
	}
}
