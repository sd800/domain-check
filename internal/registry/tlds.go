package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// tldRegistry indexes only active TLDs; an empty host means no published WHOIS.
// The cache retains IANA's complete document, including entries never queried.
type tldService struct {
	WHOIS       string
	CountryCode bool
}

// Nominet still publishes this registry service independently of IANA's
// optional WHOIS field. No response-provided destination can extend this rule.
func (s tldService) whoisHost(tld string) string {
	if s.WHOIS != "" {
		return s.WHOIS
	}
	if s.CountryCode && tld == "uk" {
		return "whois.nic.uk"
	}
	return ""
}

type tldRegistry map[string]tldService

func decodeTLDs(data []byte) (tldRegistry, error) {
	var document struct {
		Meta struct {
			Version int `json:"specversion"`
		} `json:"meta"`
		Domains []struct {
			Name   string          `json:"name"`
			Type   string          `json:"type"`
			Status string          `json:"status"`
			WHOIS  json.RawMessage `json:"whois_server"`
		} `json:"domains"`
	}
	if err := unmarshalResponse(data, &document); err != nil {
		return nil, err
	}
	if document.Meta.Version != 1 || len(document.Domains) == 0 {
		return nil, fmt.Errorf("invalid IANA TLD registry version or empty data")
	}
	registry := make(tldRegistry)
	seen := make(map[string]bool, len(document.Domains))
	for _, entry := range document.Domains {
		name, err := NormalizeDomain("iana." + entry.Name)
		if err != nil || strings.Contains(entry.Name, ".") || name != "iana."+entry.Name || seen[entry.Name] {
			return nil, fmt.Errorf("invalid or duplicate IANA TLD name")
		}
		seen[entry.Name] = true
		var host *string
		if len(entry.WHOIS) == 0 || json.Unmarshal(entry.WHOIS, &host) != nil || host != nil && !validWHOISHost(*host) {
			return nil, fmt.Errorf("invalid IANA WHOIS server")
		}
		switch entry.Status {
		case "active":
			service := tldService{CountryCode: entry.Type == "country-code"}
			if host != nil {
				service.WHOIS = strings.ToLower(*host)
			}
			registry[entry.Name] = service
		case "retired", "eligible", "reserved":
			// A registry service on an inactive TLD must not receive a query.
		default:
			return nil, fmt.Errorf("invalid IANA TLD status")
		}
	}
	if len(registry) == 0 {
		return nil, fmt.Errorf("IANA TLD registry has no active entries")
	}
	return registry, nil
}

func (c *Client) tlds(ctx context.Context) (tldRegistry, error) {
	return loadMetadata(c, ctx, "tlds.json", c.TLDsURL, "tld-data-failed", "stale-tlds", decodeTLDs)
}

// whoisServer uses the bulk IANA registry without asking IANA about a queried suffix.
func (c *Client) whoisServer(ctx context.Context, tld string) (string, error) {
	registry, err := c.tlds(ctx)
	if err != nil {
		return "", err
	}
	service, known := registry[tld]
	if !known {
		return "", queryError("unsupported-suffix", tld)
	}
	host := service.whoisHost(tld)
	if host == "" {
		return "", queryError("no-registry-service", tld)
	}
	return host, nil
}
