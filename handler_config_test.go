package caldav

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/samedi/caldav-go/data"
	"github.com/samedi/caldav-go/global"
	"github.com/samedi/caldav-go/lib"
	"github.com/samedi/caldav-go/test"
)

const (
	configTestCollection = "/test-config/"
	configTestResource   = "/test-config/123-456-789.ics"
)

// The storage passed in the config must be the one used to handle the request,
// even when the global storage points somewhere else.
func TestHandleRequestWithConfigUsesTheGivenStorage(t *testing.T) {
	defer saveGlobals()()

	global.Storage = newStorageWithEvent("Global")
	requestStorage := newStorageWithEvent("Request")

	resp := HandleRequestWithConfig(newRequest("GET", configTestResource, ""), Config{
		Storage: requestStorage,
	})

	test.AssertInt(resp.Status, http.StatusOK, t)
	test.AssertStr(resp.Body, eventData("Request"), t)
}

// The user passed in the config must be the one rendered in the `current-user-principal` prop,
// even when the global user is a different one.
func TestHandleRequestWithConfigUsesTheGivenUser(t *testing.T) {
	defer saveGlobals()()

	global.Storage = newStorageWithEvent("Global")
	global.User = &data.CalUser{Name: "global-user"}

	requestXML := `
	<?xml version="1.0" encoding="utf-8" ?>
	<D:propfind xmlns:D="DAV:">
		<D:prop>
			<D:current-user-principal/>
		</D:prop>
	</D:propfind>
	`

	resp := HandleRequestWithConfig(newRequest("PROPFIND", configTestResource, requestXML), Config{
		Storage: newStorageWithEvent("Request"),
		User:    &data.CalUser{Name: "request-user"},
	})

	expectedRespBody := fmt.Sprintf(`
	<?xml version="1.0" encoding="UTF-8"?>
	<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
		<D:response>
			<D:href>%s</D:href>
			<D:propstat>
				<D:prop>
					<D:current-user-principal>
						<D:href>/request-user/</D:href>
					</D:current-user-principal>
				</D:prop>
				<D:status>HTTP/1.1 200 OK</D:status>
			</D:propstat>
		</D:response>
	</D:multistatus>
	`, configTestResource)

	test.AssertInt(resp.Status, 207, t)
	test.AssertMultistatusXML(resp.Body, expectedRespBody, t)
}

// The supported components passed in the config must be the ones rendered in the
// `supported-calendar-component-set` prop, even when the global ones are different.
func TestHandleRequestWithConfigUsesTheGivenSupportedComponents(t *testing.T) {
	defer saveGlobals()()

	global.Storage = newStorageWithEvent("Global")
	global.SupportedComponents = []string{lib.VCALENDAR, lib.VEVENT}

	requestXML := `
	<?xml version="1.0" encoding="utf-8" ?>
	<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
		<D:prop>
			<C:supported-calendar-component-set/>
		</D:prop>
	</D:propfind>
	`

	resp := HandleRequestWithConfig(newRequest("PROPFIND", configTestCollection, requestXML), Config{
		Storage:             newStorageWithEvent("Request"),
		SupportedComponents: []string{lib.VJOURNAL, lib.VTODO},
	})

	expectedRespBody := `
	<?xml version="1.0" encoding="UTF-8"?>
	<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
		<D:response>
			<D:href>/test-config</D:href>
			<D:propstat>
				<D:prop>
					<C:supported-calendar-component-set>
						<C:comp name="VJOURNAL"/>
						<C:comp name="VTODO"/>
					</C:supported-calendar-component-set>
				</D:prop>
				<D:status>HTTP/1.1 200 OK</D:status>
			</D:propstat>
		</D:response>
	</D:multistatus>
	`

	test.AssertInt(resp.Status, 207, t)
	test.AssertMultistatusXML(resp.Body, expectedRespBody, t)
}

