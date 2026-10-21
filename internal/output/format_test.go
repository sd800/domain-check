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
