package output

import (
	"strings"

	"github.com/sd800/domain-check/internal/registry"
)

// Derive the display name locally from the queried identity, without trusting
// an unrelated Unicode name supplied in a registry response.
func unicodeDomain(domain string) string {
	lower := strings.ToLower(domain)
	if !strings.HasPrefix(lower, "xn--") && !strings.Contains(lower, ".xn--") {
		return ""
	}
	return registry.UnicodeDomain(domain)
}
