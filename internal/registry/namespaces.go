package registry

import (
	"strings"

	"golang.org/x/net/publicsuffix"
)

// A parent namespace object describes its delegation, not the queried child's
// registration. Private hosting suffixes and arbitrary mismatches do not qualify.
func namespaceParent(domain, parent string) bool {
	if !strings.Contains(parent, ".") || !strings.HasSuffix(domain, "."+parent) {
		return false
	}
	suffix, icann := publicsuffix.PublicSuffix(parent)
	// RNIDS returns the .mil.rs delegation for child queries; this delegated
	// military namespace is not currently represented as an ICANN PSL entry.
	return icann && suffix == parent || parent == "mil.rs"
}

func whoisNamespaceOnly(domain string, lines []whoisLine) bool {
	parent := ""
	for _, line := range lines {
		if !line.labeled || whoisKeys[line.key] != "domain" {
			continue
		}
		name, err := normalizeWHOISDomain(line.value)
		if err != nil || !namespaceParent(domain, name) || parent != "" && parent != name {
			return false
		}
		parent = name
	}
	return parent != ""
}

// Only fixed, independently documented namespace registry services are used.
// Response referrals cannot add destinations to this table.
func namespaceService(domain string) (rdap, whois string) {
	switch {
	case strings.HasSuffix(domain, ".gov.uk"):
		return "https://rdap.nominet.uk/gov.uk/", ""
	case strings.HasSuffix(domain, ".ac.uk"):
		return "", "whois.ja.net"
	}
	return "", ""
}

// Nominet's parent .uk interface does not expose registrations in these
// independently administered public-sector namespaces. A 404 there is not a
// determination by the namespace registry. Successful records remain usable.
func registryAbsence(domain, endpoint string) Result {
	if endpoint == rdapURL("https://rdap.nominet.uk/uk/", domain) {
		for _, suffix := range []string{"nhs.uk", "police.uk", "mod.uk", "mil.uk", "parliament.uk"} {
			if strings.HasSuffix(domain, "."+suffix) {
				return NotProvided
			}
		}
	}
	return NoRecord
}
