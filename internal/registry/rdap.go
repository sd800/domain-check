package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type rdapEntity struct {
	Roles      []string          `json:"roles"`
	VCardArray []json.RawMessage `json:"vcardArray"`
	Entities   []rdapEntity      `json:"entities"`
}

type rdapDomain struct {
	ObjectClass string   `json:"objectClassName"`
	LDHName     string   `json:"ldhName"`
	UnicodeName string   `json:"unicodeName"`
	Status      []string `json:"status"`
	Events      []struct {
		Action string `json:"eventAction"`
		Date   string `json:"eventDate"`
	} `json:"events"`
	Entities    []rdapEntity `json:"entities"`
	Nameservers []struct {
		LDHName string `json:"ldhName"`
	} `json:"nameservers"`
	SecureDNS *struct {
		DelegationSigned *bool             `json:"delegationSigned"`
		DSData           []json.RawMessage `json:"dsData"`
	} `json:"secureDNS"`
	ErrorCode int `json:"errorCode"`
}

func rdapURL(base, domain string) string {
	return strings.TrimRight(base, "/") + "/domain/" + domain
}

// A redirect must still name the same domain query at an IANA-listed registry
// endpoint. Links, entity URLs, and arbitrary same-host paths are never trusted.
func allowedRedirect(target *url.URL, endpoints []string, domain string) bool {
	if target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return false
	}
	for _, endpoint := range endpoints {
		allowed, _ := url.Parse(rdapURL(endpoint, domain))
		if strings.EqualFold(target.Host, allowed.Host) &&
			(target.Scheme == allowed.Scheme || allowed.Scheme == "http" && target.Scheme == "https") &&
			strings.TrimSuffix(target.EscapedPath(), "/") == allowed.EscapedPath() {
			return true
		}
	}
	return false
}

func (c *Client) rdap(ctx context.Context, domain string, endpoints []string) (Record, error) {
	var lastErr error
	for _, endpoint := range endpoints {
		record, err := c.rdapEndpoint(ctx, domain, rdapURL(endpoint, domain), endpoints)
		if err == nil {
			return record, nil
		}
		lastErr = err
		if e, ok := err.(*QueryError); ok && (e.Code == "rate-limited" || e.Code == "access-denied") {
			break
		}
	}
	// Having an RDAP entry means WHOIS is never used, including on RDAP failure.
	return Record{}, lastErr
}

func (c *Client) rdapEndpoint(ctx context.Context, domain, endpoint string, endpoints []string) (Record, error) {
	// Enforce redirect policy even if a caller injected a default http.Client.
	client := *c.HTTP
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	visited := map[string]bool{}
	for range 6 {
		if visited[endpoint] {
			return Record{}, queryError("redirect-blocked", "registry redirect loop")
		}
		visited[endpoint] = true
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return Record{}, queryError("rdap-failed", err.Error())
		}
		req.Header.Set("Accept", "application/rdap+json, application/json")
		req.Header.Set("User-Agent", "Domain-Check-from-Registries/1.0 (+https://github.com/sd800/domain-check)")
		response, err := client.Do(req)
		if err != nil {
			return Record{}, queryError("rdap-failed", err.Error())
		}
		status := response.StatusCode
		if status >= 300 && status < 400 {
			target, err := response.Location()
			response.Body.Close()
			if err != nil || !allowedRedirect(target, endpoints, domain) {
				return Record{}, queryError("redirect-blocked", "redirect outside the IANA-listed registry endpoints")
			}
			endpoint = target.String()
			continue
		}
		if status == http.StatusNotFound {
			response.Body.Close()
			return Record{Domain: domain, Result: NoRecord}, nil
		}
		if status != http.StatusOK {
			detail := fmt.Sprintf("HTTP %d", status)
			if retry := response.Header.Get("Retry-After"); status == 429 && retry != "" {
				detail += "; Retry-After: " + retry
			}
			response.Body.Close()
			code := "rdap-failed"
			if status == 429 {
				code = "rate-limited"
			} else if status == 401 || status == 403 {
				code = "access-denied"
			}
			return Record{}, queryError(code, detail)
		}
		data, err := readLimited(response.Body)
		response.Body.Close()
		if err != nil {
			return Record{}, queryError("rdap-failed", err.Error())
		}
		return parseRDAP(domain, data)
	}
	return Record{}, queryError("redirect-blocked", "too many registry redirects")
}

func parseRDAP(domain string, data []byte) (Record, error) {
	var d rdapDomain
	if err := json.Unmarshal(data, &d); err != nil {
		return Record{}, queryError("invalid-response", "invalid RDAP JSON: "+err.Error())
	}
	if d.ErrorCode != 0 || d.ObjectClass != "domain" {
		return Record{}, queryError("invalid-response", "response is not an RDAP domain object")
	}
	name := d.LDHName
	if name == "" {
		name = d.UnicodeName
	}
	normalized, err := NormalizeDomain(name)
	if err != nil || normalized != domain {
		return Record{}, queryError("invalid-response", "response domain does not match the query")
	}
	r := Record{Domain: domain}
	for _, event := range d.Events {
		switch event.Action {
		case "registration":
			r.Registration = formatTime(event.Date)
		case "expiration":
			r.Expiration = formatTime(event.Date)
		case "last changed":
			r.Changed = formatTime(event.Date)
		}
	}
	for _, status := range d.Status {
		r.Statuses = appendUnique(r.Statuses, status)
	}
	collectEntities(d.Entities, &r)
	for _, ns := range d.Nameservers {
		r.Nameservers = appendUnique(r.Nameservers, strings.ToLower(strings.TrimSuffix(ns.LDHName, ".")))
	}
	if d.SecureDNS != nil {
		if signed := d.SecureDNS.DelegationSigned; signed != nil {
			if *signed {
				r.DNSSEC = "signed"
			} else {
				r.DNSSEC = "unsigned"
			}
		} else if len(d.SecureDNS.DSData) > 0 {
			r.DNSSEC = "signed"
		}
	}
	return r, nil
}

func collectEntities(entities []rdapEntity, r *Record) {
	for _, entity := range entities {
		for _, role := range entity.Roles {
			if role != "registrar" && role != "registrant" {
				continue
			}
			name, emails := vcard(entity.VCardArray)
			if role == "registrar" && r.Registrar == "" {
				r.Registrar = name
			} else if role == "registrant" {
				if r.Registrant == "" {
					r.Registrant = name
				}
				for _, email := range emails {
					r.RegistrantEmail = appendUnique(r.RegistrantEmail, email)
				}
			}
		}
		collectEntities(entity.Entities, r)
	}
}

func vcard(card []json.RawMessage) (string, []string) {
	if len(card) != 2 {
		return "", nil
	}
	var properties [][]json.RawMessage
	if json.Unmarshal(card[1], &properties) != nil {
		return "", nil
	}
	var name, organization string
	var emails []string
	for _, property := range properties {
		if len(property) < 4 {
			continue
		}
		var kind string
		_ = json.Unmarshal(property[0], &kind)
		var value any
		_ = json.Unmarshal(property[3], &value)
		s := useful(vcardText(value))
		switch kind {
		case "fn":
			name = s
		case "org":
			organization = s
		case "email":
			emails = appendUnique(emails, s)
		}
	}
	if organization != "" {
		name = organization
	}
	return name, emails
}

func vcardText(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case []any:
		var parts []string
		for _, part := range value {
			if text := useful(vcardText(part)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}
