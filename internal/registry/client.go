package registry

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxResponse = 8 << 20

type Client struct {
	HTTP         *http.Client
	CacheDir     string
	BootstrapURL string
	IANAWHOIS    string
	Now          func() time.Time
	Warn         func(string)
	// DialWHOIS is injectable to verify that no registrar receives a query.
	DialWHOIS func(context.Context, string, string) (net.Conn, error)
	RunDig    func(context.Context, ...string) ([]byte, error)
}

func NewClient(cacheDir string) *Client {
	dialer := &net.Dialer{Timeout: 12 * time.Second}
	return &Client{
		HTTP: &http.Client{
			Timeout: 15 * time.Second,
			// Redirects are explicitly checked against the IANA endpoints by rdap().
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		CacheDir: cacheDir, BootstrapURL: "https://data.iana.org/rdap/dns.json",
		IANAWHOIS: "whois.iana.org", Now: time.Now, DialWHOIS: dialer.DialContext,
		RunDig: func(ctx context.Context, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, "dig", args...).Output()
		},
	}
}

func DefaultCacheDir() string {
	if dir := os.Getenv("DOMAIN_CACHE_DIR"); dir != "" {
		return dir
	}
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "domain-check")
	}
	return "" // An unavailable cache never prevents an online query.
}

func (c *Client) warning(message string) {
	if c.Warn != nil {
		c.Warn(message)
	}
}

func (c *Client) Lookup(ctx context.Context, domain string) (Record, error) {
	normalized, err := NormalizeDomain(domain)
	if err != nil {
		return Record{}, queryError("invalid-domain", err.Error())
	}
	domain = normalized
	if strings.HasSuffix(domain, ".edu.cn") {
		return c.eduCN(ctx, domain)
	}
	b, err := c.bootstrap(ctx)
	if err != nil {
		// Do not guess WHOIS when the presence of registry RDAP is unknown.
		return Record{}, err
	}
	if endpoints := b.endpoints(domain); len(endpoints) > 0 {
		return c.rdap(ctx, domain, endpoints)
	}
	tld := domain[strings.LastIndexByte(domain, '.')+1:]
	host, err := c.whoisServer(ctx, tld)
	if err != nil {
		return Record{}, err
	}
	response, err := c.whois(ctx, host, domain)
	if err != nil {
		return Record{}, queryError("whois-failed", err.Error())
	}
	return parseWHOIS(domain, response)
}

func readLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxResponse+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponse {
		return nil, fmt.Errorf("registry response exceeds %d bytes", maxResponse)
	}
	return data, nil
}
