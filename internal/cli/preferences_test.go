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
	for _, selector := range []string{"auto-lang", "-auto-lang", "--auto-lang"} {
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
		for _, selector := range []string{"en", "zh", "auto-lang"} {
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
					} else if selector == "auto-lang" {
						want = "Language set to follow the device language. Preference saved.\n"
						if strings.HasPrefix(locale, "zh") {
							want = "语言已切换为跟随设备语言，并保存设置。\n"
						}
					}
					if code != 0 || errOut.Len() != 0 || out.String() != want {
						t.Fatalf("exit=%d, stdout=%q, stderr=%q; want %q", code, out.String(), errOut.String(), want)
					}
					if saved, err := loadLanguagePreference(); err != nil || saved != map[string]output.Language{"en": output.English, "zh": output.Chinese, "auto-lang": automaticLanguage}[selector] {
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

func temporaryLanguageQueryClient(t *testing.T) *registry.Client {
	t.Helper()
	c := registry.NewClient(t.TempDir())
	c.HTTP.Transport = updateTransport(func(req *http.Request) (*http.Response, error) {
		var data string
		switch req.URL.String() {
		case "https://data.iana.org/rdap/dns.json":
			data = `{"services":[[["com"],["https://registry.test/"]]]}`
		case "https://registry.test/domain/example.com":
			data = `{"objectClassName":"domain","ldhName":"example.com"}`
		default:
			t.Errorf("unexpected network request: %s", req.URL)
			return nil, fmt.Errorf("unexpected network request")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(data)), Request: req}, nil
	})
	return c
}

func TestQueryLanguageIsTemporary(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	t.Setenv("COLUMNS", "80")
	for _, selector := range []string{"en", "zh", "auto-lang"} {
		for _, prefix := range []string{"", "-", "--"} {
			for _, input := range []string{"example.com", "https://www.example.com/path?item=1"} {
				for _, after := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s%s/%s/after=%t", prefix, selector, input, after), func(t *testing.T) {
						config := t.TempDir()
						t.Setenv("DOMAIN_CONFIG_DIR", config)
						initial, want := output.Chinese, "Domain Name:         example.com\n"
						if selector == "zh" {
							initial, want = output.English, "域名:         example.com\n"
						}
						if err := saveLanguagePreference(initial); err != nil {
							t.Fatal(err)
						}
						path := filepath.Join(config, "config.json")
						before, err := os.ReadFile(path)
						if err != nil {
							t.Fatal(err)
						}
						args := []string{prefix + selector, input}
						if after {
							args[0], args[1] = args[1], args[0]
						}
						var out, errOut bytes.Buffer
						code := run(context.Background(), args, &out, &errOut, "test", temporaryLanguageQueryClient(t))
						if code != 0 || errOut.Len() != 0 || out.String() != want {
							t.Fatalf("exit=%d, stdout=%q, stderr=%q; want %q", code, out.String(), errOut.String(), want)
						}
						saved, err := os.ReadFile(path)
						if err != nil || !bytes.Equal(before, saved) {
							t.Fatalf("query changed saved language: %q, %v", saved, err)
						}
						out.Reset()
						if code := Run(context.Background(), nil, &out, &errOut, "test"); code != 0 || errOut.Len() != 0 {
							t.Fatalf("subsequent command failed: %d, %s", code, errOut.String())
						}
						usage := "用法:"
						if initial == output.English {
							usage = "Usage:"
						}
						if !strings.Contains(out.String(), usage) {
							t.Fatalf("subsequent command lost saved language: %s", out.String())
						}
					})
				}
			}
		}
	}
}

func TestTemporaryLanguageDoesNotCreateOrWritePreferences(t *testing.T) {
	t.Setenv("LC_ALL", "zh_CN.UTF-8")
	for _, blocked := range []bool{false, true} {
		for _, selector := range []string{"en", "zh", "auto-lang"} {
			t.Run(fmt.Sprintf("%s/blocked=%t", selector, blocked), func(t *testing.T) {
				config := filepath.Join(t.TempDir(), "preferences")
				if blocked {
					if err := os.WriteFile(config, []byte("not a directory"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				t.Setenv("DOMAIN_CONFIG_DIR", config)
				var out, errOut bytes.Buffer
				code := run(context.Background(), []string{selector, "example.com"}, &out, &errOut, "test", temporaryLanguageQueryClient(t))
				label := "域名:"
				if selector == "en" {
					label = "Domain Name:"
				}
				if code != 0 || errOut.Len() != 0 || !strings.HasPrefix(out.String(), label) {
					t.Fatalf("temporary selection accessed preferences: %d, %q, %q", code, out.String(), errOut.String())
				}
				if blocked {
					data, err := os.ReadFile(config)
					if err != nil || string(data) != "not a directory" {
						t.Fatalf("configuration changed: %q, %v", data, err)
					}
				} else if _, err := os.Stat(config); !os.IsNotExist(err) {
					t.Fatalf("query created a preference directory: %v", err)
				}
			})
		}
	}
}

func TestAutoTLDIsAQueryAndSavedAutoPreferenceRemainsCompatible(t *testing.T) {
	t.Setenv("DOMAIN_CONFIG_DIR", t.TempDir())
	t.Setenv("LC_ALL", "en_US.UTF-8")
	if err := saveLanguagePreference(automaticLanguage); err != nil {
		t.Fatal(err)
	}
	c := registry.NewClient(t.TempDir())
	c.HTTP.Transport = updateTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://rdap.iana.org/domain/auto" {
			t.Fatal(req.URL)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"objectClassName":"domain","ldhName":"auto","status":["active"]}`)), Request: req}, nil
	})
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"auto"}, &out, &errOut, "test", c); code != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), "Domain Name:") || !strings.Contains(out.String(), "active") || strings.Contains(out.String(), "Preference saved") {
		t.Fatal(code, out.String(), errOut.String())
	}
	if saved, err := loadLanguagePreference(); err != nil || saved != automaticLanguage {
		t.Fatal("existing automatic preference changed", saved, err)
	}
}
