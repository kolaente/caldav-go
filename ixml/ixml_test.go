package ixml

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

// Wraps the given content in a <D:multistatus> node, the same way the
// multistatus responses are built by the handlers.
func multistatus(content string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>` +
		`<D:multistatus ` + Namespaces() + `><D:response>` + content + `</D:response></D:multistatus>`
}

// Parses the given document and returns the character data of its first <D:href> node.
func firstHrefContent(t *testing.T, document string) string {
	t.Helper()

	decoder := xml.NewDecoder(strings.NewReader(document))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			t.Fatalf("no <D:href> found in %q", document)
		} else if err != nil {
			t.Fatalf("could not parse %q: %v", document, err)
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name != HREF_TG {
			continue
		}

		var content string
		if err := decoder.DecodeElement(&content, &start); err != nil {
			t.Fatalf("could not decode <D:href> in %q: %v", document, err)
		}

		return content
	}
}

func TestHrefTag(t *testing.T) {
	t.Run("escapes special characters", func(t *testing.T) {
		// An href coming straight from a calendar-multiget request body.
		href := `/test-data/report/x</D:href><INJECTED/><D:href>.ics`

		document := multistatus(HrefTag(href))

		if strings.Contains(document, "<INJECTED/>") {
			t.Fatalf("injected markup escaped the href node: %q", document)
		}

		if got := firstHrefContent(t, document); got != href {
			t.Fatalf("expected href %q, got %q", href, got)
		}
	})

	t.Run("escapes special characters in user names", func(t *testing.T) {
		// <D:current-user-principal> renders the user name as an href path.
		href := `/some<user>&co/`

		document := multistatus(Tag(CURRENT_USER_PRINCIPAL_TG, HrefTag(href)))

		if got := firstHrefContent(t, document); got != href {
			t.Fatalf("expected href %q, got %q", href, got)
		}
	})

	t.Run("round trips whitespace", func(t *testing.T) {
		href := "/test-data/a\nb\tc/"

		if got := firstHrefContent(t, multistatus(HrefTag(href))); got != href {
			t.Fatalf("expected href %q, got %q", href, got)
		}
	})

	t.Run("leaves paths without special characters untouched", func(t *testing.T) {
		expected := "<D:href>/test-data/propfind/123-456-789.ics</D:href>"

		if got := HrefTag("/test-data/propfind/123-456-789.ics"); got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("keeps an empty href self closing", func(t *testing.T) {
		if got := HrefTag(""); got != "<D:href/>" {
			t.Fatalf("expected %q, got %q", "<D:href/>", got)
		}
	})
}

// Tag is also called with content that is already XML, so it must not escape
// what it is given. Guards the call sites in handlers/multistatus.go.
func TestTagKeepsMarkupContentIntact(t *testing.T) {
	t.Run("nested tags", func(t *testing.T) {
		content := Tag(COLLECTION_TG, "") + Tag(CALENDAR_TG, "") + Tag(PRINCIPAL_TG, "")

		expected := "<D:resourcetype><D:collection/><C:calendar/><D:principal/></D:resourcetype>"
		if got := Tag(RESOURCE_TYPE_TG, content); got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("nested href tags", func(t *testing.T) {
		expected := "<C:calendar-home-set><D:href>/test-data/</D:href></C:calendar-home-set>"

		if got := Tag(CALENDAR_HOME_SET_TG, HrefTag("/test-data/")); got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("hand built tags", func(t *testing.T) {
		expected := `<C:supported-calendar-component-set><C:comp name="VEVENT"/></C:supported-calendar-component-set>`

		if got := Tag(SUPPORTED_CALENDAR_COMPONENT_SET_TG, `<C:comp name="VEVENT"/>`); got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("content already escaped by the caller", func(t *testing.T) {
		expected := "<C:calendar-data>SUMMARY:Party &amp; Fun</C:calendar-data>"

		if got := Tag(CALENDAR_DATA_TG, EscapeText("SUMMARY:Party & Fun")); got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})
}
