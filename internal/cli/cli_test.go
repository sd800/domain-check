package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/sd800/domain-check/internal/output"
)

func TestLanguageSelectors(t *testing.T) {
	for _, lang := range []string{"en", "zh"} {
		for _, prefix := range []string{"", "-", "--"} {
			for _, args := range [][]string{{prefix + lang, "example.com"}, {"example.com", prefix + lang}} {
				o, err := parseArgs(args, output.English)
				if err != nil || string(o.lang) != lang || o.domain != "example.com" {
					t.Errorf("%v: %+v, %v", args, o, err)
				}
			}
		}
	}
	o, err := parseArgs([]string{"en", "example.com", "--zh"}, output.English)
	if err != nil || o.lang != output.Chinese {
		t.Fatalf("%+v, %v", o, err)
	}
	for _, args := range [][]string{nil, {"zh"}, {"---zh", "example.com"}, {"one.com", "two.com"}, {"--server", "registrar.test"}} {
		if _, err := parseArgs(args, output.English); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}

func TestLocale(t *testing.T) {
	for locale, want := range map[string]output.Language{"zh_CN.UTF-8": output.Chinese, "zh-Hant-TW": output.Chinese, "en_US.UTF-8": output.English, "C": output.English, "fr_FR": output.English} {
		if got := languageFor(locale); got != want {
			t.Errorf("%s: %s", locale, got)
		}
	}
	if got := firstAppleLanguage("(\n    \"zh-Hans-CN\",\n    \"en-US\"\n)"); got != "zh-Hans-CN" {
		t.Fatal(got)
	}
}

func TestHelpAndVersionDoNotQuery(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	t.Setenv("DOMAIN_CACHE_DIR", t.TempDir())
	for _, args := range [][]string{{"--version"}, {"--help"}, {"zh", "--help"}} {
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), args, &out, &errOut, "26.10.21.1"); code != 0 || out.Len() == 0 || errOut.Len() != 0 {
			t.Fatalf("%v: %d, %s, %s", args, code, &out, &errOut)
		}
	}
}
