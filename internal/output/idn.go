package output

import (
	"strings"

	"github.com/sd800/domain-check/internal/registry"
	"golang.org/x/net/idna"
)

// Derive the display name locally from the queried identity, without trusting
// an unrelated Unicode name supplied in a registry response.
func unicodeDomain(domain string) string {
	lower := strings.ToLower(domain)
	if !strings.HasPrefix(lower, "xn--") && !strings.Contains(lower, ".xn--") {
		return ""
	}
	ascii, err := registry.NormalizeDomain(domain)
	if err != nil {
		return ""
	}
	name, err := idna.Lookup.ToUnicode(ascii)
	if err != nil || name == ascii {
		return ""
	}
	if roundTrip, err := registry.NormalizeDomain(name); err != nil || roundTrip != ascii {
		return ""
	}
	return name
}
