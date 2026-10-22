package registry

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type bootstrap struct {
	Services [][][]string `json:"services"`
}

func decodeBootstrap(data []byte) (bootstrap, error) {
	var b bootstrap
	if err := unmarshalResponse(data, &b); err != nil {
		return b, err
	}
	if len(b.Services) == 0 {
		return b, fmt.Errorf("empty IANA bootstrap data")
	}
	seen := make(map[string]bool)
	for _, service := range b.Services {
		if len(service) != 2 || len(service[0]) == 0 || len(service[1]) == 0 {
			return b, fmt.Errorf("invalid IANA service entry")
		}
		for _, suffix := range service[0] {
			name, err := NormalizeDomain("iana." + suffix)
			if err != nil || name != "iana."+suffix || seen[suffix] {
				return b, fmt.Errorf("invalid or duplicate IANA suffix")
			}
			seen[suffix] = true
		}
		for _, endpoint := range service[1] {
			if !validEndpoint(endpoint) {
				return b, fmt.Errorf("invalid IANA endpoint")
			}
		}
	}
	return b, nil
}

func validEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && validServiceHost(u.Hostname()) && u.Host == u.Hostname() && u.User == nil && u.RawQuery == "" && !u.ForceQuery && !strings.Contains(endpoint, "#") && safeServicePath(u)
}

// Match whole suffix labels, selecting the longest matching bootstrap entry.
func (b bootstrap) endpoints(domain string) []string {
	var endpoints []string
	longest := -1
	for _, service := range b.Services {
		for _, suffix := range service[0] {
			if suffix != "" && strings.HasSuffix(domain, "."+strings.ToLower(suffix)) && len(suffix) > longest {
				endpoints = append([]string(nil), service[1]...)
				longest = len(suffix)
			}
		}
	}
	// Like client.rdap.org, prefer an IANA-listed HTTPS endpoint.
	sort.SliceStable(endpoints, func(i, j int) bool {
		return strings.HasPrefix(endpoints[i], "https:") && !strings.HasPrefix(endpoints[j], "https:")
	})
	return endpoints
}

// One calendar month, clamped at month end (January 31 -> February 28/29).
func nextMonth(t time.Time) time.Time {
	first := time.Date(t.Year(), t.Month()+1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	last := first.AddDate(0, 1, -1).Day()
	day := min(t.Day(), last)
	return first.AddDate(0, 0, day-1)
}

func fresh(checked, now time.Time) bool {
	return !checked.IsZero() && !checked.After(now) && now.Before(nextMonth(checked))
}

func (c *Client) cachePath(name string) string {
	if c.CacheDir == "" {
		return ""
	}
	return filepath.Join(c.CacheDir, name)
}

func (c *Client) bootstrap(ctx context.Context) (bootstrap, error) {
	return loadMetadata(c, ctx, "dns.json", c.BootstrapURL, "bootstrap-failed", "stale-bootstrap", decodeBootstrap)
}

func (c *Client) writeCache(name string, data []byte) error {
	path := c.cachePath(name)
	if path == "" {
		return fmt.Errorf("no writable cache directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".domain-cache-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// mtime means last successful fetch, not IANA's publication timestamp.
	if err := os.Chtimes(f.Name(), c.Now(), c.Now()); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
