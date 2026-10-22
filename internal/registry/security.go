package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// unmarshalResponse bounds recursive JSON processing, including embedded jCards.
func unmarshalResponse(data []byte, value any) error {
	if len(data) > maxResponse {
		return fmt.Errorf("JSON response exceeds %d bytes", maxResponse)
	}
	depth, quoted, escaped, start := 0, false, false, 0
	for i, b := range data {
		if quoted {
			if i-start > 64<<10 {
				return fmt.Errorf("JSON string exceeds 65536 bytes")
			}
			if escaped {
				escaped = false
			} else if b == '\\' {
				escaped = true
			} else if b == '"' {
				quoted = false
			}
			continue
		}
		switch b {
		case '"':
			quoted = true
			start = i
		case '{', '[':
			depth++
			if depth > 64 {
				return fmt.Errorf("JSON nesting exceeds 64 levels")
			}
		case '}', ']':
			depth--
		}
	}
	return json.Unmarshal(data, value)
}

// validServiceHost accepts canonical DNS hostnames, never IP literals or ports.
func validServiceHost(host string) bool {
	normalized, err := NormalizeDomain(host)
	return err == nil && normalized == strings.ToLower(host)
}

func safeServicePath(u *url.URL) bool {
	if u.RawPath != "" || strings.Contains(u.Path, "\\") {
		return false
	}
	for _, segment := range strings.Split(u.Path, "/") {
		if segment == "." || segment == ".." {
			return false
		}
		for _, ch := range segment {
			if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || strings.ContainsRune("-._~", ch)) {
				return false
			}
		}
	}
	return true
}

func validWHOISQuery(host, query string) bool {
	domain := query
	switch strings.ToLower(host) {
	case "whois.jprs.jp":
		domain = strings.TrimSuffix(domain, "/e")
	case "whois.denic.de":
		domain = strings.TrimPrefix(domain, "-T dn,ace ")
	case "whois.punktum.dk", "whois.dk-hostmaster.dk":
		domain = strings.TrimPrefix(domain, "--charset=utf-8 ")
	}
	normalized, err := NormalizeDomain(domain)
	return err == nil && domain == normalized && query == registryWHOISQuery(host, normalized)
}

var nonPublicNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func publicAddress(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() {
		return false
	}
	// Well-known IPv6 transition forms must not disguise a private IPv4 target.
	if addr.Is6() {
		bytes := addr.As16()
		if netip.MustParsePrefix("64:ff9b::/96").Contains(addr) {
			return publicAddress(netip.AddrFrom4([4]byte{bytes[12], bytes[13], bytes[14], bytes[15]}))
		}
		if netip.MustParsePrefix("2002::/16").Contains(addr) {
			return publicAddress(netip.AddrFrom4([4]byte{bytes[2], bytes[3], bytes[4], bytes[5]}))
		}
	}
	for _, prefix := range nonPublicNetworks {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

// dialRegistry resolves once, validates every address, then dials those exact IPs.
// No second hostname resolution can redirect the connection into a local network.
func dialRegistry(ctx context.Context, network, address string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	host, port, err := net.SplitHostPort(address)
	if err != nil || !validServiceHost(host) || (port != "43" && port != "80" && port != "443") {
		return nil, fmt.Errorf("invalid registry destination")
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	return dialPublicAddresses(ctx, network, port, addresses, (&net.Dialer{Timeout: 12 * time.Second}).DialContext)
}

func validateRecordBudget(r Record) error {
	if len(r.Statuses) > 256 || len(r.Nameservers) > 256 || len(r.RegistrantEmail) > 256 || len(r.RegistrantAddress) > 256 || len(r.AdminPhones) > 256 || len(r.AdminEmails) > 256 {
		return queryError("invalid-response", "registry record exceeds field limits")
	}
	return nil
}

func dialPublicAddresses(ctx context.Context, network, port string, addresses []netip.Addr, dial func(context.Context, string, string) (net.Conn, error)) (net.Conn, error) {
	if len(addresses) == 0 {
		return nil, fmt.Errorf("registry destination has no addresses")
	}
	for _, addr := range addresses {
		if !publicAddress(addr) {
			return nil, fmt.Errorf("registry destination resolves to a non-public address")
		}
	}
	var err error
	for _, addr := range addresses {
		var conn net.Conn
		conn, err = dial(ctx, network, net.JoinHostPort(addr.String(), port))
		if err == nil {
			return conn, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, err
}
