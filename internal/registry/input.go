package registry

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// NormalizeInput accepts a domain or URL. Registry response names still use
// strict NormalizeDomain validation and can never be reinterpreted as URLs.
func NormalizeInput(input string) (string, error) {
	input = strings.TrimSpace(input)
	if strings.ContainsAny(input, "\r\n\t") {
		return "", fmt.Errorf("expected one domain or URL")
	}
	isURL := strings.Contains(input, "://") || strings.ContainsAny(input, "/?#:")
	if !isURL {
		return NormalizeDomain(input)
	}
	value := input
	if !strings.Contains(value, "://") {
		// Scheme-less web addresses are accepted; email addresses are not URLs.
		if strings.Contains(strings.SplitN(value, "/", 2)[0], "@") {
			return "", fmt.Errorf("expected a domain or URL with a hostname")
		}
		value = "//" + strings.TrimPrefix(value, "//")
	}
	u, err := url.Parse(value)
	if err != nil || u.Opaque != "" || u.Hostname() == "" {
		// Do not echo the URL: it may include credentials or private query data.
		return "", fmt.Errorf("invalid URL hostname")
	}
	host, err := NormalizeDomain(u.Hostname())
	if err != nil {
		return "", fmt.Errorf("URL must contain a valid public domain name")
	}
	suffix, icann := publicsuffix.PublicSuffix(host)
	// Private hosting suffixes describe tenancy, not registry registrations.
	// For example, a URL on user.github.io is a query for github.io at .io.
	for !icann && strings.Contains(suffix, ".") {
		suffix = suffix[strings.IndexByte(suffix, '.')+1:]
		suffix, icann = publicsuffix.PublicSuffix(suffix)
	}
	if host == suffix {
		return host, nil // A multi-label namespace such as net.cn can itself be queried.
	}
	parent := strings.TrimSuffix(host, "."+suffix)
	if index := strings.LastIndexByte(parent, '.'); index >= 0 {
		parent = parent[index+1:]
	}
	return parent + "." + suffix, nil
}
