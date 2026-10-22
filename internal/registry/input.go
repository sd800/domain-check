package registry

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/publicsuffix"
)

// NormalizeInput accepts a domain or URL. Registry response names still use
// strict NormalizeDomain validation and can never be reinterpreted as URLs.
func NormalizeInput(input string) (string, error) {
	if len(input) > 1<<20 {
		return "", fmt.Errorf("input exceeds 1 MiB")
	}
	input = strings.TrimSpace(input)
	if strings.ContainsAny(input, "\r\n\t") {
		return "", fmt.Errorf("expected one domain or URL")
	}
	isURL := strings.Contains(input, "://") || strings.ContainsAny(input, "/?#:")
	if !isURL {
		return normalizeQueryIdentity(input)
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
	host, err := normalizeQueryIdentity(u.Hostname())
	if err != nil {
		return "", fmt.Errorf("URL must contain a valid public domain name")
	}
	suffix, icann := publicsuffix.PublicSuffix(host)
	if !strings.Contains(host, ".") && !icann {
		return "", fmt.Errorf("URL must contain a valid public domain name")
	}
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

// Single-label queries address the root registry. Service hosts and ordinary
// response identities continue to use strict multi-label NormalizeDomain rules.
func normalizeQueryIdentity(input string) (string, error) {
	if len(input) > 4096 || strings.ContainsFunc(input, unicode.IsControl) {
		return "", fmt.Errorf("invalid domain length or control character")
	}
	input = strings.TrimPrefix(strings.TrimSpace(input), ".")
	if domain, err := NormalizeDomain(input); err == nil {
		return domain, nil
	}
	return normalizeTLD(input)
}

func normalizeTLD(input string) (string, error) {
	domain, err := NormalizeDomain("tld." + input)
	if err != nil || strings.Count(domain, ".") != 1 {
		return "", fmt.Errorf("expected a domain or top-level domain such as cn")
	}
	return strings.TrimPrefix(domain, "tld."), nil
}
