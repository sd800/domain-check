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

func TestForcedIANAUpdateReplacesBothFreshCaches(t *testing.T) {
	const dns = `{"services":[[["com"],["https://new.registry.test/"]]]}`
	tlds := tldFixture(t, map[string]string{"cn": "new.registry.test", "edu": "other.registry.test"})
	calls := 0
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodGet || r.Header.Get("Accept") != "application/json" {
			t.Fatalf("unexpected update request: %s %s", r.Method, r.URL)
		}
		switch r.URL.String() {
		case "https://data.iana.org/rdap/dns.json":
			return reply(r, http.StatusOK, dns), nil
		case "https://data.iana.org/TLD/v1/tlds.json":
			return reply(r, http.StatusOK, string(tlds)), nil
		default:
			t.Fatalf("queried a domain or third party: %s", r.URL)
			return nil, nil
		}
	})
	seedBootstrap(t, c, `[[["com"],["https://old.registry.test/"]]]`)
	seedTLDs(t, c, map[string]string{"cn": "old.registry.test"})
	if err := c.UpdateIANA(context.Background()); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string][]byte{"dns.json": []byte(dns), "tlds.json": tlds} {
		data, err := os.ReadFile(c.cachePath(name))
		info, statErr := os.Stat(c.cachePath(name))
		if err != nil || statErr != nil || !bytes.Equal(data, want) || !info.ModTime().Equal(c.Now()) {
			t.Fatalf("cache %s differs after refresh: %q, %v, %v", name, data, err, statErr)
		}
	}
	if _, err := c.bootstrap(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.tlds(context.Background()); err != nil || calls != 2 {
		t.Fatalf("fresh documents not reused: %v, calls=%d", err, calls)
	}
	if err := c.UpdateIANA(context.Background()); err != nil || calls != 4 {
		t.Fatalf("explicit update skipped fresh documents: %v, calls=%d", err, calls)
	}
	assertServiceMetadataOnly(t, c, []string{"dns.json", "tlds.json"})
}

func TestFailedForcedUpdatePreservesBothCaches(t *testing.T) {
	for _, endpoint := range []string{"dns.json", "tlds.json"} {
		for _, test := range []struct {
			name, body string
			status     int
			networkErr error
		}{
			{"HTTP failure", "unavailable", 503, nil},
			{"network failure", "", 0, errors.New("connection failed")},
			{"invalid JSON", "{broken", 200, nil},
			{"empty data", `{}`, 200, nil},
			{"invalid endpoint", `{"services":[[["com"],["file:///tmp/invalid"]]],"meta":{"specversion":1},"domains":[{"name":"cn","status":"active","whois_server":"https://registrar.test/"}]}`, 200, nil},
		} {
			t.Run(endpoint+"/"+test.name, func(t *testing.T) {
				c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
					if filepath.Base(r.URL.Path) != endpoint {
						return reply(r, 200, `{"services":[[["com"],["https://new.registry.test/"]]]}`), nil
					}
					if test.networkErr != nil {
						return nil, test.networkErr
					}
					return reply(r, test.status, test.body), nil
				})
				seedBootstrap(t, c, `[[["com"],["https://old.registry.test/"]]]`)
				seedTLDs(t, c, map[string]string{"cn": "old.registry.test"})
				old := make(map[string][]byte)
				mtime := c.Now().Add(-15 * 24 * time.Hour)
				for _, name := range []string{"dns.json", "tlds.json"} {
					var err error
					old[name], err = os.ReadFile(c.cachePath(name))
					if err != nil {
						t.Fatal(err)
					}
					if err := os.Chtimes(c.cachePath(name), mtime, mtime); err != nil {
						t.Fatal(err)
					}
				}
				c.Warn = func(code string) { t.Errorf("explicit update fell back to cache: %s", code) }
				var query *QueryError
				if err := c.UpdateIANA(context.Background()); !errors.As(err, &query) || query.Code != "iana-update-failed" {
					t.Fatalf("did not report update failure: %v", err)
				}
				for name, want := range old {
					got, readErr := os.ReadFile(c.cachePath(name))
					info, statErr := os.Stat(c.cachePath(name))
					if readErr != nil || statErr != nil || !bytes.Equal(got, want) || !info.ModTime().Equal(mtime) {
						t.Fatalf("failed update changed %s: %v, %v", name, readErr, statErr)
					}
				}
				assertServiceMetadataOnly(t, c, []string{"dns.json", "tlds.json"})
			})
		}
	}
}

func TestForcedUpdateReportsUnwritableCache(t *testing.T) {
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		if filepath.Base(r.URL.Path) == "tlds.json" {
			return reply(r, 200, string(tldFixture(t, map[string]string{"cn": "registry.test"}))), nil
		}
		return reply(r, 200, `{"services":[[["com"],["https://registry.test/"]]]}`), nil
	})
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	c.CacheDir = path
	var query *QueryError
	if err := c.UpdateIANA(context.Background()); !errors.As(err, &query) || query.Code != "iana-cache-failed" {
		t.Fatalf("cache-write failure was reported as success: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "unchanged" {
		t.Fatalf("cache path was damaged: %q, %v", data, err)
	}
}

func TestForcedUpdateReportsPartialSave(t *testing.T) {
	const dns = `{"services":[[["com"],["https://new.registry.test/"]]]}`
	tlds := tldFixture(t, map[string]string{"cn": "registry.test"})
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		if filepath.Base(r.URL.Path) == "tlds.json" {
			return reply(r, 200, string(tlds)), nil
		}
		return reply(r, 200, dns), nil
	})
	seedBootstrap(t, c, `[[["com"],["https://old.registry.test/"]]]`)
	if err := os.Mkdir(c.cachePath("tlds.json"), 0700); err != nil {
		t.Fatal(err)
	}
	var query *QueryError
	if err := c.UpdateIANA(context.Background()); !errors.As(err, &query) || query.Code != "iana-cache-failed" {
		t.Fatalf("partial save reported success: %v", err)
	}
	if data, err := os.ReadFile(c.cachePath("dns.json")); err != nil || string(data) != dns {
		t.Fatalf("successful first-file refresh lost: %q, %v", data, err)
	}
	files, err := os.ReadDir(c.CacheDir)
	if err != nil || len(files) != 2 {
		t.Fatalf("partial save left temporary data: %v, %v", files, err)
	}
}
