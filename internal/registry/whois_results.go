package registry

import (
	"regexp"
	"strings"
)

var reservedResponse = regexp.MustCompile(`(?i)^(?:this )?(?:domain(?: name)?\s+(?:is )?reserved[.!]?$|reserved (?:domain|name)[.!]?$)`)
var forbiddenResponse = regexp.MustCompile(`(?i)^(?:this )?(?:domain(?: name)?(?:\s+\S+)?\s+(?:is )?(?:not allowed|not permitted|prohibited)(?: for registration)?[.!]?$|registration (?:is )?(?:not allowed|not permitted|prohibited)[.!]?$)`)
var commentedWHOISError = regexp.MustCompile(`(?i)^error:\s*(?:\d+:\s*)?(.*)$`)
var numberedWHOISError = regexp.MustCompile(`^-?\d+:\s*%?\s*(.*)$`)
var namedWHOISResult = regexp.MustCompile(`(?i)^(?:domain\s+)?(\S+)\s+(is free|is reserved|has been reserved by .+?)[.!]?$`)
var namedWHOISMissing = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^no such domain\s+["']?([^\s"']+?)["']?[.!]?$`),
	regexp.MustCompile(`(?i)^(?:the\s+)?domain\s+["']?([^\s"']+?)["']?\s+(?:was\s+)?not found[.!]?$`),
	regexp.MustCompile(`(?i)^no (?:match(?: found)?|matching record|record found|data found) for\s+(?:domain\s*:?\s*)?["']?([^\s"']+?)["']?[.!]?$`),
	regexp.MustCompile(`(?i)^([^\s"']+):\s*no entries found[.!]?$`),
	regexp.MustCompile(`(?i)^([^\s"']+)\s+-\s+no match[.!]?$`),
}
var namedWHOISAvailable = regexp.MustCompile(`(?i)^domain\s+([^\s"']+)\s+is available for (?:purchase|registration)[.!]?$`)

// Service and query-syntax failures are operational errors, never availability.
func whoisFailure(line string) bool {
	message := strings.TrimLeft(strings.TrimSpace(line), "%>* ")
	if match := commentedWHOISError.FindStringSubmatch(message); match != nil {
		message = match[1]
	}
	if match := numberedWHOISError.FindStringSubmatch(message); match != nil {
		message = match[1]
	}
	lower := strings.ToLower(Clean(message))
	if strings.HasPrefix(lower, "warning: query rate is now high,") || strings.HasPrefix(lower, "blacklisted: you have exceeded the query limit") || strings.HasPrefix(lower, "subdomains such as ") && strings.HasSuffix(lower, " are not supported") || strings.HasPrefix(lower, "invalid domain name: ") {
		return true
	}
	switch strings.TrimRight(lower, ".!") {
	case "malformed request", "invalid input", "invalid request", "invalid query", "invalid query format", "invalid domain name", "invalid .lt domain name", "invalid pattern", "object is blocked", "wrong top level domain name in query", "wrong second level domain name in query", "wrong syntax", "cadena_invalida/invalid_string", "too many requests", "maximum query rate reached", "the whois service is temporarily unavailable", "token: invalid_query_error", "tld not supported by this registry interface", "erreur syntax [ domain ]", "whois.nic.bo solo acepta consultas con dominios .bo":
		return true
	}
	return false
}

func negativeWHOIS(line string) Result {
	line = strings.ToLower(strings.TrimSpace(line))
	switch strings.TrimRight(line, ".!:") {
	case "the domain name is not available", "this name is not available for registration":
		return Unavailable
	case "iana reserved":
		return Reserved
	case "domain name matches a restricted word", "stop-list: domain can not be registered":
		return NotAllowed
	}
	if line == "this domain is currently not available for registration. please select other domain." {
		return Unavailable
	}
	if line == "the domain name you apply can not be registered online. please consult your domain name registrar" {
		return Restricted // CNNIC does not distinguish namespace names from other restrictions here.
	}
	if strings.TrimSuffix(line, ".") == "the domain name you requested was not found in our database" {
		return NoRecord
	}
	if reservedResponse.MatchString(line) {
		return Reserved
	}
	if forbiddenResponse.MatchString(line) {
		return NotAllowed
	}
	for _, prefix := range []string{"no matching record", "no entries found", "no data found", "no information available about domain name", "not found:", "domain not found", "no such domain", "no object found", "the queried object does not exist:"} {
		if strings.HasPrefix(line, prefix) {
			return NoRecord
		}
	}
	switch strings.TrimRight(line, ".!") {
	case "not found", "no found", "no entries found", "no match", "nothing found", "nothing found for this query", "object does not exist", "no_se_encontro_el_objeto/object_not_found", "this query returned 0 objects":
		return NoRecord
	}
	if line == "domain is not registered" || line == "domain not registered" || strings.TrimRight(line, ".!") == "the domain has not been registered" {
		return NotRegistered
	}
	if line == "reserved domain name" || line == "reserved domain name." || line == "this name is reserved by the registry." || line == "this name is reserved by the registry" {
		return Reserved
	}
	return ""
}

func negativeWHOISFor(domain, line string) Result {
	if match := namedWHOISResult.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
		if name, err := NormalizeDomain(match[1]); err == nil && name == domain {
			if strings.EqualFold(match[2], "is free") {
				return NotRegistered
			}
			return Reserved
		}
	}
	for _, pattern := range namedWHOISMissing {
		if match := pattern.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
			if name, err := NormalizeDomain(match[1]); err == nil && name == domain {
				return NoRecord
			}
			return ""
		}
	}
	if match := namedWHOISAvailable.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
		if name, err := NormalizeDomain(match[1]); err == nil && name == domain {
			return NotRegistered
		}
	}
	return negativeWHOIS(line)
}

// Interpret availability labels only as a domain status, never as contact data
// or a statement in the registry's terms of use.
func negativeWHOISStatus(value string) Result {
	switch strings.ToLower(Clean(value)) {
	case "free", "available", "not registered", "not registered, and may be available if valid":
		return NotRegistered
	case "no object found", "not found":
		return NoRecord
	case "reserved":
		return Reserved
	case "prohibited string - domain cannot be registered":
		return NotAllowed
	}
	return ""
}
