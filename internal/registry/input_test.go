package registry

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestDomainAndURLInputs(t *testing.T) {
	for input, want := range map[string]string{
		"example.com":      "example.com",
		".github.io":       "github.io",
		".com":             "com",
		"cn":               "cn",
		"CN.":              "cn",
		"auto":             "auto",
		".中国":              "xn--fiqs8s",
		"https://cn/":      "cn",
		"cn/../other":      "cn",
		"net.cn":           "net.cn",
		"home.cn":          "home.cn",
		"example.net.cn":   "example.net.cn",
		"WWW.EXAMPLE.COM.": "example.com",
		"https://www.example.com:8443/path?item=1&lang=zh#section": "example.com",
		"http://mail.example.com/":                                 "example.com",
		"ftp://files.example.com/pub/":                             "example.com",
		"//www.example.com/path":                                   "example.com",
		"www.example.com/path":                                     "example.com",
		"example.com?item=1":                                       "example.com",
		"example.com:443":                                          "example.com",
		"https://books.example.co.uk/title":                        "example.co.uk",
		"https://department.neu.edu.cn/path":                       "neu.edu.cn",
		"https://net.cn/":                                          "net.cn",
		"https://home.cn/path":                                     "home.cn",
		"https://www.example.net.cn/path":                          "example.net.cn",
		"https://co.uk/":                                           "co.uk",
		"https://www.例子.中国/中文":                                     "xn--fsqu00a.xn--fiqs8s",
		"https://რეგისტრაცია.გე/?utm_source=chatgpt.com": "xn--lodaehvb5cdik4g.xn--node",
		"https://user.github.io/project":                 "github.io",
		"https://user.blogspot.co.uk/post":               "blogspot.co.uk",
		"https://user:password@www.example.com/private":  "example.com",
		"https://www.example.com./":                      "example.com",

		"hello@example.com":                                    "example.com",
		"alice:123456@example.com":                             "example.com",
		"//alice:123456@www.example.com:8443/private":          "example.com",
		"alice:p%40ss@www.例子.中国/path?next=https://example.com": "xn--fsqu00a.xn--fiqs8s",
		"site:example.com":                                     "example.com",
		" SITE: www.example.com ":                              "example.com",
		"site:https://www.example.com/":                        "example.com",
		"www.example.com/path?next=https://other.test":         "example.com",
		"www.example.com?next=https://other.test":              "example.com",
		"www.example.com#https://other.test":                   "example.com",
		"。中国":                                                  "xn--fiqs8s",
	} {
		t.Run(input, func(t *testing.T) {
			got, err := NormalizeInput(input)
			if err != nil || got != want {
				t.Fatalf("got %q, %v; want %q", got, err, want)
			}
		})
	}
	for _, input := range []string{
		"https://localhost/path", "https://127.0.0.1/", "https://[2001:db8::1]/",
		"https:///missing-host", "mailto:user@example.com",
		"file:///etc/passwd", "https://example.com:invalid/", "https://user:secret@/path",
		"https://-bad.com/", "https://exa\nmple.com/", "example.com\nother.com",
		"alice:secret@/path", "alice:secret@127.0.0.1", "alice:secret@\nexample.com",
		"alice:p%ZZ@www.example.com", "MAILTO:user@example.com", "site:", "site:example.com other",
		"www..example.com", "-bad.example.com", "bad_.example.com",
		strings.Repeat("a", 64) + ".example.com",
		"https://[www.example.com]/", "//[example.com]:443/", "owner:secret@[example.com]",
		"。．example.com", ".。example.com", "https://。．example.com/",
		"..github.io", "..cn", ".", "123", "-cn",
	} {
		if _, err := NormalizeInput(input); err == nil {
			t.Errorf("accepted invalid input %q", input)
		}
	}
}

func TestBareHostsAndURLsUseTheSameRegistryDomain(t *testing.T) {
	for host, want := range map[string]string{
		"www.example.org":               "example.org",
		"www.example.org.cn":            "example.org.cn",
		"www2.mail.example.co.uk":       "example.co.uk",
		"api.department.example.edu.cn": "example.edu.cn",
		"www.example。org．cn.":           "example.org.cn",
		"www.例子.中国":                     "xn--fsqu00a.xn--fiqs8s",
		"www.👩‍💻.ws":                    "xn--1ug1855pcha.ws",
		"www.user.github.io":            "github.io",
		"mail.user.blogspot.co.uk":      "blogspot.co.uk",
		"a.b.example.unknown-tld":       "example.unknown-tld",
		"a.b.example.ck":                "b.example.ck",
		"www.www.ck":                    "www.ck",
		"www.city.kawasaki.jp":          "city.kawasaki.jp",
		"www.gov.uk":                    "www.gov.uk",
		"www.com":                       "www.com",
		"www.example.com.other.test":    "other.test",
		"net.cn":                        "net.cn",
		"co.uk":                         "co.uk",
		"github.io":                     "github.io",
	} {
		for _, input := range []string{host, "." + host, "。" + host, "．" + host, "｡" + host, "site:" + host,
			"https://" + host + "/private", "//" + host,
			"user:secret@" + host + ":8443/path", "site:https://" + host + "/"} {
			t.Run(input, func(t *testing.T) {
				got, err := NormalizeInput(input)
				if err != nil || got != want {
					t.Fatalf("got %q, %v; want %q", got, err, want)
				}
			})
		}
	}
	for input, want := range map[string]string{
		"www.example.com@other.test":         "other.test",
		"https://example.com@www.other.test": "other.test",
	} {
		if got, err := NormalizeInput(input); err != nil || got != want {
			t.Fatalf("userinfo changed host selection: %q became %q, %v", input, got, err)
		}
	}
}

func TestTLDInputDoesNotRelaxServiceOrWHOISValidation(t *testing.T) {
	if _, err := normalizeQueryIdentity("cn\r\nother"); err == nil {
		t.Fatal("TLD query accepted line injection")
	}
	if _, err := NormalizeDomain("cn"); err == nil || validServiceHost("cn") || validWHOISQuery("whois.registry.test", "cn") {
		t.Fatal("single-label lookup relaxed transport validation")
	}
}

func TestURLLookupOnlyContactsRegistry(t *testing.T) {
	domain, err := NormalizeInput("site:user:secret@www.example.com/private?token=secret#section")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.String() != "https://registry.test/domain/example.com" || req.Header.Get("Authorization") != "" {
			t.Fatalf("URL components leaked outside domain extraction: %s", req.URL)
		}
		return reply(req, 200, `{"objectClassName":"domain","ldhName":"example.com"}`), nil
	})
	seedBootstrap(t, c, `[[["com"],["https://registry.test/"]]]`)
	if _, err := c.Lookup(context.Background(), domain); err != nil || calls != 1 {
		t.Fatalf("%v, calls %d", err, calls)
	}
}

func TestLeadingDotOnOrdinaryDomainKeepsRegistryRouting(t *testing.T) {
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://registry.test/domain/github.io" {
			t.Fatalf("leading dot changed routing: %s", req.URL)
		}
		return reply(req, 200, `{"objectClassName":"domain","ldhName":"github.io"}`), nil
	})
	seedBootstrap(t, c, `[[["io"],["https://registry.test/"]]]`)
	r, err := c.Lookup(context.Background(), ".github.io")
	if err != nil || r.Domain != "github.io" || r.TLD {
		t.Fatal(r, err)
	}
}
