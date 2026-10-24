package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sd800/domain-check/internal/localweb"
	"github.com/sd800/domain-check/internal/output"
)

func TestUninstallSelectorsAndConflicts(t *testing.T) {
	for _, prefix := range []string{"", "-", "--"} {
		o, err := parseArgs([]string{"zh", prefix + "uninstall"}, output.English)
		if err != nil || !o.uninstall || o.intro || o.domain != "" || o.lang != output.Chinese {
			t.Fatal(o, err)
		}
	}
	for _, args := range [][]string{{"uninstall", "example.com"}, {"--uninstall", "setup"}, {"update", "-uninstall"}, {"---uninstall"}} {
		if _, err := parseArgs(args, output.English); err == nil {
			t.Fatal("accepted conflicting uninstall", args)
		}
	}
	o, err := parseArgs([]string{".uninstall"}, output.English)
	if err != nil || o.uninstall || o.domain != "uninstall" {
		t.Fatal(o, err)
	}
}

func TestUninstallConfirmation(t *testing.T) {
	for _, lang := range []output.Language{output.English, output.Chinese} {
		for _, answer := range []string{"\n", "n\n", "no\n", "", "yes", "maybe\n", "y please\n", "yes\x00\n", "y\n", " YES \r\n", "是\n", "确认\n"} {
			var out, errOut bytes.Buffer
			called := false
			code := uninstall(context.Background(), strings.NewReader(answer), &out, &errOut, lang, func(context.Context) error {
				called = true
				return nil
			})
			want := answer == "y\n" || answer == " YES \r\n" || answer == "是\n" || answer == "确认\n"
			if code != 0 || errOut.Len() != 0 || called != want || !strings.Contains(out.String(), "[y/N]") {
				t.Fatalf("%s %q: exit=%d removed=%v output=%q errors=%q", lang, answer, code, called, out.String(), errOut.String())
			}
		}
	}
}

func TestUninstallCommandCancellationKeepsTemporaryLanguageAndData(t *testing.T) {
	config, cache := t.TempDir(), t.TempDir()
	t.Setenv("DOMAIN_CONFIG_DIR", config)
	t.Setenv("DOMAIN_CACHE_DIR", cache)
	if err := saveLanguagePreference(output.Chinese); err != nil {
		t.Fatal(err)
	}
	cacheFile := filepath.Join(cache, "dns.json")
	if err := os.WriteFile(cacheFile, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, selector := range []string{"uninstall", "-uninstall", "--uninstall"} {
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.WriteString("n\n"); err != nil {
			t.Fatal(err)
		}
		writer.Close()
		original := os.Stdin
		os.Stdin = reader
		var out, errOut bytes.Buffer
		code := Run(context.Background(), []string{"en", selector}, &out, &errOut, "test")
		os.Stdin = original
		reader.Close()
		if code != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), "Uninstall cancelled.") {
			t.Fatal(code, out.String(), errOut.String())
		}
		if language, err := loadLanguagePreference(); err != nil || language != output.Chinese {
			t.Fatal("saved temporary uninstall language", language, err)
		}
		if data, err := os.ReadFile(cacheFile); err != nil || string(data) != "keep" {
			t.Fatal("cancelled command changed cache", err)
		}
	}
}

func TestUninstallCancellationAndFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	input, writer := io.Pipe()
	defer input.Close()
	defer writer.Close()
	var out, errOut bytes.Buffer
	if code := uninstall(ctx, input, &out, &errOut, output.English, func(context.Context) error {
		t.Fatal("removed after interrupt")
		return nil
	}); code != 0 {
		t.Fatal(code)
	}
	for _, answer := range []string{strings.Repeat("y", 5000) + "\n", "y\n"} {
		out.Reset()
		errOut.Reset()
		if code := uninstall(context.Background(), strings.NewReader(answer), &out, &errOut, output.English, func(context.Context) error {
			return os.ErrPermission
		}); code != 3 || strings.Contains(out.String(), "uninstalled.") {
			t.Fatal("failed uninstall claimed success", code, out.String(), errOut.String())
		}
	}
}

