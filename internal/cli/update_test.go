package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sd800/domain-check/internal/output"
	"github.com/sd800/domain-check/internal/registry"
)

type updateTransport func(*http.Request) (*http.Response, error)

func (f updateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUpdateCommandsRefreshOnlyBootstrapData(t *testing.T) {
	for _, lang := range []string{"en", "zh"} {
		for _, prefix := range []string{"", "-", "--"} {
			t.Run(lang+" "+prefix+"update", func(t *testing.T) {
				t.Setenv("LC_ALL", "en_US.UTF-8")
				t.Setenv("DOMAIN_CONFIG_DIR", t.TempDir())
				c := registry.NewClient(t.TempDir())
				calls := 0
				const data = `{"services":[[["com"],["https://registry.test/"]]]}`
				c.HTTP.Transport = updateTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.URL.String() != "https://data.iana.org/rdap/dns.json" {
						t.Fatalf("update queried a domain: %s", r.URL)
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(data)), Request: r}, nil
				})
				c.DialWHOIS = func(context.Context, string, string) (net.Conn, error) {
					t.Error("update must not perform WHOIS queries")
					return nil, fmt.Errorf("unexpected WHOIS query")
				}
				var out, errOut bytes.Buffer
				if code := run(context.Background(), []string{prefix + "update", lang}, &out, &errOut, "test", c); code != 0 || errOut.Len() != 0 || calls != 1 {
					t.Fatalf("exit=%d, stdout=%q, stderr=%q, calls=%d", code, out.String(), errOut.String(), calls)
				}
				want := "IANA DNS bootstrap data updated."
				if lang == "zh" {
					want = "IANA DNS 引导数据已更新。"
				}
				if strings.TrimSpace(out.String()) != want {
					t.Fatalf("unexpected update message: %q", out.String())
				}
				cached, err := os.ReadFile(filepath.Join(c.CacheDir, "dns.json"))
				files, listErr := os.ReadDir(c.CacheDir)
				if err != nil || listErr != nil || string(cached) != data || len(files) != 1 {
					t.Fatalf("unexpected cache data: %q, %v, %v, %v", cached, err, files, listErr)
				}
			})
		}
	}
}

func TestFailedUpdateCommandReportsFailure(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	t.Setenv("DOMAIN_CONFIG_DIR", t.TempDir())
	c := registry.NewClient(t.TempDir())
	c.HTTP.Transport = updateTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unavailable")), Request: r}, nil
	})
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"update"}, &out, &errOut, "test", c); code != 3 || out.Len() != 0 || !strings.Contains(errOut.String(), "Cannot update IANA DNS bootstrap data") {
		t.Fatalf("exit=%d, stdout=%q, stderr=%q", code, out.String(), errOut.String())
	}
}

func TestSpecialCommandSpellingsAndInvalidCombinations(t *testing.T) {
	for _, prefix := range []string{"", "-", "--"} {
		o, err := parseArgs([]string{"zh", prefix + "update"}, output.English)
		if err != nil || !o.update || o.intro || o.domain != "" || o.lang != output.Chinese {
			t.Fatalf("%supdate: %+v, %v", prefix, o, err)
		}
	}
	for _, args := range [][]string{{"update", "example.com"}, {"https://example.com/", "--update"}, {"---update"}, {"update", "--unknown"}} {
		if _, err := parseArgs(args, output.English); err == nil {
			t.Fatalf("accepted invalid update arguments: %v", args)
		}
	}
	t.Setenv("LC_ALL", "en_US.UTF-8")
	t.Setenv("DOMAIN_CONFIG_DIR", t.TempDir())
	c := registry.NewClient(t.TempDir())
	c.HTTP.Transport = updateTransport(func(r *http.Request) (*http.Response, error) {
		t.Fatal("help/version unexpectedly fetched data")
		return nil, fmt.Errorf("unexpected network request")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, command := range []string{"help", "version"} {
		for _, prefix := range []string{"", "-", "--"} {
			var out, errOut bytes.Buffer
			if code := run(ctx, []string{prefix + command}, &out, &errOut, "test", c); code != 0 || out.Len() == 0 || errOut.Len() != 0 {
				t.Fatalf("%s%s: exit=%d, stderr=%q", prefix, command, code, errOut.String())
			}
		}
	}
}
