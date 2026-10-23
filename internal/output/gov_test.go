package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rivo/uniseg"
	"github.com/sd800/domain-check/internal/registry"
)

func TestGovRegistrantLocationLayout(t *testing.T) {
	r := registry.Record{Domain: "example.gov", Registrant: "Example Federal Office", RegistrantCity: "Washington", RegistrantState: "DC", RegistrantCountry: "United States"}
	var b bytes.Buffer
	if err := Format(&b, r, English, 80); err != nil {
		t.Fatal(err)
	}
	want := "Domain Name:         example.gov\n\nRegistrant:          Example Federal Office\nCity:                Washington\nState:               DC\nCountry:             United States\n"
	if b.String() != want || strings.Contains(b.String(), "Address:") || strings.Contains(b.String(), "Postal Code:") {
		t.Fatal(b.String())
	}
	r.RegistrantAddress, r.RegistrantPostalCode = []string{"One Federal Street with a long continuation", "Building 2"}, "12345"
	for _, lang := range []Language{English, Chinese} {
		b.Reset()
		if err := Format(&b, r, lang, 43); err != nil {
			t.Fatal(err)
		}
		column := 21
		if lang == Chinese {
			column = 14
			for _, label := range []string{"地址:", "城市:", "州/地区:", "邮编:", "国家:"} {
				if !strings.Contains(b.String(), label) {
					t.Fatal("missing Chinese address label", label)
				}
			}
		}
		for _, line := range strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n") {
			if line == "" {
				continue
			}
			if uniseg.StringWidth(line) > 43 {
				t.Fatal("address overflow", line)
			}
			if label, value, ok := strings.Cut(line, ":"); ok {
				padding := len(value) - len(strings.TrimLeft(value, " "))
				if uniseg.StringWidth(label)+1+padding != column {
					t.Fatal("address column moved", line)
				}
			} else if !strings.HasPrefix(line, strings.Repeat(" ", column)) {
				t.Fatal("address continuation moved", line)
			}
		}
	}
}
