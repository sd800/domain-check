package registry

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBulkWHOISCacheDoesNotRevealQueriedSuffixes(t *testing.T) {
	data := tldFixture(t, map[string]string{"cn": "cn.registry.test", "edu": "edu.registry.test", "im": "never-queried.registry.test"})
	fetches := 0
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		fetches++
		if r.URL.String() != "https://data.iana.org/TLD/v1/tlds.json" || r.URL.RawQuery != "" {
			t.Fatalf("suffix-dependent discovery request: %s", r.URL)
		}
		return reply(r, 200, string(data)), nil
	})
	seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
	legacy := c.cachePath("whois")
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "cn.json"), []byte(`{"host":"untrusted.test","checked_at":"2026-10-21T00:00:00Z"}`), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	c.DialWHOIS = func(ctx context.Context, network, address string) (net.Conn, error) {
		calls++
		if address != "cn.registry.test:43" && address != "edu.registry.test:43" {
			t.Errorf("unexpected WHOIS discovery or referral: %s", address)
		}
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			query, _ := bufio.NewReader(server).ReadString('\n')
			fmt.Fprintf(server, "Domain Name: %s\nRegistrant: Example Owner\nRegistrar WHOIS Server: registrar.test\n", strings.TrimSpace(query))
		}()
		return client, nil
	}
	for _, domain := range []string{"example.cn", "example.edu", "example.cn"} {
		if _, err := c.Lookup(context.Background(), domain); err != nil {
			t.Fatal(err)
		}
		cached, err := os.ReadFile(c.cachePath("tlds.json"))
		info, statErr := os.Stat(c.cachePath("tlds.json"))
		if err != nil || statErr != nil || !bytes.Equal(cached, data) || !info.ModTime().Equal(c.Now()) {
			t.Fatalf("cache varied with lookup: %v, %v", err, statErr)
		}
	}
	if fetches != 1 || calls != 3 {
		t.Fatalf("fetches=%d, WHOIS queries=%d", fetches, calls)
	}
	if _, err := os.Lstat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy suffix cache remains: %v", err)
	}
	assertServiceMetadataOnly(t, c, []string{"dns.json", "tlds.json"}, "example.cn", "example.edu", "Example Owner", "registrar.test", "checked_at")
}

func TestTLDCacheMonthlyRefreshAndValidation(t *testing.T) {
	data := tldFixture(t, map[string]string{"cn": "registry.test"})
	fetches := 0
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
		fetches++
		return reply(r, 200, string(data)), nil
	})
	now := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)
	c.Now = func() time.Time { return now }
	if _, err := c.tlds(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = time.Date(2026, 2, 28, 11, 59, 59, 0, time.UTC)
	if _, err := c.tlds(context.Background()); err != nil || fetches != 1 {
		t.Fatalf("premature refresh: %v, fetches=%d", err, fetches)
	}
	now = now.Add(time.Second)
	if _, err := c.tlds(context.Background()); err != nil || fetches != 2 {
		t.Fatalf("monthly refresh missing: %v, fetches=%d", err, fetches)
	}
	mtime := now
	now = now.AddDate(0, 1, 0)
	c.HTTP.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		return reply(r, 200, `{"meta":{"specversion":1},"domains":[{"name":"cn","status":"active","whois_server":"registry.test:443"}]}`), nil
	})
	warnings := 0
	c.Warn = func(code string) {
		if code == "stale-tlds" {
			warnings++
		}
	}
	for i := 0; i < 2; i++ {
		if host, err := c.whoisServer(context.Background(), "cn"); err != nil || host != "registry.test" {
			t.Fatalf("validated stale cache not used: %q, %v", host, err)
		}
	}
	info, err := os.Stat(c.cachePath("tlds.json"))
	if err != nil || !info.ModTime().Equal(mtime) || warnings != 2 {
		t.Fatalf("stale cache renewed: %v, warnings=%d", err, warnings)
	}
	if err := os.WriteFile(c.cachePath("tlds.json"), []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.tlds(context.Background()); err == nil {
		t.Fatal("invalid cached and downloaded data accepted")
	}
}

