package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rivo/uniseg"
	"github.com/sd800/domain-check/internal/registry"
)

func TestIDNRowFollowsDomainInBothLanguages(t *testing.T) {
	for _, name := range []string{"示例.中国", "bücher.example", "テスト.example", "예시.example", "www.示例.中国", "😀.ws", "👩‍💻.ws", "👩🏽‍💻.ws"} {
		domain, err := registry.NormalizeDomain(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, lang := range []Language{English, Chinese} {
			for _, result := range []registry.Result{"", registry.NoRecord, registry.Reserved} {
				r := registry.Record{Domain: domain, Result: result, Registration: "2000-10-21"}
				var b bytes.Buffer
				if err := Format(&b, r, lang, 120); err != nil {
					t.Fatal(err)
				}
				want := "Domain Name:         " + domain + "\nIDN:                 " + name + "\n"
				if lang == Chinese {
					want = "域名:         " + domain + "\n国际化域名:   " + name + "\n"
				}
				if !strings.HasPrefix(b.String(), want) {
					t.Fatalf("%s (%s):\n%s\nwant prefix:\n%s", name, lang, b.String(), want)
				}
				if result != "" && (strings.Count(b.String(), "\n") != 3 || strings.Contains(b.String(), "2000-10-21")) {
					t.Fatalf("negative result layout: %s", b.String())
				}
			}
		}
	}
}

func TestEmojiIDNWrapPreservesJoinedClusters(t *testing.T) {
	name := strings.Repeat("👩‍💻", 6) + ".ws"
	domain, err := registry.NormalizeDomain(name)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := Format(&b, registry.Record{Domain: domain, Registrant: "Unsafe\x1b\u202e name"}, English, 27); err != nil {
		t.Fatal(err)
	}
	if strings.Count(b.String(), "👩‍💻") != 6 || strings.ContainsAny(b.String(), "\x1b\u202e") {
		t.Fatalf("emoji cluster or terminal sanitization changed: %q", b.String())
	}
	for _, line := range strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n") {
		if uniseg.StringWidth(line) > 27 || strings.HasSuffix(line, "\u200d") {
			t.Fatalf("emoji wrapped incorrectly: %q", line)
		}
	}
}

func TestIDNRowOmittedForPlainOrInvalidNames(t *testing.T) {
	for _, domain := range []string{"example.com", "example.xn--not-an-idn-", "xn--.example", "xn--a.example", "axn--b.example", "xn--bcher-kva.example/path"} {
		var b bytes.Buffer
		if err := Format(&b, registry.Record{Domain: domain}, English, 120); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(b.String(), "IDN:") {
			t.Fatalf("unexpected decoded name for %s: %s", domain, b.String())
		}
	}
}

func TestIDNContinuationAlignment(t *testing.T) {
	name := strings.Repeat("示例", 10) + ".中国"
	domain, err := registry.NormalizeDomain(name)
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range []Language{English, Chinese} {
		var b bytes.Buffer
		if err := Format(&b, registry.Record{Domain: domain}, lang, 36); err != nil {
			t.Fatal(err)
		}
		column, label := 21, "IDN"
		if lang == Chinese {
			column, label = 14, "国际化域名"
		}
		inside := false
		var decoded strings.Builder
		for _, line := range strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n") {
			if uniseg.StringWidth(line) > 36 {
				t.Fatalf("line exceeds terminal width: %q", line)
			}
			if strings.HasPrefix(line, label+":") {
				inside = true
				decoded.WriteString(strings.TrimLeft(strings.TrimPrefix(line, label+":"), " "))
			} else if inside {
				if !strings.HasPrefix(line, strings.Repeat(" ", column)) {
					t.Fatalf("misaligned IDN continuation: %q", line)
				}
				decoded.WriteString(strings.TrimLeft(line, " "))
			}
		}
		if decoded.String() != name {
			t.Fatalf("wrapped Unicode name changed: %q; want %q", decoded.String(), name)
		}
	}
}
