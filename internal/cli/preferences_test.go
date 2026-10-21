package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sd800/domain-check/internal/output"
	"github.com/sd800/domain-check/internal/registry"
)

func TestLanguagePersistsAcrossInvocations(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	t.Setenv("DOMAIN_CONFIG_DIR", t.TempDir())
	t.Setenv("DOMAIN_CACHE_DIR", t.TempDir())
	run := func(args []string, want string) {
		t.Helper()
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), args, &out, &errOut, "test"); code != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), want) {
			t.Fatalf("%v: exit %d, stdout %q, stderr %q", args, code, out.String(), errOut.String())
		}
	}
	run(nil, "Usage:")
	for _, selector := range []string{"zh", "-zh", "--zh"} {
		run([]string{selector}, "语言已切换为简体中文，并保存为默认语言。")
		run(nil, "用法:")
		run([]string{"--help"}, "用法:")
		run([]string{"en"}, "Language switched to English and saved as the default.")
		run(nil, "Usage:")
	}
	run([]string{"en", "--zh"}, "语言已切换为简体中文，并保存为默认语言。")
	run(nil, "用法:")
	// An invalid invocation must not overwrite the saved language.
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"-en", "bad/domain"}, &out, &errOut, "test"); code != 2 {
		t.Fatalf("invalid invocation returned %d", code)
	}
	run(nil, "用法:")
	for _, selector := range []string{"auto", "-auto", "--auto"} {
		run([]string{selector}, "Language set to follow the device language. Preference saved.")
		run(nil, "Usage:")
		t.Setenv("LC_ALL", "zh_CN.UTF-8")
		run(nil, "用法:")
		t.Setenv("LC_ALL", "en_US.UTF-8")
	}
}

func TestInvalidPreferencesFallBackAndCanBeReplaced(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	dir := t.TempDir()
	t.Setenv("DOMAIN_CONFIG_DIR", dir)
	for _, data := range []string{`not JSON`, `{"language":"invalid"}`} {
		if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), nil, &out, &errOut, "test"); code != 0 || !strings.Contains(out.String(), "Usage:") || !strings.Contains(errOut.String(), "Cannot read the saved language") {
			t.Fatalf("corrupt preference: exit %d, stdout %q, stderr %q", code, out.String(), errOut.String())
		}
		out.Reset()
		errOut.Reset()
		if code := Run(context.Background(), []string{"zh"}, &out, &errOut, "test"); code != 0 || errOut.Len() != 0 {
			t.Fatalf("cannot replace corrupt preference: %d, %s", code, errOut.String())
		}
	}
}

func TestPreferenceWriteFailureIsReported(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOMAIN_CONFIG_DIR", path)
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"zh"}, &out, &errOut, "test"); code != 3 || out.Len() != 0 || !strings.Contains(errOut.String(), "无法保存语言偏好") {
		t.Fatalf("write failure: exit %d, stdout %q, stderr %q", code, out.String(), errOut.String())
	}
}

func TestStandaloneLanguageConfirmationIsSavedAndOffline(t *testing.T) {
	for _, locale := range []string{"en_US.UTF-8", "zh_CN.UTF-8"} {
		for _, selector := range []string{"en", "zh", "auto"} {
			for _, prefix := range []string{"", "-", "--"} {
				t.Run(locale+"/"+prefix+selector, func(t *testing.T) {
					t.Setenv("LC_ALL", locale)
					t.Setenv("DOMAIN_CONFIG_DIR", t.TempDir())
					cache := t.TempDir()
					t.Setenv("DOMAIN_CACHE_DIR", cache)
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					var out, errOut bytes.Buffer
					code := Run(ctx, []string{prefix + selector}, &out, &errOut, "test")
					want := "Language switched to English and saved as the default.\n"
					if selector == "zh" {
						want = "语言已切换为简体中文，并保存为默认语言。\n"
					} else if selector == "auto" {
						want = "Language set to follow the device language. Preference saved.\n"
						if strings.HasPrefix(locale, "zh") {
							want = "语言已切换为跟随设备语言，并保存设置。\n"
						}
					}
					if code != 0 || errOut.Len() != 0 || out.String() != want {
						t.Fatalf("exit=%d, stdout=%q, stderr=%q; want %q", code, out.String(), errOut.String(), want)
					}
					if saved, err := loadLanguagePreference(); err != nil || string(saved) != selector {
						t.Fatalf("confirmation without saved preference: %s, %v", saved, err)
					}
					if entries, err := os.ReadDir(cache); err != nil || len(entries) != 0 {
						t.Fatalf("language command touched IANA cache: %v, %v", entries, err)
					}
				})
			}
		}
	}
}

func TestLanguageWithQueryPreservesOutputAndContinuesOnSaveFailure(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	for _, saveFails := range []bool{false, true} {
		t.Run(fmt.Sprint(saveFails), func(t *testing.T) {
			config := t.TempDir()
			if saveFails {
				config = filepath.Join(config, "file")
				if err := os.WriteFile(config, []byte("not a directory"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("DOMAIN_CONFIG_DIR", config)
			c := registry.NewClient(t.TempDir())
			c.HTTP.Transport = updateTransport(func(req *http.Request) (*http.Response, error) {
				var data string
				switch req.URL.String() {
				case "https://data.iana.org/rdap/dns.json":
					data = `{"services":[[["com"],["https://registry.test/"]]]}`
				case "https://registry.test/domain/example.com":
					data = `{"objectClassName":"domain","ldhName":"example.com"}`
				default:
					t.Fatalf("unexpected network request: %s", req.URL)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(data)), Request: req}, nil
			})
			var out, errOut bytes.Buffer
			code := run(context.Background(), []string{"en", "example.com"}, &out, &errOut, "test", c)
			if code != 0 || out.String() != "Domain Name:         example.com\n" {
				t.Fatalf("lookup output changed: exit=%d, stdout=%q, stderr=%q", code, out.String(), errOut.String())
			}
			if saveFails != strings.Contains(errOut.String(), "Cannot save the language preference") {
				t.Fatalf("unexpected preference warning: %s", errOut.String())
			}
			if !saveFails {
				if saved, err := loadLanguagePreference(); err != nil || saved != output.English {
					t.Fatalf("query did not save language: %s, %v", saved, err)
				}
			}
		})
	}
}
