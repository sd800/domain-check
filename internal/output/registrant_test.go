package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rivo/uniseg"
	"github.com/sd800/domain-check/internal/registry"
)

func TestDetailedRegistrantLabelsOrderAndAlignment(t *testing.T) {
	c := &registry.RegistrantContact{Name: "Alex Example", Organization: "Example City Authority", Street: []string{"400 Example Street"}, City: "Example City", StateProvince: "IL", PostalCode: "60540", Country: "US", Phones: []string{"+1.5550100"}, Emails: []string{"owner@example.test"}}
	r := registry.Record{Domain: "example.us", Registrant: "legacy organization must not replace name", RegistrantContact: c, RegistrantEmail: []string{"wrong@example.test"}}
	var b bytes.Buffer
	if err := Format(&b, r, English, 80); err != nil {
		t.Fatal(err)
	}
	want := "Domain Name:         example.us\n\nRegistrant:          Alex Example\nOrganization:        Example City Authority\nStreet:              400 Example Street\nCity:                Example City\nState/Province:      IL\nPostal Code:         60540\nCountry:             US\nPhone:               +1.5550100\nEmail:               owner@example.test\n"
	if b.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", b.String(), want)
	}
	b.Reset()
	if err := Format(&b, r, Chinese, 80); err != nil {
		t.Fatal(err)
	}
	last := -1
	for _, label := range []string{"注册人:", "所属组织:", "地址:", "城市:", "省/州:", "邮编:", "国家/地区:", "电话:", "邮箱:"} {
		index := strings.Index(b.String(), label)
		if index <= last {
			t.Fatal("missing or reordered label", label, b.String())
		}
		last = index
	}
	for _, lang := range []Language{English, Chinese} {
		b.Reset()
		c.Street = []string{strings.Repeat("long street ", 8), "Building 2"}
		if err := Format(&b, r, lang, 43); err != nil {
			t.Fatal(err)
		}
		column := 21
		if lang == Chinese {
			column = 14
		}
		for _, line := range strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n") {
			if line == "" {
				continue
			}
			if uniseg.StringWidth(line) > 43 {
				t.Fatal("overflow", line)
			}
			if label, value, ok := strings.Cut(line, ":"); ok {
				padding := len(value) - len(strings.TrimLeft(value, " "))
				if uniseg.StringWidth(label)+1+padding != column {
					t.Fatal("column moved", line)
				}
			} else if !strings.HasPrefix(line, strings.Repeat(" ", column)) {
				t.Fatal("continuation moved", line)
			}
		}
	}
}

func TestOpaqueRegistrantAddressAndOrganizationOnly(t *testing.T) {
	r := registry.Record{Domain: "example.us", Registrant: "legacy value", RegistrantContact: &registry.RegistrantContact{Organization: "Example Organization", Address: []string{"One Street", "City: Keep as supplied"}}}
	var b bytes.Buffer
	if err := Format(&b, r, English, 80); err != nil {
		t.Fatal(err)
	}
	want := "Domain Name:         example.us\n\nRegistrant:          Example Organization\nAddress:             One Street\n                     City: Keep as supplied\n"
	if b.String() != want {
		t.Fatal(b.String())
	}
}

func TestRegistrantNameAndOrganizationPresence(t *testing.T) {
	for _, tc := range []struct {
		name, organization string
		separate           bool
	}{
		{"", "Example Organization", false},
		{"REDACTED FOR PRIVACY", "Example Organization", false},
		{"Alex Example", "", false},
		{"Alex Example", "Example Organization", true},
		{"Example Organization", "Example Organization", true},
	} {
		for _, lang := range []Language{English, Chinese} {
			var b bytes.Buffer
			r := registry.Record{Domain: "example.us", RegistrantContact: &registry.RegistrantContact{Name: tc.name, Organization: tc.organization, City: "Example City"}}
			if err := Format(&b, r, lang, 80); err != nil {
				t.Fatal(err)
			}
			registrantLabel, organizationLabel := "Registrant:", "Organization:"
			if lang == Chinese {
				registrantLabel, organizationLabel = "注册人:", "所属组织:"
			}
			if !strings.Contains(b.String(), registrantLabel) || strings.Contains(b.String(), organizationLabel) != tc.separate {
				t.Fatalf("name %q, organization %q: %s", tc.name, tc.organization, b.String())
			}
		}
	}
}
