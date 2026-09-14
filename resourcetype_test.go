package caldav

import (
	"testing"

	"github.com/samedi/caldav-go/test"
)

// A collection holding calendars, like a calendar home set, is not a calendar itself.
// Clients such as Apple Calendar try to sync it as one otherwise and fail.
func TestPropfindResourceTypeOfNonCalendarCollection(t *testing.T) {
	stg := test.NewMemoryStorage()
	stg.AddNonCalendarCollection("/users/home/")
	stg.AddResource("/users/home/cal/", "")

	requestXML := `
	<?xml version="1.0" encoding="utf-8" ?>
	<D:propfind xmlns:D="DAV:">
		<D:prop>
			<D:resourcetype/>
		</D:prop>
	</D:propfind>
	`

	request := newRequest("PROPFIND", "/users/home/", requestXML)
	request.Header.Set("Depth", "1")
	resp := HandleRequestWithConfig(request, Config{Storage: stg})

	expectedRespBody := `
	<?xml version="1.0" encoding="UTF-8"?>
	<D:multistatus xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav" xmlns:CS="http://calendarserver.org/ns/">
		<D:response>
			<D:href>/users/home</D:href>
			<D:propstat>
				<D:prop>
					<D:resourcetype><D:collection/></D:resourcetype>
				</D:prop>
				<D:status>HTTP/1.1 200 OK</D:status>
			</D:propstat>
		</D:response>
		<D:response>
			<D:href>/users/home/cal</D:href>
			<D:propstat>
				<D:prop>
					<D:resourcetype><D:collection/><C:calendar/></D:resourcetype>
				</D:prop>
				<D:status>HTTP/1.1 200 OK</D:status>
			</D:propstat>
		</D:response>
	</D:multistatus>
	`

	test.AssertInt(resp.Status, 207, t)
	test.AssertMultistatusXML(resp.Body, expectedRespBody, t)
}
