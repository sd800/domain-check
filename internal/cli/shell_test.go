package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestZshInteractivePasteAndWidgetPreservation(t *testing.T) {
	zsh := zshForTest(t)
	script, err := filepath.Abs("../../domain.zsh")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	rc := `PROMPT='DOMAIN_PROMPT> '
RPROMPT=''
bindkey -e
function domain { printf 'DOMAIN_ARG:%s\n' "$@"; print DOMAIN_DONE }
function domain-other { printf 'DOMAIN_ARG:%s\n' "$@"; print DOMAIN_DONE }
function original-paste { LBUFFER+='ORIGINAL:'; zle .bracketed-paste -- "$@" }
zle -N bracketed-paste original-paste
source "$DOMAIN_TEST_SCRIPT"
source "$DOMAIN_TEST_SCRIPT"
`
	if err := os.WriteFile(filepath.Join(dir, ".zshrc"), []byte(rc), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ZDOTDIR", dir)
	t.Setenv("DOMAIN_TEST_SCRIPT", script)
	t.Setenv("TERM", "xterm-256color")
	const driver = `emulate -L zsh
zmodload zsh/zpty || exit 77
zpty -b terminal zsh -di || exit 77
trap 'zpty -d terminal' EXIT
local data
for attempt in {1..100}; do
  if zpty -r terminal data '*DOMAIN_PROMPT> *'; then break; fi
  sleep .02
done
zpty -w -n terminal "$2"
zpty -w -n terminal $'\e[200~'"$1"$'\e[201~'
zpty -w -n terminal $'\r'
local all=''
for attempt in {1..100}; do
  if zpty -r terminal data; then
    all+=$data
    if [[ $all == *$'\r\nDOMAIN_DONE\r\n'* ]]; then
      print -rn -- "$all"
      exit 0
    fi
  fi
  sleep .02
done
print -rn -- "$all"
exit 1
`
	for _, test := range []struct{ prefix, pasted, want string }{
		{"domain ", "https://რეგისტრაცია.გე/?utm_source=example", "https://რეგისტრაცია.გე/?utm_source=example"},
		{"domain --zh ", "https://👩‍💻.ws/path?x=[a]*{one,two}(test)&y=2", "https://👩‍💻.ws/path?x=[a]*{one,two}(test)&y=2"},
		{"domain ", "https://example.com/?x=$(printf NOT_EXECUTED);`printf NOT_EXECUTED`", "https://example.com/?x=$(printf NOT_EXECUTED);`printf NOT_EXECUTED`"},
		{"domain-other ", "SAFE", "ORIGINAL:SAFE"},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		cmd := exec.CommandContext(ctx, zsh, "-dfc", driver, "paste-test", test.pasted, test.prefix)
		got, err := cmd.CombinedOutput()
		cancel()
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 77 {
			t.Skip("zsh pseudoterminal module is unavailable")
		}
		if err != nil || !bytes.Contains(got, []byte("\r\nDOMAIN_ARG:"+test.want+"\r\n")) {
			t.Fatalf("interactive paste %q: %q, %v", test.prefix, got, err)
		}
	}
}

func zshForTest(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh is unavailable")
	}
	return path
}

func TestZshDomainGlobsAndPastedURLMetacharacters(t *testing.T) {
	zsh := zshForTest(t)
	script, err := filepath.Abs("../../domain.zsh")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"https://რეგისტრაცია.გე/?utm_source=example",
		"https://😀.ws/path?name=one&other=two#fragment",
		"https://example.com/*?[abc]{one,two}(test)",
		"https://example.com/path?q='a'\"b\";|<>$HOME`pwd`$(pwd)",
		"https://user:password@example.com/path with spaces?a=1&b=2",
		"www.example.com/path?a=1&b=2",
		"  https://example.com/?a=1&b=2  ",
		"https://example.com/path\n;printf SHOULD_NOT_EXECUTE",
	} {
		for _, prefix := range []string{"domain ", "domain --zh ", "command domain -auto "} {
			cmd := exec.Command(zsh, "-dfc", `
function domain { printf '%s\0' "$@" }
source "$1"
LBUFFER=$2 RBUFFER=''
_domain_prepare_paste "$3" || exit 10
eval "domain $REPLY"
`, "shell-test", script, prefix, value)
			got, err := cmd.Output()
			if err != nil || string(got) != value+"\x00" {
				t.Fatalf("paste %q (%q): %q, %v", value, prefix, got, err)
			}
		}
	}
	cmd := exec.Command(zsh, "-dfc", `
function domain { printf '%s\0' "$@" }
source "$1"
eval 'domain https://example.com/path?x=*[]'
`, "shell-test", script)
	if got, err := cmd.Output(); err != nil || string(got) != "https://example.com/path?x=*[]\x00" {
		t.Fatalf("domain glob protection: %q, %v", got, err)
	}
	cmd = exec.Command(zsh, "-dfc", `
function domain { printf '%s\0' "$@" }
source "$1"
LBUFFER='domain ' RBUFFER=' --zh'
_domain_prepare_paste 'www.example.com/path?a=1&b=2' || exit 10
eval "domain $REPLY --zh"
`, "shell-test", script)
	if got, err := cmd.Output(); err != nil || string(got) != "www.example.com/path?a=1&b=2\x00--zh\x00" {
		t.Fatalf("paste before a trailing selector: %q, %v", got, err)
	}
	for _, prefix := range []string{"curl ", "echo domain ", "echo x; domain ", "domain update ", "domain '"} {
		cmd := exec.Command(zsh, "-dfc", `source "$1"; LBUFFER=$2 RBUFFER=''; _domain_prepare_paste 'https://example.com/?x=1&y=2'`, "shell-test", script, prefix)
		if err := cmd.Run(); err == nil {
			t.Fatalf("URL handler changed unrelated context: %q", prefix)
		}
	}
}

