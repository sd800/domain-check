package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		run([]string{selector}, "用法:")
		run(nil, "用法:")
		run([]string{"--help"}, "用法:")
		run([]string{"en"}, "Usage:")
		run(nil, "Usage:")
	}
	run([]string{"en", "--zh"}, "用法:")
	run(nil, "用法:")
	// An invalid invocation must not overwrite the saved language.
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"-en", "bad/domain"}, &out, &errOut, "test"); code != 2 {
		t.Fatalf("invalid invocation returned %d", code)
	}
	run(nil, "用法:")
	for _, selector := range []string{"auto", "-auto", "--auto"} {
		run([]string{selector}, "Usage:")
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
	if code := Run(context.Background(), []string{"zh"}, &out, &errOut, "test"); code != 0 || !strings.Contains(out.String(), "用法:") || !strings.Contains(errOut.String(), "无法保存语言偏好") {
		t.Fatalf("write failure: exit %d, stdout %q, stderr %q", code, out.String(), errOut.String())
	}
}
