package registry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sd800/domain-check/internal/safeio"
)

// loadMetadata caches whole IANA documents, never a query-dependent selection.
// A failed download or validation may reuse validated stale data without renewing it.
func loadMetadata[T any](c *Client, ctx context.Context, name, endpoint, failure, stale string, decode func([]byte) (T, error)) (T, error) {
	path := c.cachePath(name)
	cached, info, readErr := safeio.ReadFile(path, maxResponse)
	value, parseErr := decode(cached)
	validCache := readErr == nil && parseErr == nil
	if validCache && fresh(info.ModTime(), c.Now()) {
		return value, nil
	}
	data, err := c.downloadMetadata(ctx, endpoint)
	var updated T
	if err == nil {
		updated, err = decode(data)
	}
	if err != nil {
		if validCache {
			c.warning(stale)
			return value, nil
		}
		return updated, queryError(failure, err.Error())
	}
	if err := c.writeCache(name, data); err != nil {
		c.warning("cache-write-failed")
	}
	return updated, nil
}

// downloadMetadata refuses redirects and bounds both IANA JSON responses.
func (c *Client) downloadMetadata(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Domain-Check-from-Registries/1.0 (+https://github.com/sd800/domain-check)")
	client := *c.HTTP
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IANA returned HTTP %d", response.StatusCode)
	}
	return readLimited(response.Body)
}

// UpdateIANA validates both full documents before replacing either cached file.
// Replacement is atomic per file; an incomplete update must never report success.
func (c *Client) UpdateIANA(ctx context.Context) error {
	dns, err := c.downloadMetadata(ctx, c.BootstrapURL)
	if err == nil {
		_, err = decodeBootstrap(dns)
	}
	if err != nil {
		return queryError("iana-update-failed", "dns.json: "+err.Error())
	}
	tlds, err := c.downloadMetadata(ctx, c.TLDsURL)
	if err == nil {
		_, err = decodeTLDs(tlds)
	}
	if err != nil {
		return queryError("iana-update-failed", "tlds.json: "+err.Error())
	}
	for _, file := range []struct {
		name string
		data []byte
	}{{"dns.json", dns}, {"tlds.json", tlds}} {
		if err := c.writeCache(file.name, file.data); err != nil {
			return queryError("iana-cache-failed", file.name+": "+err.Error())
		}
	}
	return nil
}
