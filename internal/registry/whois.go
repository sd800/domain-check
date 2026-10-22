package registry

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func validWHOISHost(host string) bool { return validServiceHost(host) }

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
	if !validWHOISHost(host) || !validWHOISQuery(host, query) {
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
	return decodeWHOIS(host, data)
}

func decodeWHOIS(host string, data []byte) (string, error) {
	var err error
	if !utf8.Valid(data) {
		// CNNIC's legacy WHOIS service may still emit GB18030/GBK.
		if strings.EqualFold(host, "whois.cnnic.cn") {
			data, err = simplifiedchinese.GB18030.NewDecoder().Bytes(data)
			if err != nil {
				return "", err
			}
		} else if strings.EqualFold(host, "whois.nic.hu") || strings.EqualFold(host, "whois.dns.lu") || strings.EqualFold(host, "whois.registro.br") || strings.EqualFold(host, "whois.fi") {
			// These registry services use ISO-8859-1 legacy responses.
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
