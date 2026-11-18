package registry

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

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
	if prefix, target, ok := strings.Cut(input, ":"); ok && strings.EqualFold(prefix, "site") {
		input = strings.TrimSpace(target)
	}
	if !strings.ContainsAny(input, "/?#:@") {
		host, err := normalizeQueryIdentity(input)
		if err != nil {
			return "", err
		}
		return registryDomain(host), nil
	}
	value := input
	scheme, afterColon, _ := strings.Cut(input, ":")
	if !strings.HasPrefix(afterColon, "//") || strings.ContainsAny(scheme, "/?#@") {
		// Keep mail links distinct from scheme-less credentials.
		if strings.EqualFold(scheme, "mailto") {
			return "", fmt.Errorf("invalid URL hostname")
		}
		// Force authority parsing so a user:password pair is not mistaken for a scheme.
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
	if !strings.Contains(host, ".") {
		if _, icann := publicsuffix.PublicSuffix(host); !icann {
			return "", fmt.Errorf("URL must contain a valid public domain name")
		}
	}
	return registryDomain(host), nil
}

// registryDomain selects the same registry boundary for bare hosts and URLs.
// Its caller validates the complete host before any subdomain is removed.
func registryDomain(host string) string {
	suffix, icann := publicsuffix.PublicSuffix(host)
	// Private hosting suffixes describe tenancy, not registry registrations.
	// For example, a URL on user.github.io is a query for github.io at .io.
	for !icann && strings.Contains(suffix, ".") {
		suffix = suffix[strings.IndexByte(suffix, '.')+1:]
		suffix, icann = publicsuffix.PublicSuffix(suffix)
	}
	if host == suffix {
		return host // A multi-label namespace such as net.cn can itself be queried.
	}
	parent := strings.TrimSuffix(host, "."+suffix)
	if index := strings.LastIndexByte(parent, '.'); index >= 0 {
		parent = parent[index+1:]
	}
	return parent + "." + suffix
}

// Single-label queries address the root registry. Service hosts and ordinary
// response identities continue to use strict multi-label NormalizeDomain rules.
func normalizeQueryIdentity(input string) (string, error) {
	if len(input) > 4096 || strings.ContainsFunc(input, unicode.IsControl) {
		return "", fmt.Errorf("invalid domain length or control character")
	}
	input = strings.TrimSpace(input)
	// IDNA recognizes these four dot separators. Ignore exactly one leading dot.
	first, width := utf8.DecodeRuneInString(input)
	if strings.ContainsRune(".。．｡", first) {
		input = input[width:]
	}
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
