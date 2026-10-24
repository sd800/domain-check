package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sd800/domain-check/internal/localweb"
	"github.com/sd800/domain-check/internal/output"
	"github.com/sd800/domain-check/internal/registry"
	"github.com/sd800/domain-check/internal/safeio"
)

func uninstall(ctx context.Context, input io.Reader, out, errOut io.Writer, lang output.Language, remove func(context.Context) error) int {
	fail := func(err error) int {
		fmt.Fprintln(errOut, "domain: "+output.Text(lang, "Uninstall failed", "卸载失败")+": "+registry.Clean(err.Error()))
		return 3
	}
	if _, err := fmt.Fprintln(out, output.Text(lang,
		"Uninstall Domain Check from Registries?\nThis removes this installation, its command link, web service, login item, settings, and IANA cache files.\nSource files, development samples, and shell PATH settings are kept.",
		"确认卸载 Domain Check from Registries？\n将移除此安装、命令链接、网页版服务、自启动项、设置和 IANA 缓存文件。\n保留源码、开发样本和 shell 的 PATH 配置。")); err != nil {
		return fail(err)
	}
	if _, err := fmt.Fprint(out, output.Text(lang, "Continue? [y/N] ", "继续卸载？[y/N] ")); err != nil {
		return fail(err)
	}
	type reply struct {
		line string
		err  error
	}
	answer := make(chan reply, 1)
	go func() {
		reader := bufio.NewReaderSize(input, 4096)
		line, err := reader.ReadSlice('\n')
		answer <- reply{string(line), err}
	}()
	var response reply
	select {
	case <-ctx.Done():
		fmt.Fprintln(out, output.Text(lang, "\nUninstall cancelled.", "\n已取消卸载。"))
		return 0
	case response = <-answer:
	}
	// A complete line is required: EOF and truncated or oversized input never confirm.
	if response.err != nil && response.err != io.EOF {
		return fail(fmt.Errorf("cannot read confirmation: %w", response.err))
	}
	choice := strings.ToLower(strings.TrimSpace(response.line))
	if response.err != nil || (choice != "y" && choice != "yes" && choice != "是" && choice != "确认") || ctx.Err() != nil {
		fmt.Fprintln(out, output.Text(lang, "Uninstall cancelled.", "已取消卸载。"))
		return 0
	}
	if err := remove(ctx); err != nil {
		return fail(err)
	}
	fmt.Fprintln(out, output.Text(lang, "Domain Check from Registries uninstalled.", "Domain Check from Registries 已卸载。"))
	return 0
}

func removeInstallation(ctx context.Context, manager *localweb.Manager) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if filepath.Base(manager.Executable) != "domain-check" {
		return fmt.Errorf("run uninstall using the installed domain command")
	}
	dir := filepath.Dir(manager.Executable)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	binary, err := root.Lstat("domain-check")
	if err != nil {
		return err
	}
	if !binary.Mode().IsRegular() {
		return fmt.Errorf("the installed executable is not a regular file")
	}
	command, err := root.Lstat("domain")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if command != nil {
		if command.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("domain is not this installation's command link; it was preserved")
		}
		target, err := root.Readlink("domain")
		if err != nil {
			return err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(dir, target)
		}
		if filepath.Clean(target) != filepath.Clean(manager.Executable) {
			return fmt.Errorf("domain points to another program; it was preserved")
		}
	}
	if err := manager.Remove(ctx); err != nil {
		return err
	}
	if err := safeio.RemoveFiles(manager.ConfigDir, "config.json"); err != nil {
		return err
	}
	if err := safeio.RemoveFiles(manager.CacheDir, "dns.json", "tlds.json"); err != nil {
		return err
	}
	// Recheck file identities after service and data cleanup, before unlinking.
	current, err := root.Lstat("domain-check")
	if err != nil || !os.SameFile(binary, current) {
		return fmt.Errorf("the executable changed during uninstall; it was preserved")
	}
	if command != nil {
		current, err := root.Lstat("domain")
		if err != nil || !os.SameFile(command, current) {
			return fmt.Errorf("the domain command changed during uninstall; executable preserved")
		}
		if err := root.Remove("domain"); err != nil {
			return err
		}
	}
	return root.Remove("domain-check")
}
