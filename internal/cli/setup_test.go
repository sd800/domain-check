package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sd800/domain-check/internal/localweb"
	"github.com/sd800/domain-check/internal/output"
)

type fakeWebSettings struct {
	state   localweb.Status
	fail    bool
	actions []string
}

func (f *fakeWebSettings) Status(context.Context) (localweb.Status, error) { return f.state, nil }
func (f *fakeWebSettings) SetEnabled(_ context.Context, on bool) error {
	if f.fail {
		return errors.New("test failure")
	}
	f.state.Enabled = on
	f.state.Running = on
	if !on {
		f.state.AtLogin = false
	}
	f.actions = append(f.actions, "web")
	return nil
}
func (f *fakeWebSettings) SetLogin(_ context.Context, on bool) error {
	f.state.AtLogin = on
	if on {
		f.state.Enabled = true
		f.state.Running = true
	}
	f.actions = append(f.actions, "login")
	return nil
}

func TestSetupSelectorsAreOfflineAndExclusive(t *testing.T) {
	for _, prefix := range []string{"", "-", "--"} {
		o, err := parseArgs([]string{"zh", prefix + "setup"}, output.English)
		if err != nil || !o.setup || o.intro || o.lang != output.Chinese {
			t.Fatal(o, err)
		}
		for _, extra := range []string{"example.com", "update"} {
			if _, err := parseArgs([]string{prefix + "setup", extra}, output.English); err == nil {
				t.Fatal("setup accepted conflicting command")
			}
		}
	}
}

func TestTwoStepSetupAndSavedStatus(t *testing.T) {
	for _, lang := range []output.Language{output.English, output.Chinese} {
		manager := &fakeWebSettings{}
		var out, errOut bytes.Buffer
		opened := 0
		code := setup(context.Background(), strings.NewReader("1\n3\n1\n2\n3\n"), &out, &errOut, lang, manager, func() { opened++ })
		if code != 0 || errOut.Len() != 0 || !manager.state.Enabled || manager.state.AtLogin || opened != 1 || len(manager.actions) != 3 {
			t.Fatal(code, manager, opened, errOut.String())
		}
		if !strings.Contains(out.String(), localweb.URL) || !strings.Contains(out.String(), output.Text(lang, "Setup complete.", "设置完成。")) {
			t.Fatal(out.String())
		}
	}
}

func TestSetupExitAndFailureDoNotReportSuccess(t *testing.T) {
	for _, input := range []string{"0\n", "", "7\n0\n"} {
		manager := &fakeWebSettings{}
		var out, errOut bytes.Buffer
		if code := setup(context.Background(), strings.NewReader(input), &out, &errOut, output.English, manager, nil); code != 0 || len(manager.actions) != 0 {
			t.Fatal(code, manager)
		}
	}
	manager := &fakeWebSettings{fail: true}
	var out, errOut bytes.Buffer
	if code := setup(context.Background(), strings.NewReader("1\n0\n"), &out, &errOut, output.English, manager, nil); code != 0 || !strings.Contains(errOut.String(), "test failure") || strings.Contains(out.String(), "Setting saved.") {
		t.Fatal(code, out.String(), errOut.String())
	}
}
