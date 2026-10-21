package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

var whoisHostPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]$`)

func validWHOISHost(host string) bool {
	return strings.Contains(host, ".") && !strings.Contains(host, "..") && whoisHostPattern.MatchString(host)
}

func (c *Client) whois(ctx context.Context, host, query string) (string, error) {
	if !validWHOISHost(host) || strings.ContainsAny(query, "\r\n") {
		return "", fmt.Errorf("invalid WHOIS destination or query")
	}
	conn, err := c.DialWHOIS(ctx, "tcp", net.JoinHostPort(host, "43"))
	if err != nil {
		return "", err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline := time.Now().Add(15 * time.Second)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(conn, "%s\r\n", query); err != nil {
		return "", err
	}
	data, err := readLimited(conn)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		// CNNIC's legacy WHOIS service may still emit GB18030/GBK.
		if strings.EqualFold(host, "whois.cnnic.cn") {
			data, err = simplifiedchinese.GB18030.NewDecoder().Bytes(data)
			if err != nil {
				return "", err
			}
		} else {
			return "", fmt.Errorf("WHOIS response uses an unsupported character encoding")
		}
	}
	return string(data), nil
}

type whoisCache struct {
	Host      string    `json:"host"`
	CheckedAt time.Time `json:"checked_at"`
}

func (c *Client) whoisServer(ctx context.Context, tld string) (string, error) {
	name := "whois/" + tld + ".json"
	var cached whoisCache
	data, _ := os.ReadFile(c.cachePath(name))
	_ = json.Unmarshal(data, &cached)
	validCache := validWHOISHost(cached.Host)
	if validCache && fresh(cached.CheckedAt, c.Now()) {
		return cached.Host, nil
	}
	response, err := c.whois(ctx, c.IANAWHOIS, tld)
	if err != nil {
		if validCache {
			c.warning("stale-whois")
			return cached.Host, nil
		}
		return "", queryError("iana-whois-failed", err.Error())
	}
	var host string
	known := false
	for _, line := range strings.Split(response, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "domain":
			known = strings.EqualFold(strings.TrimSpace(value), tld)
		case "whois":
			host = strings.ToLower(strings.TrimSpace(value))
		}
	}
	if !known {
		return "", queryError("unsupported-suffix", tld)
	}
	if !validWHOISHost(host) {
		return "", queryError("no-registry-service", tld)
	}
	cached = whoisCache{Host: host, CheckedAt: c.Now()}
	data, _ = json.Marshal(cached)
	if err := c.writeCache(name, data); err != nil {
		c.warning("cache-write-failed")
	}
	return host, nil
}

var whoisKeys = map[string]string{
	"domain": "domain", "domain name": "domain", "domainname": "domain", "[domain name]": "domain",
	"creation date": "registration", "created": "registration", "created on": "registration", "created date": "registration",
	"registration time": "registration", "registration date": "registration", "registered on": "registration", "[created on]": "registration",
	"expiry date": "expiration", "expiration date": "expiration", "expiration time": "expiration", "expires": "expiration",
	"expires on": "expiration", "expire date": "expiration", "registry expiry date": "expiration", "paid-till": "expiration", "[expires on]": "expiration",
	"updated date": "changed", "last changed": "changed", "last changed time": "changed", "last modified": "changed",
	"last update": "changed", "last updated": "changed", "changed": "changed", "[last updated]": "changed",
	"domain status": "status", "status": "status", "state": "status", "registration status": "status", "[status]": "status",
	"registrar": "registrar", "sponsoring registrar": "registrar", "registrar name": "registrar",
	"registrant": "registrant", "registrant name": "registrant", "registrant organization": "organization",
	"registrant organisation": "organization", "[registrant]": "registrant", "holder": "registrant", "org": "organization",
	"registrant email": "email", "registrant e-mail": "email", "contact email": "email", "[email]": "email",
	"registrant contact email": "email", "registrant contact e-mail": "email",
	"name server": "nameserver", "nameserver": "nameserver", "name servers": "nameserver", "nserver": "nameserver", "[name server]": "nameserver",
	"dnssec": "dnssec", "[dnssec]": "dnssec",
}

var reservedResponse = regexp.MustCompile(`(?i)^(?:this )?(?:domain(?: name)?(?:\s+\S+)?\s+(?:is )?reserved\b|reserved(?: domain| name)?[.!]?$)`)
var forbiddenResponse = regexp.MustCompile(`(?i)^(?:this )?(?:domain(?: name)?(?:\s+\S+)?\s+(?:is )?(?:not allowed|not permitted|prohibited)(?: for registration)?[.!]?$|registration (?:is )?(?:not allowed|not permitted|prohibited)[.!]?$)`)

func negativeWHOIS(line string) Result {
	line = strings.ToLower(strings.TrimSpace(line))
	if reservedResponse.MatchString(line) {
		return Reserved
	}
	if forbiddenResponse.MatchString(line) {
		return NotAllowed
	}
	for _, prefix := range []string{"no match for ", "no matching record", "no entries found", "no data found", "no information available about domain name", "not found:", "domain not found", "no such domain", "no object found"} {
		if strings.HasPrefix(line, prefix) {
			return NoRecord
		}
	}
	if line == "not found" || line == "not found." {
		return NoRecord
	}
	if line == "status: free" || line == "status: available" || line == "domain is not registered" || line == "domain not registered" {
		return NotRegistered
	}
	return ""
}

func parseWHOIS(domain, response string) (Record, error) {
	r := Record{Domain: domain}
	var activeKey, organization string
	found := false
	for _, raw := range strings.Split(response, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			activeKey = ""
			continue
		}
		if strings.HasPrefix(line, "%") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ">>>") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "limit exceeded") || strings.Contains(lower, "quota exceeded") || strings.HasPrefix(lower, "too many queries") || strings.Contains(lower, "access denied") {
			return Record{}, queryError("whois-failed", line)
		}
		if result := negativeWHOIS(line); result != "" {
			r.Result = result
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		// JPRS uses [Field Name] rather than a colon.
		if !ok && strings.HasPrefix(line, "[") {
			if end := strings.IndexByte(line, ']'); end >= 0 {
				key, value, ok = line[:end+1], line[end+1:], true
			}
		}
		if ok {
			activeKey = whoisKeys[strings.ToLower(strings.TrimSpace(key))]
			value = useful(value)
		} else if activeKey != "" && len(raw) > len(strings.TrimLeft(raw, " \t")) {
			value = useful(line)
		} else {
			activeKey = ""
			continue
		}
		if value == "" || activeKey == "" {
			continue
		}
		switch activeKey {
		case "domain":
			normalized, err := NormalizeDomain(value)
			if err != nil || normalized != domain {
				return Record{}, queryError("invalid-response", "WHOIS response domain does not match the query")
			}
			found = true
		case "registration":
			r.Registration = formatTime(value)
		case "expiration":
			r.Expiration = formatTime(value)
		case "changed":
			r.Changed = formatTime(value)
		case "status":
			if index := strings.Index(value, " http"); index >= 0 {
				value = value[:index]
			}
			r.Statuses = appendUnique(r.Statuses, value)
		case "registrar":
			if r.Registrar == "" {
				r.Registrar = value
			}
		case "registrant":
			if r.Registrant == "" {
				r.Registrant = value
			}
		case "organization":
			organization = value
		case "email":
			r.RegistrantEmail = appendUnique(r.RegistrantEmail, value)
		case "nameserver":
			ns := strings.ToLower(strings.TrimSuffix(strings.Fields(value)[0], "."))
			if _, err := NormalizeDomain(ns); err == nil {
				r.Nameservers = appendUnique(r.Nameservers, ns)
			}
		case "dnssec":
			r.DNSSEC = value
		}
		// Multiline sections repeat only for nameservers and status.
		if activeKey != "nameserver" && activeKey != "status" && value != "" {
			activeKey = ""
		}
	}
	if r.Result != "" {
		// Never print stale/contradictory fields beside a negative result.
		if found && (r.Registration != "" || len(r.Nameservers) > 0) {
			return Record{}, queryError("invalid-response", "contradictory WHOIS response")
		}
		return Record{Domain: domain, Result: r.Result}, nil
	}
	if !found {
		return Record{}, queryError("invalid-response", "unrecognized or empty WHOIS response")
	}
	if organization != "" {
		r.Registrant = organization
	}
	return r, nil
}
