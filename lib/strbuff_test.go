package lib

import (
	"strings"
	"testing"
)

func TestStringBufferWrite(t *testing.T) {
	t.Run("preserves percent signs without format arguments", func(t *testing.T) {
		var buffer StringBuffer

		input := strings.Join([]string{"SUMMARY:Improve coverage from 37", "% to 65", "%"}, "")
		write := buffer.Write
		write(input)

		if buffer.String() != input {
			t.Fatalf("expected %q, got %q", input, buffer.String())
		}
	})

	t.Run("formats provided arguments", func(t *testing.T) {
		var buffer StringBuffer

		buffer.Write("<%s>%d</%s>", "status", 200, "status")

		expected := "<status>200</status>"
		if buffer.String() != expected {
			t.Fatalf("expected %q, got %q", expected, buffer.String())
		}
	})
}
