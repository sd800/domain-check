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

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
)

var whoisHostPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]$`)

func validWHOISHost(host string) bool {
	return strings.Contains(host, ".") && !strings.Contains(host, "..") && whoisHostPattern.MatchString(host)
}

func registryWHOISQuery(host, domain string) string {
	switch strings.ToLower(host) {
	case "whois.jprs.jp":
		return domain + "/e"
	case "whois.denic.de":
		return "-T dn,ace " + domain
	case "whois.punktum.dk", "whois.dk-hostmaster.dk":
		return "--charset=utf-8 " + domain
	}
	return domain
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
		} else if strings.EqualFold(host, "whois.nic.hu") || strings.EqualFold(host, "whois.dns.lu") {
			// These registry services document ISO-8859-1 legacy responses.
			data, err = charmap.ISO8859_1.NewDecoder().Bytes(data)
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
			// IANA returns U-labels for IDN suffixes even when queried by
			// their A-label. Reuse strict domain validation before comparing.
			domain, err := NormalizeDomain("iana." + strings.TrimSpace(value))
			known = err == nil && domain == "iana."+tld
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