// Requests handled at the same time with different configs must not see each other's data.
func TestHandleRequestWithConfigIsConcurrencySafe(t *testing.T) {
	defer saveGlobals()()

	global.Storage = newStorageWithEvent("Global")
	global.User = &data.CalUser{Name: "global-user"}

	requestXML := `
	<?xml version="1.0" encoding="utf-8" ?>
	<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
		<D:prop>
			<C:calendar-data/>
			<D:current-user-principal/>
		</D:prop>
	</D:propfind>
	`

	const (
		concurrency = 8
		repetitions = 25
	)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			// each goroutine handles requests for its own user, on its own storage, but
			// on the very same resource path, so that any leaking between them is visible.
			owner := fmt.Sprintf("user-%d", i)
			config := Config{
				Storage: newStorageWithEvent(owner),
				User:    &data.CalUser{Name: owner},
			}

			for j := 0; j < repetitions; j++ {
				resp := HandleRequestWithConfig(newRequest("PROPFIND", configTestResource, requestXML), config)

				if !strings.Contains(resp.Body, "SUMMARY:"+owner) {
					t.Errorf("response of %s does not contain its own data. Got: %s", owner, resp.Body)
				}

				if !strings.Contains(resp.Body, fmt.Sprintf("<D:href>/%s/</D:href>", owner)) {
					t.Errorf("response of %s does not contain its own user. Got: %s", owner, resp.Body)
				}
			}
		}(i)
	}

	wg.Wait()
}

// The config fields left empty must fall back to the global setup.
func TestHandleRequestWithConfigFallsBackToTheGlobalSetup(t *testing.T) {
	defer saveGlobals()()

	global.Storage = newStorageWithEvent("Global")
	global.User = &data.CalUser{Name: "global-user"}

	resp := HandleRequestWithConfig(newRequest("GET", configTestResource, ""), Config{})

	test.AssertInt(resp.Status, http.StatusOK, t)
	test.AssertStr(resp.Body, eventData("Global"), t)
}

// The pre-existing entry points keep being served by the global setup.
func TestHandleRequestKeepsUsingTheGlobalSetup(t *testing.T) {
	defer saveGlobals()()

	SetupStorage(newStorageWithEvent("Global"))
	SetupUser("global-user")

	resp := HandleRequest(newRequest("GET", configTestResource, ""))
	test.AssertInt(resp.Status, http.StatusOK, t)
	test.AssertStr(resp.Body, eventData("Global"), t)

	requestXML := `
	<?xml version="1.0" encoding="utf-8" ?>
	<D:propfind xmlns:D="DAV:">
		<D:prop>
			<D:current-user-principal/>
		</D:prop>
	</D:propfind>
	`

	resp = HandleRequest(newRequest("PROPFIND", configTestResource, requestXML))
	expectedRespBody := fmt.Sprintf(`
	<?xml version="1.0" encoding="UTF-8"?>
	<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
		<D:response>
			<D:href>%s</D:href>
			<D:propstat>
				<D:prop>
					<D:current-user-principal>
						<D:href>/global-user/</D:href>
					</D:current-user-principal>
				</D:prop>
				<D:status>HTTP/1.1 200 OK</D:status>
			</D:propstat>
		</D:response>
	</D:multistatus>
	`, configTestResource)

	test.AssertInt(resp.Status, 207, t)
	test.AssertMultistatusXML(resp.Body, expectedRespBody, t)

	// `HandleRequestWithStorage` keeps setting the global storage and using it
	resp = HandleRequestWithStorage(newRequest("GET", configTestResource, ""), newStorageWithEvent("Storage"))
	test.AssertInt(resp.Status, http.StatusOK, t)
	test.AssertStr(resp.Body, eventData("Storage"), t)
	test.AssertStr(eventDataOf(global.Storage, t), eventData("Storage"), t)
}

// ================ FUNCS ========================

func newStorageWithEvent(summary string) data.Storage {
	stg := test.NewMemoryStorage()
	stg.AddResource(configTestCollection, "")
	stg.AddResource(configTestResource, eventData(summary))

	return stg
}

func eventData(summary string) string {
	return fmt.Sprintf("BEGIN:VEVENT\nSUMMARY:%s\nEND:VEVENT", summary)
}

func eventDataOf(stg data.Storage, t *testing.T) string {
	resource, _, err := stg.GetResource(configTestResource)
	if err != nil {
		t.Error(err)
		return ""
	}

	content, _ := resource.GetContentData()

	return content
}

func newRequest(method, path, body string) *http.Request {
	request, err := http.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
	panicerr(err)

	return request
}

// Saves the current global setup and returns the func that restores it, so that
// tests messing with the globals do not leak into the other tests.
func saveGlobals() func() {
	storage, user, components := global.Storage, global.User, global.SupportedComponents

	return func() {
		global.Storage, global.User, global.SupportedComponents = storage, user, components
	}
}
