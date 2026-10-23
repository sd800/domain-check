package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sd800/domain-check/internal/registry"
)

func TestEmptyRegistrantRemainsAboveDisclosedContactFields(t *testing.T) {
	for _, tc := range []struct {
		name   string
		record registry.Record
	}{
		{"detailed city", registry.Record{RegistrantContact: &registry.RegistrantContact{City: "Public City"}}},
		{"redacted name", registry.Record{RegistrantContact: &registry.RegistrantContact{Name: "REDACTED FOR PRIVACY", Organization: "REDACTED", City: "Public City"}}},
		{"opaque address", registry.Record{RegistrantContact: &registry.RegistrantContact{Address: []string{"One Street", "Building 2"}}}},
		{"phone", registry.Record{RegistrantContact: &registry.RegistrantContact{Phones: []string{"+1.5550100"}}}},
		{"email", registry.Record{RegistrantEmail: []string{"owner@example.test"}}},
		{"dedicated location", registry.Record{RegistrantCity: "Public City"}},
		{"allocation address", registry.Record{TLD: true, RegistrantAddress: []string{"One Street"}}},
	} {
		for _, lang := range []Language{English, Chinese} {
			t.Run(tc.name+"/"+string(lang), func(t *testing.T) {
				r := tc.record
				r.Domain = "example.test"
				var b bytes.Buffer
				if err := Format(&b, r, lang, 80); err != nil {
					t.Fatal(err)
				}
				label := Text(lang, "Registrant:", "注册人:")
				lines := strings.Split(b.String(), "\n")
				if len(lines) < 5 || strings.TrimSpace(lines[2]) != label || strings.TrimSpace(lines[3]) == "" || strings.Contains(b.String(), "REDACTED") {
					t.Fatal("missing empty registrant header above disclosed details:", b.String())
				}
			})
		}
	}
}

func TestEmptyRegistrantGroupStaysHidden(t *testing.T) {
	for _, r := range []registry.Record{
		{Domain: "example.test", Nameservers: []string{"ns1.example.test"}},
		{Domain: "example.test", RegistrantContact: &registry.RegistrantContact{Name: "REDACTED FOR PRIVACY"}},
		{Domain: "zz", TLD: true, Admin: "Example Admin", AdminEmails: []string{"admin@example.test"}},
		{Domain: "example.test", Result: registry.NotRegistered, RegistrantCity: "must not display"},
	} {
		for _, lang := range []Language{English, Chinese} {
			var b bytes.Buffer
			if err := Format(&b, r, lang, 80); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(b.String(), Text(lang, "Registrant:", "注册人:")) {
				t.Fatal("empty or unrelated group acquired a registrant header:", b.String())
			}
		}
	}
}
