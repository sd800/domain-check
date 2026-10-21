package registry

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	eduCNEndpoint = "https://www.nic.edu.cn/cgi-bin/reg/otherobj"
	eduCNTimeout  = 3 * time.Second
)

func (c *Client) eduCN(ctx context.Context, domain string) (Record, error) {
	if record, err := c.eduCNRegistry(ctx, domain); err == nil {
		return record, nil
	}
	if err := ctx.Err(); err != nil {
		return Record{}, queryError("edu-cn-failed", err.Error())
	}
	// DNS is a separate operation; it must not inherit the expired HTTP context.
	dnsCtx, cancel := context.WithTimeout(ctx, eduCNTimeout)
	defer cancel()
	data, err := c.RunDig(dnsCtx, "+time=2", "+tries=1", "+noall", "+answer", domain, "NS")
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
	ctx, cancel := context.WithTimeout(ctx, eduCNTimeout)
	defer cancel()
	client := *c.HTTP
	client.Timeout = eduCNTimeout
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
	// Decode before parsing HTML. GB2312 labels map to GBK for HTML documents;
	// the registry's legacy GBK encoding is also the default when undeclared.
	encoding, name, certain := charset.DetermineEncoding(data, contentType)
	if !certain && name == "windows-1252" {
		encoding = simplifiedchinese.GBK
	}
	decoded, err := encoding.NewDecoder().Bytes(data)
	if err != nil {
		return Record{}, err
	}
	root, err := html.Parse(bytes.NewReader(decoded))
	if err != nil {
		return Record{}, err
	}
	var blocks []string
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "pre" {
			var text strings.Builder
			preText(node, &text)
			blocks = append(blocks, text.String())
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	if len(blocks) == 0 {
		return Record{}, fmt.Errorf("registry response has no WHOIS text")
	}
	text := strings.Join(blocks, "\n\n")
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

func preText(node *html.Node, text *strings.Builder) {
	if node.Type == html.TextNode {
		text.WriteString(node.Data)
	} else if node.Type == html.ElementNode {
		switch node.Data {
		case "script", "style":
			return
		case "br":
			text.WriteByte('\n')
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		preText(child, text)
	}
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
