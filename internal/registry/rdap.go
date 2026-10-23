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
	Links       []struct {
		Rel   string `json:"rel"`
		Href  string `json:"href"`
		Title string `json:"title"`
	} `json:"links"`
	Events []struct {
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
	if target.User != nil || target.RawQuery != "" || target.ForceQuery || target.Fragment != "" || target.RawFragment != "" {
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
		if e, ok := err.(*QueryError); ok && (e.Code == "rate-limited" || e.Code == "access-denied" || e.Code == "redirect-blocked") {
			break
		}
	}
	// The caller decides whether this failure permits a registry WHOIS fallback.
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
			return Record{Domain: domain, Result: registryAbsence(domain, endpoint)}, nil
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
	if err := unmarshalResponse(data, &d); err != nil {
		return Record{}, queryError("invalid-response", "invalid RDAP JSON: "+err.Error())
	}
	if strings.HasSuffix(domain, ".sn") && d.ObjectClass == "" && d.ErrorCode == 0 {
		return parseSNEnvelope(domain, data)
	}
	if d.ErrorCode != 0 || d.ObjectClass != "domain" {
		return Record{}, queryError("invalid-response", "response is not an RDAP domain object")
	}
	if len(d.Events) > 256 || len(d.Status) > 256 || len(d.Nameservers) > 256 || len(d.Links) > 256 || !boundedEntities(d.Entities, new(int)) {
		return Record{}, queryError("invalid-response", "RDAP record exceeds parsing limits")
	}
	name := d.LDHName
	if name == "" {
		name = d.UnicodeName
	}
	normalize := NormalizeDomain
	isTLD := !strings.Contains(domain, ".")
	if isTLD {
		normalize = normalizeTLD
	}
	normalized, err := normalize(name)
	if err != nil || normalized != domain {
		if err == nil && !isTLD && namespaceParent(domain, normalized) {
			if d.UnicodeName == "" {
				return Record{Domain: domain, Result: NotProvided}, nil
			}
			if alternate, err := normalize(d.UnicodeName); err == nil && alternate == normalized {
				return Record{Domain: domain, Result: NotProvided}, nil
			}
		}
		return Record{}, queryError("invalid-response", "response domain does not match the query")
	}
	if d.UnicodeName != "" {
		unicodeName, err := normalize(d.UnicodeName)
		if err != nil || unicodeName != domain {
			return Record{}, queryError("invalid-response", "response Unicode domain does not match the query")
		}
	}
	r := Record{Domain: domain, TLD: isTLD}
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
	if isTLD {
		collectTLDEntities(d.Entities, &r)
		for _, link := range d.Links {
			if link.Rel != "related" || link.Title != "Registration URL" {
				continue
			}
			text := useful(link.Href)
			u, err := url.Parse(text)
			if err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil {
				r.RegistrationURL = text
				break
			}
		}
	} else {
		collectEntities(d.Entities, &r)
	}
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
	if err := validateRecordBudget(r); err != nil {
		return Record{}, err
	}
	return r, nil
}

func boundedEntities(entities []rdapEntity, count *int) bool {
	*count += len(entities)
	if *count > 512 {
		return false
	}
	for _, entity := range entities {
		if len(entity.Roles) > 16 || len(entity.VCardArray) > 2 {
			return false
		}
		if len(entity.VCardArray) == 2 {
			var properties []json.RawMessage
			if json.Unmarshal(entity.VCardArray[1], &properties) != nil || len(properties) > 256 {
				return false
			}
		}
		if !boundedEntities(entity.Entities, count) {
			return false
		}
	}
	return true
}

func collectEntities(entities []rdapEntity, r *Record) {
	for _, entity := range entities {
		for _, role := range entity.Roles {
			if role != "registrar" && role != "registrant" {
				continue
			}
			name, emails, country := vcard(entity.VCardArray)
			if role == "registrar" && r.Registrar == "" {
				r.Registrar = name
				if name != "" && country != "" {
					r.Registrar += " [" + country + "]"
				}
			} else if role == "registrant" {
				if r.Registrant == "" {
					r.Registrant = name
					if strings.HasSuffix(r.Domain, ".gov") {
						collectGovAddress(entity.VCardArray, r, name != "")
					} else {
						collectRegistrantContact(entity.VCardArray, r, name != "")
					}
				}
				for _, email := range emails {
					r.RegistrantEmail = appendUnique(r.RegistrantEmail, email)
				}
			}
		}
		collectEntities(entity.Entities, r)
	}
}

func vcard(card []json.RawMessage) (string, []string, string) {
	contact := readContactCard(card, false)
	name := contact.Organization
	if name == "" {
		name = contact.Name
	}
	return name, contact.Emails, contact.Country
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