func TestInstallerPreservesConfigurationAndIsRepeatable(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	prefix := filepath.Join(dir, "prefix with ' apostrophe")
	zdot := filepath.Join(dir, "zsh")
	if err := os.Mkdir(zdot, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "dotfiles-zshrc")
	original := []byte("export EXISTING_SETTING=preserved\n")
	if err := os.WriteFile(target, original, 0640); err != nil {
		t.Fatal(err)
	}
	rc := filepath.Join(zdot, ".zshrc")
	if err := os.Symlink(target, rc); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PREFIX", prefix)
	t.Setenv("DESTDIR", "")
	t.Setenv("ZDOTDIR", zdot)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("DOMAIN_ZSH_INTEGRATION", "1")
	run := func(mode string) {
		t.Helper()
		cmd := exec.Command("sh", filepath.Join(root, "install.sh"), mode)
		if got, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("installer %s: %s, %v", mode, got, err)
		}
	}
	run("--shell-only")
	first, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	run("--shell-only")
	second, err := os.ReadFile(rc)
	if err != nil || !bytes.Equal(first, second) || !bytes.HasPrefix(second, original) || strings.Count(string(second), "# >>> Domain Check from Registries >>>") != 1 {
		t.Fatalf("installer duplicated or changed shell settings: %q, %v", second, err)
	}
	if link, err := os.Readlink(rc); err != nil || link != target {
		t.Fatalf("installer replaced existing symlink: %q, %v", link, err)
	}
	if info, err := os.Stat(rc); err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("installer changed dotfile permissions: %v, %v", info, err)
	}
	zsh := zshForTest(t)
	cmd := exec.Command(zsh, "-dfc", `source "$1"; [[ $EXISTING_SETTING == preserved && $aliases[domain] == 'noglob domain' ]]`, "shell-test", rc)
	if got, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("quoted installation prefix failed: %s, %v", got, err)
	}
	run("--uninstall")
	if got, err := os.ReadFile(rc); err != nil || !bytes.Equal(got, original) {
		t.Fatalf("uninstall changed unrelated settings: %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(prefix, "share/domain-check/domain.zsh")); !os.IsNotExist(err) {
		t.Fatalf("uninstall retained shell integration: %v", err)
	}
}

func TestStagedInstallDoesNotTouchUserZsh(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Setenv("DESTDIR", filepath.Join(dir, "stage"))
	t.Setenv("PREFIX", "/opt/domain-check")
	t.Setenv("ZDOTDIR", filepath.Join(dir, "user-zsh"))
	t.Setenv("DOMAIN_ZSH_INTEGRATION", "1")
	cmd := exec.Command("sh", filepath.Join(root, "install.sh"), "--shell-only")
	if got, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("staged shell installation: %s, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "user-zsh")); !os.IsNotExist(err) {
		t.Fatalf("staged install touched user settings: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "stage/opt/domain-check/share/domain-check/domain.zsh")); err != nil {
		t.Fatal(err)
	}
}

func TestInstallerRejectsMalformedManagedBlocksWithoutChangingZshrc(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{
		"before\n# >>> Domain Check from Registries >>>\nafter\n",
		"before\n# <<< Domain Check from Registries <<<\nafter\n",
		"# >>> Domain Check from Registries >>>\n# <<< Domain Check from Registries <<<\n# >>> Domain Check from Registries >>>\n# <<< Domain Check from Registries <<<\n",
	} {
		dir := t.TempDir()
		rc := filepath.Join(dir, ".zshrc")
		if err := os.WriteFile(rc, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PREFIX", filepath.Join(dir, "prefix"))
		t.Setenv("DESTDIR", "")
		t.Setenv("ZDOTDIR", dir)
		t.Setenv("DOMAIN_ZSH_INTEGRATION", "1")
		cmd := exec.Command("sh", filepath.Join(root, "install.sh"), "--shell-only")
		if err := cmd.Run(); err == nil {
			t.Fatal("installer accepted a malformed managed block")
		}
		if got, err := os.ReadFile(rc); err != nil || string(got) != content {
			t.Fatalf("installer damaged shell settings: %q, %v", got, err)
		}
	}
}
