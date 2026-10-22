package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rivo/uniseg"
	"github.com/sd800/domain-check/internal/registry"
)

func TestEnglishLayout(t *testing.T) {
	r := registry.Record{Domain: "example.com", Registration: "2003-03-10 11:19:18", Expiration: "2027-05-15 11:25:25", Statuses: []string{"ok"}, Registrar: "Example Registrar", Nameservers: []string{"ns1.example.com", "ns2.example.com"}, DNSSEC: "unsigned"}
	var b bytes.Buffer
	if err := Format(&b, r, English, 80); err != nil {
		t.Fatal(err)
	}
	want := "Domain Name:         example.com\n\nRegistration Time:   2003-03-10 11:19:18\nExpiration Time:     2027-05-15 11:25:25\n\nRegistrar:           Example Registrar\nDomain Status:       ok\n\nName Server:         ns1.example.com\nName Server:         ns2.example.com\nDNSSEC:              unsigned\n"
	if b.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", b.String(), want)
	}
	if strings.Contains(b.String(), "Registrant") || strings.Contains(b.String(), "Last Changed") {
		t.Fatal("printed missing fields")
	}
}

func TestChineseAndContinuationColumns(t *testing.T) {
	r := registry.Record{Domain: "shanghai.cn", Registrant: strings.Repeat("上海溪诚信息科技有限公司", 4), RegistrantEmail: []string{"admin@" + strings.Repeat("long", 10) + ".com"}, Nameservers: []string{"dns1.example.cn"}, DNSSEC: "unsigned"}
	for _, lang := range []Language{English, Chinese} {
		var b bytes.Buffer
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
				t.Errorf("line too wide: %q", line)
			}
			if label, _, ok := strings.Cut(line, ":"); ok {
				rest := line[len(label)+1:]
				padding := len(rest) - len(strings.TrimLeft(rest, " "))
				if uniseg.StringWidth(label)+1+padding != column {
					t.Errorf("misaligned %q", line)
				}
			} else if !strings.HasPrefix(line, strings.Repeat(" ", column)) {
				t.Errorf("misaligned continuation: %q", line)
			}
		}
	}
}

func TestNegativeLayout(t *testing.T) {
	for _, lang := range []Language{English, Chinese} {
		var b bytes.Buffer
		Format(&b, registry.Record{Domain: "example.com", Result: registry.NoRecord, Registrant: "must not display"}, lang, 80)
		if strings.Count(b.String(), "\n") != 2 || strings.Contains(b.String(), "must not display") {
			t.Fatal(b.String())
		}
	}
}

func TestUndisclosedNamespaceResultDoesNotImplyAvailability(t *testing.T) {
	for lang, want := range map[Language]string{English: "Domain-specific registration data not provided", Chinese: "注册局未提供此域名的独立注册信息"} {
		var b bytes.Buffer
		if err := Format(&b, registry.Record{Domain: "example.edu.tw", Result: registry.NotProvided}, lang, 100); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(b.String(), want) || strings.Count(b.String(), "\n") != 2 {
			t.Fatal(b.String())
		}
	}
}

func TestTLDLayoutAndAddressContinuation(t *testing.T) {
	r := registry.Record{Domain: "zz", TLD: true, Registration: "1994-04-08 04:00:00 UTC", Changed: "2026-10-22", Expiration: "must omit", Registrar: "must omit", Statuses: []string{"inactive"}, Registrant: "Example Registry", RegistrantAddress: []string{"One Registry Road", "Example City 12345", "Example Country"}, RegistrationURL: "http://registry.example.test/", Admin: "Alex Example", AdminOrganization: "Example Organization", AdminRole: "Operations Lead", AdminPhones: []string{"+1-555-0100", "+1-555-0101"}, AdminEmails: []string{"admin@example.test"}, Nameservers: []string{"ns1.example.test"}, DNSSEC: "signed"}
	var b bytes.Buffer
	if err := Format(&b, r, English, 100); err != nil {
		t.Fatal(err)
	}
	want := "Domain Name:         zz\n\nRegistration Time:   1994-04-08 04:00:00 UTC\nLast Changed Time:   2026-10-22\nDomain Status:       inactive\n\nRegistrant:          Example Registry\nAddress:             One Registry Road\n                     Example City 12345\n                     Example Country\nRegistration URL:    http://registry.example.test/\n\nAdmin:               Alex Example\nOrganization:        Example Organization\nRole:                Operations Lead\nPhone:               +1-555-0100\nPhone:               +1-555-0101\nEmail:               admin@example.test\n\nName Server:         ns1.example.test\nDNSSEC:              signed\n"
	if b.String() != want {
		t.Fatalf("TLD layout:\n%s\nwant:\n%s", b.String(), want)
	}
	for _, lang := range []Language{English, Chinese} {
		b.Reset()
		if err := Format(&b, r, lang, 43); err != nil {
			t.Fatal(err)
		}
		column := 21
		if lang == Chinese {
			column = 14
			if !strings.Contains(b.String(), "1994 年 4 月 8 日") {
				t.Fatal(b.String())
			}
		}
		for _, line := range strings.Split(b.String(), "\n") {
			if uniseg.StringWidth(line) > 43 {
				t.Fatalf("TLD row overflow: %q", line)
			}
			if strings.HasPrefix(line, " ") && !strings.HasPrefix(line, strings.Repeat(" ", column)) {
				t.Fatalf("TLD continuation misaligned: %q", line)
			}
		}
	}
}

