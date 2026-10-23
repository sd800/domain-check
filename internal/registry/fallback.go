package registry

import (
	"context"
	"fmt"
	"strings"
)

// RDAP remains the primary source. A failure may use only the active
// registry's published WHOIS service; successful replies are never merged.
func (c *Client) rdapWithWHOIS(ctx context.Context, domain string, endpoints []string) (Record, error) {
	r, rdapErr := c.rdap(ctx, domain, endpoints)
	if rdapErr == nil || ctx.Err() != nil {
		return r, rdapErr
	}
	if q, ok := rdapErr.(*QueryError); ok && (q.Code == "rate-limited" || q.Code == "redirect-blocked") {
		return Record{}, rdapErr
	}
	services, err := c.tlds(ctx)
	if err != nil {
		return Record{}, rdapErr
	}
	tld := domain[strings.LastIndexByte(domain, '.')+1:]
	service, ok := services[tld]
	host := service.whoisHost(tld)
	if !ok || host == "" {
		return Record{}, rdapErr
	}
	if ctx.Err() != nil {
		return Record{}, rdapErr
	}
	response, err := c.whois(ctx, host, registryWHOISQuery(host, domain))
	if err == nil {
		r, err = parseWHOIS(domain, response)
		if err == nil {
			if r.Result == NoRecord && host == "whois.nic.uk" && delegatedUKChild(domain) {
				r.Result = NotProvided
			}
			return r, nil
		}
	}
	return Record{}, queryError("fallback-failed", fmt.Sprintf("RDAP: %v; WHOIS: %v", rdapErr, err))
}
