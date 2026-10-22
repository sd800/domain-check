package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sd800/domain-check/internal/localweb"
	"github.com/sd800/domain-check/internal/output"
	"github.com/sd800/domain-check/internal/registry"
	"golang.org/x/term"
)

type webSettings interface {
	Status(context.Context) (localweb.Status, error)
	SetEnabled(context.Context, bool) error
	SetLogin(context.Context, bool) error
}

func setup(ctx context.Context, input io.Reader, out, errOut io.Writer, lang output.Language, manager webSettings, openBrowser func()) int {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 256), 4096)
	step := 1
	for {
		if err := ctx.Err(); err != nil {
			return 3
		}
		status, err := manager.Status(ctx)
		if err != nil {
			fmt.Fprintln(errOut, "domain: "+registry.Clean(err.Error()))
			return 3
		}
		state := output.Text(lang, "Disabled", "已关闭")
		if step == 1 && status.Enabled {
			state = output.Text(lang, "Enabled", "已开启")
			if !status.Running {
				state += output.Text(lang, " (not running)", "（未运行）")
			}
		} else if step == 2 && status.AtLogin {
			state = output.Text(lang, "Enabled", "已开启")
		}
		if step == 1 {
			fmt.Fprintln(out, output.Text(lang, "\nWeb interface", "\n网页版查询"))
			fmt.Fprintln(out, localweb.URL)
			fmt.Fprintln(out, output.Text(lang, "Current status: ", "当前状态：")+state)
			setupChoices(out, output.Text(lang, "1 Enable\n2 Disable\n3 Next\n0 Exit", "1 开启\n2 关闭\n3 下一步\n0 退出菜单"))
		} else {
			fmt.Fprintln(out, output.Text(lang, "\nStart automatically when you log in to macOS", "\n登录 macOS 后自动启动"))
			fmt.Fprintln(out, output.Text(lang, "Current status: ", "当前状态：")+state)
			if !status.Enabled {
				fmt.Fprintln(out, output.Text(lang, "Enabling this also turns on the web interface.", "开启此选项也会开启网页版查询。"))
			}
			setupChoices(out, output.Text(lang, "1 Enable\n2 Disable\n3 Finish", "1 开启\n2 关闭\n3 完成"))
		}
		fmt.Fprint(out, "> ")
		scanned := make(chan bool, 1)
		go func() { scanned <- scanner.Scan() }()
		var hasChoice bool
		select {
		case <-ctx.Done():
			return 0
		case hasChoice = <-scanned:
		}
		if !hasChoice {
			if scanner.Err() != nil {
				fmt.Fprintln(errOut, output.Text(lang, "Cannot read menu choice.", "无法读取菜单选项。"))
				return 3
			}
			return 0
		}
		choice := strings.TrimSpace(scanner.Text())
		if step == 1 && choice == "0" {
			return 0
		}
		if choice == "3" {
			if step == 2 {
				fmt.Fprintln(out, output.Text(lang, "Setup complete.", "设置完成。"))
				return 0
			}
			step = 2
			continue
		}
		if choice != "1" && choice != "2" {
			fmt.Fprintln(out, output.Text(lang, "Choose one of the listed numbers.", "请输入菜单中的数字。"))
			continue
		}
		on := choice == "1"
		if step == 1 {
			err = manager.SetEnabled(ctx, on)
		} else {
			err = manager.SetLogin(ctx, on)
		}
		if err != nil {
			fmt.Fprintln(errOut, "domain: "+registry.Clean(err.Error()))
			continue
		}
		fmt.Fprintln(out, output.Text(lang, "Setting saved.", "设置已保存。"))
		if step == 1 && on && openBrowser != nil {
			openBrowser()
		}
	}
}

func setupChoices(out io.Writer, text string) {
	if file, ok := out.(*os.File); ok && term.IsTerminal(int(file.Fd())) && os.Getenv("TERM") != "dumb" && os.Getenv("NO_COLOR") == "" {
		text = "\x1b[36m" + text + "\x1b[0m"
	}
	fmt.Fprintln(out, text)
}

func systemBrowser() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "/usr/bin/open", localweb.URL).Run()
}
