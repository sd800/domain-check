package cli

import (
	"strings"
	"testing"

	"github.com/sd800/domain-check/internal/output"
	"github.com/sd800/domain-check/internal/registry"
)

func TestDiagnosticsSanitizeEvenExternallyConstructedErrors(t *testing.T) {
	err := &registry.QueryError{Code: "rdap-failed", Detail: "server\x1b]52;c;YXR0YWNr\a\r\u202espoof"}
	for _, lang := range []output.Language{output.English, output.Chinese} {
		if message := errorText(lang, err); strings.ContainsAny(message, "\x1b\a\r\u202e") {
			t.Fatalf("unsafe diagnostic: %q", message)
		}
	}
}
