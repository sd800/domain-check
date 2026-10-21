package registry

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	eduCNEndpoint    = "https://www.nic.edu.cn/cgi-bin/reg/otherobj"
	eduCNHTTPTimeout = 3 * time.Second
)

func (c *Client) eduCN(ctx context.Context, domain string) (Record, error) {
	if record, err := c.eduCNRegistry(ctx, domain); err == nil {
		return record, nil
	}
	if err := ctx.Err(); err != nil {
		return Record{}, queryError("edu-cn-failed", err.Error())
	}
	// Only HTTP has the three-second limit. Keep dig's default timeouts and
	// retries while preserving cancellation of the overall command.
	data, err := c.RunDig(ctx, "+noall", "+answer", domain, "NS")
	if err != nil {
		return Record{}, queryError("dns-failed", err.Error())
	}
	nameservers := parseDigNS(string(data))
	if len(nameservers) == 0 {
		// DNS absence says nothing about registration or availability.
		return Record{}, queryError("dns-failed", "no NS records returned")
	}
	return Record{Domain: domain, Nameservers: nameservers}, nil
}

func (c *Client) eduCNRegistry(ctx context.Context, domain string) (Record, error) {
	ctx, cancel := context.WithTimeout(ctx, eduCNHTTPTimeout)
	defer cancel()
	client := *c.HTTP
	client.Timeout = eduCNHTTPTimeout
	// No HTML link or redirect is allowed to initiate a registrar query.
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, eduCNEndpoint+"?query="+url.QueryEscape(domain), nil)
	if err != nil {
		return Record{}, err
	}
	req.Header.Set("Accept", "text/html")
	req.Header.Set("User-Agent", "Domain-Check-from-Registries/1.0 (+https://github.com/sd800/domain-check)")
	response, err := client.Do(req)
	if err != nil {
		return Record{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Record{}, fmt.Errorf("registry returned HTTP %d", response.StatusCode)
	}
	data, err := readLimited(response.Body)
	if err != nil {
		return Record{}, err
	}
	return parseEDUCN(domain, data, response.Header.Get("Content-Type"))
}

func parseEDUCN(domain string, data []byte, contentType string) (Record, error) {
	decoded, err := decodeEDUCN(data, contentType)
	if err != nil {
		return Record{}, err
	}
	text, err := eduCNPreText(decoded)
	if err != nil {
		return Record{}, err
	}
	var normalized strings.Builder
	var firstLine string
	inServers := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if firstLine == "" && trimmed != "" {
			firstLine = trimmed
		}
		if strings.EqualFold(Clean(trimmed), "Domain Servers in listed order:") {
			inServers = true
			continue
		}
		if inServers {
			if trimmed == "" {
				continue
			}
			if ns, err := NormalizeDomain(strings.Fields(trimmed)[0]); err == nil {
				fmt.Fprintln(&normalized, "Name Server: "+ns)
				continue
			}
			inServers = false
		}
		normalized.WriteString(line)
		normalized.WriteByte('\n')
	}
	record, err := parseWHOIS(domain, normalized.String())
	if err != nil {
		return Record{}, err
	}
	if record.Result == "" && record.Registrant == "" && !strings.Contains(firstLine, ":") {
		record.Registrant = useful(firstLine)
	}
	return record, nil
}

func parseDigNS(answer string) []string {
	var nameservers []string
	for _, line := range strings.Split(answer, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 5 || strings.HasPrefix(fields[0], ";") || !strings.EqualFold(fields[2], "IN") || !strings.EqualFold(fields[3], "NS") {
			continue
		}
		if name, err := NormalizeDomain(fields[4]); err == nil {
			nameservers = appendUnique(nameservers, name)
		}
	}
	return nameservers
}
