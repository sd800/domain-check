package registry

import (
	"context"
	"net/http"
	"testing"
)

func TestDomainAndURLInputs(t *testing.T) {
	for input, want := range map[string]string{
		"example.com":      "example.com",
		"net.cn":           "net.cn",
		"home.cn":          "home.cn",
		"example.net.cn":   "example.net.cn",
		"WWW.EXAMPLE.COM.": "www.example.com",
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
		"https://user.github.io/project":                           "github.io",
		"https://user.blogspot.co.uk/post":                         "blogspot.co.uk",
		"https://user:password@www.example.com/private":            "example.com",
		"https://www.example.com./":                                "example.com",
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
	} {
		if _, err := NormalizeInput(input); err == nil {
			t.Errorf("accepted invalid input %q", input)
		}
	}
}

func TestURLLookupOnlyContactsRegistry(t *testing.T) {
	domain, err := NormalizeInput("https://user:secret@www.example.com/private?token=secret#section")
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
