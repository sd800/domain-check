package registry

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestForcedBootstrapUpdateReplacesFreshCache(t *testing.T) {
	const body = `{"services":[[["com"],["https://new.registry.test/"]]]}`
	calls := 0
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodGet || r.URL.String() != "https://data.iana.org/rdap/dns.json" || r.Header.Get("Accept") != "application/json" {
			t.Fatalf("unexpected update request: %s %s", r.Method, r.URL)
		}
		return reply(r, http.StatusOK, body), nil
	})
	seedBootstrap(t, c, `[[["com"],["https://old.registry.test/"]]]`)
	if err := c.UpdateBootstrap(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(c.cachePath("dns.json"))
	if err != nil || string(data) != body || calls != 1 {
		t.Fatalf("cache %q, %v, calls=%d", data, err, calls)
	}
	info, err := os.Stat(c.cachePath("dns.json"))
	if err != nil || !info.ModTime().Equal(c.Now()) {
		t.Fatalf("successful refresh did not reset cache age: %v, %v", info, err)
	}
	// The next ordinary lookup reuses the newly refreshed monthly cache.
	if _, err := c.bootstrap(context.Background()); err != nil || calls != 1 {
		t.Fatalf("cache was not reused: %v, calls=%d", err, calls)
	}
	// Repeating the explicit command must fetch again, even immediately.
	if err := c.UpdateBootstrap(context.Background()); err != nil || calls != 2 {
		t.Fatalf("explicit update used a fresh cache: %v, calls=%d", err, calls)
	}
	assertServiceMetadataOnly(t, c, []string{"dns.json"})
}

func TestFailedForcedUpdatePreservesExistingCache(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
		networkErr error
	}{
		{"HTTP failure", "unavailable", 503, nil},
		{"network failure", "", 0, errors.New("connection failed")},
		{"invalid JSON", "{broken", 200, nil},
		{"empty data", `{"services":[]}`, 200, nil},
		{"invalid endpoint", `{"services":[[["com"],["file:///tmp/invalid"]]]}`, 200, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
				if test.networkErr != nil {
					return nil, test.networkErr
				}
				return reply(r, test.status, test.body), nil
			})
			seedBootstrap(t, c, `[[["com"],["https://old.registry.test/"]]]`)
			old, err := os.ReadFile(c.cachePath("dns.json"))
			if err != nil {
				t.Fatal(err)
			}
			mtime := c.Now().Add(-15 * 24 * time.Hour)
			if err := os.Chtimes(c.cachePath("dns.json"), mtime, mtime); err != nil {
				t.Fatal(err)
			}
			c.Warn = func(code string) { t.Errorf("explicit update fell back to cache: %s", code) }
			err = c.UpdateBootstrap(context.Background())
			var query *QueryError
			if !errors.As(err, &query) || query.Code != "bootstrap-update-failed" {
				t.Fatalf("did not report update failure: %v", err)
			}
			got, readErr := os.ReadFile(c.cachePath("dns.json"))
			info, statErr := os.Stat(c.cachePath("dns.json"))
			if readErr != nil || statErr != nil || !bytes.Equal(got, old) || !info.ModTime().Equal(mtime) {
				t.Fatalf("failed update changed old cache: %v, %v", readErr, statErr)
			}
			assertServiceMetadataOnly(t, c, []string{"dns.json"})
		})
	}
}

func TestForcedUpdateReportsUnwritableCache(t *testing.T) {
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		return reply(r, 200, `{"services":[[["com"],["https://registry.test/"]]]}`), nil
	})
	// A regular file is an unwritable directory on all supported platforms,
	// including test runners with permissions that bypass chmod restrictions.
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	c.CacheDir = path
	var query *QueryError
	if err := c.UpdateBootstrap(context.Background()); !errors.As(err, &query) || query.Code != "bootstrap-cache-failed" {
		t.Fatalf("cache-write failure was reported as success: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "unchanged" {
		t.Fatalf("cache path was damaged: %q, %v", data, err)
	}
}
