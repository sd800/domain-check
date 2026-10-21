package registry

import (
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const benchmarkWHOISDomain = `Domain Name: example.test
Creation Date: 2000-10-21T01:02:03Z
Registry Expiry Date: 2030-10-21T01:02:03Z
Updated Date: 2026-10-20T12:34:56Z
Registrar: Example Registrar
Domain Status: ok
Registrant Name: Example Owner
Registrant Organization: Example Organization
Registrant Email: owner@example.test
Name Server: ns1.example.test
Name Server: ns2.example.test
DNSSEC: unsigned
`

func BenchmarkParseWHOIS(b *testing.B) {
	for _, test := range []struct{ name, source string }{
		{"flat", benchmarkWHOISDomain},
		{"objects", "domain: example.test\nregistrant: OWNER1\nnsset: NS1\nkeyset: KEY1\n\nnic-hdl: OWNER1\norg: Example Organization\ne-mail: owner@example.test\n\nnsset: NS1\nnserver: ns1.example.test\n\nkeyset: KEY1\ndnskey: 257 3 13 ABCDEF0123456789\n"},
		{"footer", benchmarkWHOISDomain + ">>> Last update of WHOIS database: 2026-10-21T00:00:00Z <<<\n" + strings.Repeat("These records are provided for lawful query-based purposes.\n", 64)},
		{"negative", "% Registry WHOIS service\nDomain Name: example.test\nStatus: AVAILABLE\n"},
	} {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := parseWHOIS("example.test", test.source); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParseEDUCN(b *testing.B) {
	for _, test := range []struct{ name, source string }{
		{"registry", eduCNHTML},
		{"surroundingHTML", "<div>" + strings.Repeat("<span>Unrelated content</span>", 512) + "</div>" + eduCNHTML},
	} {
		b.Run(test.name, func(b *testing.B) {
			data, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(test.source))
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := parseEDUCN("neu.edu.cn", data, "text/html"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParseEDUCAUSE(b *testing.B) {

	b.ReportAllocs()
	for b.Loop() {
		if _, err := parseWHOIS("example.edu", eduCAUSERecord); err != nil {
			b.Fatal(err)
		}
	}
}