func uninstallManager(t *testing.T) (*localweb.Manager, *[]string) {
	t.Helper()
	dir := t.TempDir()
	m := &localweb.Manager{Home: dir, ConfigDir: filepath.Join(dir, "configuration"), CacheDir: filepath.Join(dir, "cache"), Executable: filepath.Join(dir, "bin", "domain-check"), Platform: "darwin"}
	for _, path := range []string{m.Executable, filepath.Join(m.ConfigDir, "config.json"), filepath.Join(m.ConfigDir, "web.json"), filepath.Join(m.ConfigDir, "web-agent.plist"), filepath.Join(m.CacheDir, "dns.json"), filepath.Join(m.CacheDir, "tlds.json"), filepath.Join(dir, "Library", "LaunchAgents", "localhost.domain-check.web.plist")} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("synthetic"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("domain-check", filepath.Join(filepath.Dir(m.Executable), "domain")); err != nil {
		t.Fatal(err)
	}
	calls := []string{}
	m.Command = func(_ context.Context, args ...string) error {
		calls = append(calls, strings.Join(args, " "))
		return nil
	}
	return m, &calls
}

func TestRemoveInstallationIsScopedAndStopsBeforeDeleting(t *testing.T) {
	m, calls := uninstallManager(t)
	keep := []string{filepath.Join(m.ConfigDir, "keep.txt"), filepath.Join(m.CacheDir, "keep.txt"), filepath.Join(m.Home, "source.go"), filepath.Join(m.Home, ".zshrc")}
	for _, path := range keep {
		if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	baseCommand := m.Command
	m.Command = func(ctx context.Context, args ...string) error {
		if _, err := os.Stat(m.Executable); err != nil {
			t.Fatal("executable removed before service stop", err)
		}
		return baseCommand(ctx, args...)
	}
	if err := removeInstallation(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 2 || !strings.HasPrefix((*calls)[0], "print gui/") || !strings.HasPrefix((*calls)[1], "bootout gui/") {
		t.Fatal("unexpected commands", *calls)
	}
	for _, path := range []string{m.Executable, filepath.Join(filepath.Dir(m.Executable), "domain"), filepath.Join(m.ConfigDir, "config.json"), filepath.Join(m.ConfigDir, "web.json"), filepath.Join(m.ConfigDir, "web-agent.plist"), filepath.Join(m.CacheDir, "dns.json"), filepath.Join(m.CacheDir, "tlds.json"), filepath.Join(m.Home, "Library", "LaunchAgents", "localhost.domain-check.web.plist")} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatal("left installation file", path, err)
		}
	}
	for _, path := range keep {
		if data, err := os.ReadFile(path); err != nil || string(data) != "keep" {
			t.Fatal("changed unrelated file", path, err)
		}
	}
}

func TestRemoveInstallationPreservesExecutableOnFailures(t *testing.T) {
	for _, failure := range []string{"foreign-link", "regular-command", "service-error", "cache-directory", "replaced-executable"} {
		t.Run(failure, func(t *testing.T) {
			m, calls := uninstallManager(t)
			command := filepath.Join(filepath.Dir(m.Executable), "domain")
			switch failure {
			case "foreign-link", "regular-command":
				if err := os.Remove(command); err != nil {
					t.Fatal(err)
				}
				if failure == "foreign-link" {
					if err := os.Symlink("another-program", command); err != nil {
						t.Fatal(err)
					}
				} else if err := os.WriteFile(command, []byte("keep"), 0700); err != nil {
					t.Fatal(err)
				}
			case "service-error":
				m.Command = func(context.Context, ...string) error { return os.ErrPermission }
			case "cache-directory":
				path := filepath.Join(m.CacheDir, "dns.json")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "replaced-executable":
				m.Command = func(_ context.Context, args ...string) error {
					if args[0] == "bootout" {
						pending := filepath.Join(filepath.Dir(m.Executable), "pending")
						if err := os.WriteFile(pending, []byte("replacement"), 0700); err != nil {
							return err
						}
						return os.Rename(pending, m.Executable)
					}
					return nil
				}
			}
			if err := removeInstallation(context.Background(), m); err == nil {
				t.Fatal("ignored unsafe uninstall")
			}
			if _, err := os.Stat(m.Executable); err != nil {
				t.Fatal("removed executable after failure", err)
			}
			if _, err := os.Lstat(command); err != nil {
				t.Fatal("removed command after failure", err)
			}
			if (failure == "foreign-link" || failure == "regular-command") && len(*calls) != 0 {
				t.Fatal("changed service before path validation", *calls)
			}
		})
	}
}

func TestUninstallContextCancellationPreservesAllFiles(t *testing.T) {
	m, calls := uninstallManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := removeInstallation(ctx, m); !errors.Is(err, context.Canceled) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
	if _, err := os.Stat(m.Executable); err != nil {
		t.Fatal(err)
	}
}