func TestTLDNoRecordDoesNotClaimAvailability(t *testing.T) {
	for lang, want := range map[Language]string{English: "No IANA TLD record", Chinese: "IANA 无此顶级域名记录"} {
		var b bytes.Buffer
		if err := Format(&b, registry.Record{Domain: "example-unregistered", TLD: true, Result: registry.NoRecord}, lang, 120); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(b.String(), want) || strings.Count(b.String(), "\n") != 2 {
			t.Fatal(b.String())
		}
	}
}

func TestEDURegistrantLinesShareOneAlignedLabel(t *testing.T) {
	r := registry.Record{Domain: "example.edu", Registration: "1985-07-15", Registrant: "Example University", RegistrantDetails: []string{"Example Information Technologies", "Network Operations Center", "136 College Road", "Example City, EX 12345", "USA"}, Nameservers: []string{"ns1.example.test"}}
	for _, lang := range []Language{English, Chinese} {
		var b bytes.Buffer
		if err := Format(&b, r, lang, 80); err != nil {
			t.Fatal(err)
		}
		column, label := 21, "Registrant:"
		if lang == Chinese {
			column, label = 14, "注册人:"
		}
		if strings.Count(b.String(), label) != 1 || strings.Contains(b.String(), "Address:") || strings.Contains(b.String(), "地址:") {
			t.Fatal(b.String())
		}
		for _, line := range r.RegistrantDetails {
			if !strings.Contains(b.String(), "\n"+strings.Repeat(" ", column)+line+"\n") {
				t.Fatalf("misaligned registrant line: %q\n%s", line, b.String())
			}
		}
		b.Reset()
		if err := Format(&b, r, lang, 43); err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(b.String(), "\n") {
			if uniseg.StringWidth(line) > 43 || strings.HasPrefix(line, " ") && !strings.HasPrefix(line, strings.Repeat(" ", column)) {
				t.Fatalf("invalid wrapped line: %q", line)
			}
		}
	}
}

func TestRestrictedRegistrationResult(t *testing.T) {
	for lang, want := range map[Language]string{
		English: "Registration restricted (online registration unavailable)",
		Chinese: "注册受限（无法在线注册）",
	} {
		var b bytes.Buffer
		if err := Format(&b, registry.Record{Domain: "net.cn", Result: registry.Restricted}, lang, 80); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(b.String(), want) || strings.Count(b.String(), "\n") != 2 {
			t.Fatalf("%s: %s", lang, b.String())
		}
	}
}

func TestTerminalControlsAndGraphemes(t *testing.T) {
	var b bytes.Buffer
	Format(&b, registry.Record{Domain: "example.com", Registrant: "abc\x1b]2;evil\a\nnext"}, English, 80)
	if strings.ContainsAny(b.String(), "\x1b\a") {
		t.Fatal("terminal control escaped sanitation")
	}
	for _, line := range wrap("👩‍💻👩‍💻👩‍💻ééé", 4) {
		if uniseg.StringWidth(line) > 4 || strings.HasPrefix(line, "\u200d") {
			t.Fatalf("split grapheme: %q", line)
		}
	}
}

func TestUnavailableRegistrationResult(t *testing.T) {
	for lang, want := range map[Language]string{English: "Not available for registration", Chinese: "不可注册"} {
		var b bytes.Buffer
		if err := Format(&b, registry.Record{Domain: "example.test", Result: registry.Unavailable}, lang, 80); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(b.String(), want) || strings.Count(b.String(), "\n") != 2 {
			t.Fatalf("%s: %s", lang, b.String())
		}
	}
}