func TestTLDRegistryRejectsInvalidRouting(t *testing.T) {
	for _, entry := range []string{
		`{"name":"cn","status":"active"}`,
		`{"name":"cn","status":"active","whois_server":false}`,
		`{"name":"cn","status":"active","whois_server":""}`,
		`{"name":"cn","status":"active","whois_server":"https://registry.test"}`,
		`{"name":"cn","status":"active","whois_server":"registry.test:43"}`,
		`{"name":"cn","status":"mystery","whois_server":null}`,
		`{"name":"other.cn","status":"active","whois_server":null}`,
		`{"name":"https://cn","status":"active","whois_server":null}`,
		`{"name":"中国","status":"active","whois_server":null}`,
		`{"name":"cn","status":"active","whois_server":null},{"name":"cn","status":"active","whois_server":"other.test"}`,
	} {
		if _, err := decodeTLDs([]byte(`{"meta":{"specversion":1},"domains":[` + entry + `]}`)); err == nil {
			t.Errorf("invalid routing accepted: %s", entry)
		}
	}
	for _, data := range []string{`{}`, `{"meta":{"specversion":2},"domains":[{"name":"cn","status":"active","whois_server":null}]}`, `{"meta":{"specversion":1},"domains":[]}`} {
		if _, err := decodeTLDs([]byte(data)); err == nil {
			t.Errorf("invalid document accepted: %s", data)
		}
	}
}

func TestTLDRegistryMissingAndInactiveServices(t *testing.T) {
	const data = `{"meta":{"specversion":1},"domains":[{"name":"cn","status":"active","whois_server":null},{"name":"old","status":"retired","whois_server":"old.registry.test"},{"name":"reserved","status":"reserved","whois_server":null},{"name":"eligible","status":"eligible","whois_server":null}]}`
	c := fixtureClient(t, func(r *http.Request) (*http.Response, error) { return reply(r, 200, data), nil })
	seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
	for _, tld := range []string{"cn", "old", "reserved", "eligible", "unknown"} {
		record, err := c.Lookup(context.Background(), "example."+tld)
		var query *QueryError
		want := "unsupported-suffix"
		if tld == "cn" {
			want = "no-registry-service"
		}
		if !errors.As(err, &query) || query.Code != want || record.Result != "" {
			t.Errorf("%s: %+v, %v", tld, record, err)
		}
	}
}

func TestLegacyCacheCleanupDoesNotFollowSymlinks(t *testing.T) {
	c := NewClient(t.TempDir())
	other := t.TempDir()
	keep := filepath.Join(other, "keep.json")
	if err := os.WriteFile(keep, []byte("unrelated"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, c.cachePath("whois")); err != nil {
		t.Fatal(err)
	}
	if err := c.removeLegacyWHOISCache(); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(keep); err != nil || string(data) != "unrelated" {
		t.Fatalf("symlink target changed: %q, %v", data, err)
	}
	if _, err := os.Lstat(c.cachePath("whois")); !os.IsNotExist(err) {
		t.Fatalf("legacy link was not removed: %v", err)
	}
}

func TestIANAMetadataRedirectsAreNotFollowed(t *testing.T) {
	for _, name := range []string{"dns.json", "tlds.json"} {
		c := fixtureClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.Host != "data.iana.org" {
				t.Errorf("followed IANA metadata redirect: %s", r.URL)
			}
			response := reply(r, 302, "")
			response.Header.Set("Location", "https://registrar.test/metadata")
			return response, nil
		})
		var err error
		if name == "dns.json" {
			_, err = c.bootstrap(context.Background())
		} else {
			_, err = c.tlds(context.Background())
		}
		if err == nil {
			t.Fatalf("%s: accepted redirected metadata", name)
		}
	}
}
