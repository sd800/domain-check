package localweb

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeManager(t *testing.T) (*Manager, *bool, *[]string) {
	t.Helper()
	root := t.TempDir()
	running := false
	commands := []string{}
	m := &Manager{ConfigDir: filepath.Join(root, "configuration"), Home: root, Executable: filepath.Join(root, "folder & name", "domain"), CacheDir: filepath.Join(root, "cache"), Platform: "darwin"}
	m.Probe = func(context.Context) (string, bool) { return "test", running }
	m.Available = func() error { return nil }
	m.Command = func(_ context.Context, args ...string) error {
		commands = append(commands, strings.Join(args, " "))
		switch args[0] {
		case "print":
			if !running {
				return errors.New("not loaded")
			}
		case "bootout":
			running = false
		case "bootstrap":
			running = true
		}
		return nil
	}
	return m, &running, &commands
}

func TestWebServiceLifecycleAndNoLoginByDefault(t *testing.T) {
	m, running, commands := fakeManager(t)
	ctx := context.Background()
	initial, err := m.Status(ctx)
	if err != nil || initial.Enabled || initial.AtLogin || initial.Running {
		t.Fatal(initial, err)
	}
	if err = m.SetEnabled(ctx, true); err != nil {
		t.Fatal(err)
	}
	status, err := m.Status(ctx)
	if err != nil || !status.Enabled || !status.Running || status.AtLogin {
		t.Fatal(status, err)
	}
	if _, err = os.Stat(m.agentPath(true)); !os.IsNotExist(err) {
		t.Fatal("created login item without selection")
	}
	if err = m.SetLogin(ctx, true); err != nil {
		t.Fatal(err)
	}
	status, _ = m.Status(ctx)
	if !status.AtLogin || !status.Running {
		t.Fatal(status)
	}
	data, _, err := readTestFile(m.agentPath(true))
	if err != nil {
		t.Fatal(err)
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		_, err = decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(string(data), "&amp;") || !strings.Contains(string(data), "--web-serve") || strings.Contains(string(data), "sudo") {
		t.Fatal("invalid launch registration")
	}
	if err = m.SetLogin(ctx, false); err != nil {
		t.Fatal(err)
	}
	status, _ = m.Status(ctx)
	if status.AtLogin || !status.Enabled || !status.Running {
		t.Fatal(status)
	}
	if _, err = os.Stat(m.agentPath(true)); !os.IsNotExist(err) {
		t.Fatal("left login item")
	}
	if err = m.SetEnabled(ctx, false); err != nil {
		t.Fatal(err)
	}
	status, _ = m.Status(ctx)
	if status.Enabled || status.AtLogin || *running {
		t.Fatal(status)
	}
	if len(*commands) == 0 {
		t.Fatal("no lifecycle calls")
	}
	entries, err := os.ReadDir(m.ConfigDir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "web.json" {
		t.Fatal("unexpected persisted data", entries, err)
	}
}

func readTestFile(path string) ([]byte, os.FileInfo, error) {
	data, err := os.ReadFile(path)
	info, statErr := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	return data, info, statErr
}

func TestLoginEnablesWebAndFailedStartPreservesSettings(t *testing.T) {
	m, _, _ := fakeManager(t)
	ctx := context.Background()
	if err := m.SetLogin(ctx, true); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(m.settingsPath())
	m.Command = func(_ context.Context, args ...string) error {
		if args[0] == "bootstrap" {
			return errors.New("test launch failure")
		}
		return nil
	}
	if err := m.SetLogin(ctx, false); err == nil {
		t.Fatal("failed launch reported success")
	}
	after, _ := os.ReadFile(m.settingsPath())
	if string(before) != string(after) {
		t.Fatal("failure overwrote saved settings")
	}
}

func TestDisabledRefreshHasNoSideEffects(t *testing.T) {
	m, _, commands := fakeManager(t)
	if err := m.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(*commands) != 0 {
		t.Fatal("disabled refresh invoked launchctl")
	}
	if _, err := os.Stat(m.ConfigDir); !os.IsNotExist(err) {
		t.Fatal("disabled refresh created state")
	}
}

func TestPortConflictPreservesSettingsAndRestoresPreviousJob(t *testing.T) {
	m, running, _ := fakeManager(t)
	ctx := context.Background()
	if err := m.SetEnabled(ctx, true); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(m.settingsPath())
	m.Available = func() error { return errors.New("port conflict") }
	if err := m.SetLogin(ctx, true); err == nil {
		t.Fatal("port conflict reported success")
	}
	after, _ := os.ReadFile(m.settingsPath())
	if string(before) != string(after) || !*running {
		t.Fatal("previous settings or job were not restored")
	}
	if _, err := os.Stat(m.agentPath(true)); !os.IsNotExist(err) {
		t.Fatal("failed transition created a login registration")
	}
}
