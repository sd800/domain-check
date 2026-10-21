package registry

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const eduCNHTML = `<HEAD>
<TITLE>Whois Results for neu.edu.cn</TITLE>
<meta http-equiv="Content-Type" content="text/html; charset=gb2312">
</HEAD>
<BODY bgcolor=#ffffff>
<H1>Whois Query</H1>
<font>对象关键字查询<br>on accurate entity keys</font>
<H2>Whois <font color=red>neu.edu.cn</font> ?</H2>
<PRE>
Northeastern University
   Shenyang, LN 110004
   China

   Domain Name: <a href= >NEU.EDU.CN</a >

   Network Name:   NEU-CN

   Administrative Contact, Technical Contact:
      Zhankao Wen  (ZW45-CER)

   Domain Servers in listed order:

   ns.neu.edu.cn        202.118.1.35 2001:da8:9000::35
   ns1.neu.edu.cn       202.118.1.45 2001:da8:9000::45
   ns2.neu.edu.cn       202.112.31.197
</PRE>
<a href="https://registrar.test/entity/1">Related link</a>
</BODY>`

const eduCNDNSAnswer = `neu.edu.cn. 3600 IN NS ns.neu.edu.cn.
neu.edu.cn. 3600 IN NS ns1.neu.edu.cn.
neu.edu.cn. 3600 IN NS ns2.neu.edu.cn.
`

func gbkBytes(t *testing.T, s string) []byte {
	t.Helper()
	data, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEDUCNDecodesHTMLBeforeParsing(t *testing.T) {
	for _, test := range []struct {
		name, contentType, source, registrant string
		gbk                                   bool
	}{
		{"GB2312 meta", "text/html", eduCNHTML, "Northeastern University", true},
		{"GBK header", "text/html; charset=gbk", strings.ReplaceAll(eduCNHTML, "Northeastern University", "东北大学"), "东北大学", true},
		{"GBK default", "text/html", strings.ReplaceAll(strings.ReplaceAll(eduCNHTML, "charset=gb2312", ""), "Northeastern University", "东北大学"), "东北大学", true},
		{"UTF-8 declaration", "text/html; charset=utf-8", strings.ReplaceAll(eduCNHTML, "Northeastern University", "东北大学 &amp; Research"), "东北大学 & Research", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := []byte(test.source)
			if test.gbk {
				data = gbkBytes(t, test.source)
			}
			r, err := parseEDUCN("neu.edu.cn", data, test.contentType)
			want := Record{Domain: "neu.edu.cn", Registrant: test.registrant, Nameservers: []string{"ns.neu.edu.cn", "ns1.neu.edu.cn", "ns2.neu.edu.cn"}}
			if err != nil || !reflect.DeepEqual(r, want) {
				t.Fatalf("got %+v, %v; want %+v", r, err, want)
			}
		})
	}
}

func TestEDUCNUsesDedicatedRegistryBeforeBootstrap(t *testing.T) {
	calls := 0
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.String() != "https://www.nic.edu.cn/cgi-bin/reg/otherobj?query=neu.edu.cn" {
			t.Fatalf("unexpected query %s", req.URL)
		}
		deadline, ok := req.Context().Deadline()
		if !ok || time.Until(deadline) > 3*time.Second {
			t.Fatalf("request lacks three-second deadline: %v", deadline)
		}
		response := reply(req, 200, string(gbkBytes(t, eduCNHTML)))
		response.Header.Set("Content-Type", "text/html")
		return response, nil
	})
	c.RunDig = func(context.Context, ...string) ([]byte, error) {
		t.Fatal("successful registry query must not use DNS")
		return nil, nil
	}
	r, err := c.Lookup(context.Background(), "NEU.EDU.CN.")
	if err != nil || r.Registrant != "Northeastern University" || calls != 1 {
		t.Fatalf("%+v, %v, calls %d", r, err, calls)
	}
	assertServiceMetadataOnly(t, c, nil)
}

func TestEDUCNAccessDenialsAndTimeoutsFallBackDirectly(t *testing.T) {
	for _, test := range []struct {
		name    string
		status  int
		body    string
		timeout bool
	}{
		{"denied", 403, "Forbidden", false},
		{"unauthorized", 401, "Unauthorized", false},
		{"registrar redirect", 302, "", false},
		{"server error", 503, "Unavailable", false},
		{"HTML denial", 200, "<html>Access denied</html>", false},
		{"WHOIS denial", 200, "<pre>Access denied</pre>", false},
		{"unsupported HTML encoding", 200, `<meta charset="shift_jis"><pre>Domain Name: neu.edu.cn</pre>`, false},
		{"wrong domain", 200, "<pre>Domain Name: other.edu.cn</pre>", false},
		{"timeout", 0, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			httpCalls, dnsCalls := 0, 0
			c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
				httpCalls++
				if req.URL.Host != "www.nic.edu.cn" {
					t.Fatalf("queried another host: %s", req.URL)
				}
				if test.timeout {
					return nil, context.DeadlineExceeded
				}
				response := reply(req, test.status, test.body)
				response.Header.Set("Location", "https://registrar.test/domain/neu.edu.cn")
				return response, nil
			})
			c.Warn = func(message string) { t.Errorf("unexpected fallback warning: %s", message) }
			lookupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			lookupDeadline, _ := lookupCtx.Deadline()
			c.RunDig = func(ctx context.Context, args ...string) ([]byte, error) {
				dnsCalls++
				if ctx.Err() != nil {
					t.Fatal("DNS inherited a canceled HTTP context")
				}
				if deadline, ok := ctx.Deadline(); !ok || !deadline.Equal(lookupDeadline) {
					t.Fatal("DNS received an extra timeout instead of the caller's context")
				}
				if strings.Join(args, " ") != "+noall +answer neu.edu.cn NS" {
					t.Fatalf("unexpected dig arguments: %v", args)
				}
				return []byte(eduCNDNSAnswer), nil
			}
			r, err := c.Lookup(lookupCtx, "neu.edu.cn")
			want := Record{Domain: "neu.edu.cn", Nameservers: []string{"ns.neu.edu.cn", "ns1.neu.edu.cn", "ns2.neu.edu.cn"}}
			if err != nil || !reflect.DeepEqual(r, want) || httpCalls != 1 || dnsCalls != 1 {
				t.Fatalf("got %+v, %v, HTTP %d, DNS %d", r, err, httpCalls, dnsCalls)
			}
			assertServiceMetadataOnly(t, c, nil)
		})
	}
}

func TestEDUCNDNSFailureDoesNotBecomeAvailability(t *testing.T) {
	for _, test := range []struct {
		name, output string
		err          error
	}{
		{"empty answer", "", nil},
		{"NXDOMAIN", ";; status: NXDOMAIN", nil},
		{"no NS answer", "neu.edu.cn. 60 IN A 202.118.1.35", nil},
		{"dig error", "", fmt.Errorf("dig failed")},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := fixtureClient(t, func(req *http.Request) (*http.Response, error) { return reply(req, 403, "Forbidden"), nil })
			c.RunDig = func(context.Context, ...string) ([]byte, error) { return []byte(test.output), test.err }
			r, err := c.Lookup(context.Background(), "neu.edu.cn")
			if e, ok := err.(*QueryError); !ok || e.Code != "dns-failed" || r.Result != "" {
				t.Fatalf("%+v, %v", r, err)
			}
		})
	}
}

func TestEDUCNExplicitNegativeResponse(t *testing.T) {
	r, err := parseEDUCN("neu.edu.cn", []byte("<pre>No matching record.</pre>"), "text/html")
	if err != nil || r.Result != NoRecord || r.Registrant != "" {
		t.Fatalf("%+v, %v", r, err)
	}
}

func TestDigNSIgnoresOtherRecordTypes(t *testing.T) {
	answer := eduCNDNSAnswer + `neu.edu.cn. 60 IN NS NS.NEU.EDU.CN.
neu.edu.cn. 60 IN A 202.118.1.35
neu.edu.cn. 60 IN AAAA 2001:da8:9000::35
neu.edu.cn. 60 IN CNAME unrelated.edu.cn.
neu.edu.cn. 60 IN NS 202.118.1.35
;; communications error
`
	want := []string{"ns.neu.edu.cn", "ns1.neu.edu.cn", "ns2.neu.edu.cn"}
	if got := parseDigNS(answer); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestEDUCNLikeSuffixStillUsesItsOwnRegistry(t *testing.T) {
	c := fixtureClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "registry.test" {
			t.Fatalf("query leaked to %s", req.URL)
		}
		return reply(req, 200, `{"objectClassName":"domain","ldhName":"neu.edu.cn.test"}`), nil
	})
	seedBootstrap(t, c, `[[["test"],["https://registry.test/"]]]`)
	c.RunDig = func(context.Context, ...string) ([]byte, error) {
		t.Fatal("unexpected dig invocation")
		return nil, nil
	}
	if _, err := c.Lookup(context.Background(), "neu.edu.cn.test"); err != nil {
		t.Fatal(err)
	}
}
