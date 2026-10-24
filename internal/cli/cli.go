package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sd800/domain-check/internal/localweb"
	"github.com/sd800/domain-check/internal/output"
	"github.com/sd800/domain-check/internal/registry"
	"golang.org/x/term"
)

type options struct {
	domain     string
	lang       output.Language
	preference output.Language
	intro      bool
	help       bool
	version    bool
	update     bool
	setup      bool
	uninstall  bool
}

func parseArgs(args []string, lang output.Language) (options, error) {
	o := options{lang: lang}
	for _, arg := range args {
		switch strings.ToLower(arg) {
		case "en", "-en", "--en":
			o.lang = output.English
			o.preference = output.English
		case "zh", "-zh", "--zh":
			o.lang = output.Chinese
			o.preference = output.Chinese
		case "auto-lang", "-auto-lang", "--auto-lang":
			o.lang = lang
			o.preference = automaticLanguage
		case "help", "-help", "--help", "-h":
			o.help = true
		case "version", "-version", "--version", "-v":
			o.version = true
		case "update", "-update", "--update":
			o.update = true
		case "setup", "-setup", "--setup":
			o.setup = true
		case "uninstall", "-uninstall", "--uninstall":
			o.uninstall = true
		default:
			if strings.HasPrefix(arg, "-") || o.domain != "" {
				return o, fmt.Errorf("unexpected argument; expected one domain or URL")
			}
			o.domain = arg
		}
	}
	if o.help || o.version {
		return o, nil
	}
	if o.uninstall {
		if o.domain != "" || o.update || o.setup {
			return o, fmt.Errorf("uninstall cannot be combined with a domain, URL, update, or setup")
		}
		return o, nil
	}
	if o.setup {
		if o.domain != "" || o.update {
			return o, fmt.Errorf("setup cannot be combined with a domain, URL, or update")
		}
		return o, nil
	}
	if o.update {
		if o.domain != "" {
			return o, fmt.Errorf("update cannot be combined with a domain or URL")
		}
		return o, nil
	}
	if o.domain == "" {
		o.intro = true
		return o, nil
	}
	domain, err := registry.NormalizeInput(o.domain)
	if err != nil {
		return o, err
	}
	o.domain = domain
	return o, nil
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	return run(ctx, args, stdout, stderr, version, registry.NewClient(registry.DefaultCacheDir()))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, version string, client *registry.Client) int {
	if len(args) == 1 && args[0] == "--web-serve" {
		if err := localweb.Run(ctx, version); err != nil {
			fmt.Fprintln(stderr, "domain: "+registry.Clean(err.Error()))
			return 3
		}
		return 0
	}
	if len(args) == 1 && args[0] == "--web-refresh" {
		manager, err := localweb.NewManager(version)
		if err == nil {
			err = manager.Refresh(ctx)
		}
		if err != nil {
			fmt.Fprintln(stderr, "domain: "+registry.Clean(err.Error()))
			return 3
		}
		return 0
	}
	o, err := parseArgs(args, preferredLanguage())
	if o.preference == "" {
		if saved, loadErr := loadLanguagePreference(); loadErr != nil {
			fmt.Fprintln(stderr, "domain: "+output.Text(o.lang, "Cannot read the saved language; using the device language", "无法读取已保存的语言，使用设备语言")+": "+registry.Clean(loadErr.Error()))
		} else if saved == output.English || saved == output.Chinese {
			o.lang = saved
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, output.Text(o.lang, "Usage: domain [en|zh|auto-lang] [domain|URL|update|setup|uninstall]", "用法: domain [en|zh|auto-lang] [域名|URL|update|setup|uninstall]"))
		fmt.Fprintln(stderr, "domain: "+output.Text(o.lang, "Invalid arguments, domain, or URL", "参数、域名或 URL 无效")+": "+registry.Clean(err.Error()))
		return 2
	}
	if o.help {
		fmt.Fprint(stdout, help(o.lang))
		return 0
	}
	if o.version {
		fmt.Fprintln(stdout, "Domain Check from Registries "+version)
		return 0
	}
	if o.uninstall {
		return uninstall(ctx, os.Stdin, stdout, stderr, o.lang, func(ctx context.Context) error {
			manager, err := localweb.NewManager(version)
			if err != nil {
				return err
			}
			return removeInstallation(ctx, manager)
		})
	}
	if o.setup {
		manager, err := localweb.NewManager(version)
		if err != nil {
			fmt.Fprintln(stderr, "domain: "+registry.Clean(err.Error()))
			return 3
		}
		return setup(ctx, os.Stdin, stdout, stderr, o.lang, manager, systemBrowser)
	}
	if o.intro {
		var message string
		if o.preference != "" {
			if err := saveLanguagePreference(o.preference); err != nil {
				fmt.Fprintln(stderr, "domain: "+output.Text(o.lang, "Cannot save the language preference", "无法保存语言偏好")+": "+registry.Clean(err.Error()))
				return 3
			}
			message = languageConfirmation(o.preference, o.lang) + "\n"
		} else {
			message = introduction(o.lang, version)
		}
		if _, err := fmt.Fprint(stdout, message); err != nil {
			fmt.Fprintln(stderr, "domain: "+registry.Clean(err.Error()))
			return 3
		}
		return 0
	}
	client.Warn = func(code string) {
		message := output.Text(o.lang, "Cannot save the local cache; continuing online.", "无法保存本地缓存，继续使用在线数据。")
		if code == "stale-bootstrap" || code == "stale-tlds" {
			message = output.Text(o.lang, "Cannot refresh IANA data; using a previously validated cache.", "无法更新 IANA 数据，使用已验证的旧缓存。")
		}
		fmt.Fprintln(stderr, "domain: "+message)
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if o.update {
		if err := client.UpdateIANA(ctx); err != nil {
			fmt.Fprintln(stderr, "domain: "+errorText(o.lang, err))
			return 3
		}
		if _, err := fmt.Fprintln(stdout, output.Text(o.lang, "IANA RDAP and WHOIS routing data updated.", "IANA RDAP 和 WHOIS 路由数据已更新。")); err != nil {
			fmt.Fprintln(stderr, "domain: "+registry.Clean(err.Error()))
			return 3
		}
		return 0
	}
	record, err := client.Lookup(ctx, o.domain)
	if err != nil {
		fmt.Fprintln(stderr, "domain: "+errorText(o.lang, err))
		return 3
	}
	if err := output.Format(stdout, record, o.lang, terminalWidth(stdout)); err != nil {
		fmt.Fprintln(stderr, "domain: "+registry.Clean(err.Error()))
		return 3
	}
	if record.Result != "" {
		return 1
	}
	return 0
}

func terminalWidth(w io.Writer) int {
	if file, ok := w.(*os.File); ok {
		if width, _, err := term.GetSize(int(file.Fd())); err == nil && width > 0 {
			return width
		}
	}
	if width, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && width > 0 {
		return width
	}
	return 80
}

func errorText(lang output.Language, err error) string {
	var query *registry.QueryError
	if !errors.As(err, &query) {
		return output.Text(lang, "Query failed", "查询失败") + ": " + registry.Clean(err.Error())
	}
	messages := map[string][2]string{
		"edu-cn-failed":       {"Domain query failed", "域名查询失败"},
		"dns-failed":          {"DNS nameserver query failed", "DNS 域名服务器查询失败"},
		"bootstrap-failed":    {"Cannot load IANA RDAP bootstrap data", "无法读取 IANA RDAP 引导数据"},
		"iana-update-failed":  {"Cannot update IANA routing data", "无法更新 IANA 路由数据"},
		"iana-cache-failed":   {"Cannot save IANA routing data", "无法保存 IANA 路由数据缓存"},
		"tld-data-failed":     {"Cannot load IANA TLD registry data", "无法读取 IANA 顶级域名注册表"},
		"unsupported-suffix":  {"Unknown or unsupported domain suffix", "未知或不支持的域名后缀"},
		"no-registry-service": {"No public registry RDAP or WHOIS service", "注册局没有公开的 RDAP 或 WHOIS 服务"},
		"rdap-failed":         {"Registry RDAP query failed", "注册局 RDAP 查询失败"},
		"whois-failed":        {"Registry WHOIS query failed", "注册局 WHOIS 查询失败"},
		"fallback-failed":     {"Registry RDAP and WHOIS queries failed", "注册局 RDAP 和 WHOIS 查询均失败"},
		"invalid-response":    {"Invalid registry response", "注册局响应无效"},
		"redirect-blocked":    {"Redirect blocked: queries stay within the selected registry endpoints", "已阻止重定向：仅查询选定的注册局端点"},
		"rate-limited":        {"Registry rate limit reached; try again later", "已触发注册局查询频率限制，请稍后重试"},
		"access-denied":       {"Registry denied access", "注册局拒绝访问"},
		"invalid-domain":      {"Invalid domain", "域名无效"},
	}
	message, ok := messages[query.Code]
	if !ok {
		return registry.Clean(query.Error())
	}
	return output.Text(lang, message[0], message[1]) + ": " + registry.Clean(query.Detail)
}

func introduction(lang output.Language, version string) string {
	return "Domain Check from Registries\nversion " + version + "\n© 2026 Songming.org\n\n" + output.Text(lang, `Query domain registration information directly from registries.
Registrar referrals are never followed.

Usage: domain [en|zh|auto-lang] [domain|URL|update|setup|uninstall]

  domain example.com          Use the default language
  domain .com                 Query a TLD at IANA
  domain .github.io           A leading dot is ignored
  domain -en example.com      English for this query
  domain zh example.com       Simplified Chinese for this query
  domain auto-lang            Follow the device language
  domain update               Refresh IANA RDAP and WHOIS data now
  domain setup                Set up the local web interface
  domain uninstall            Uninstall after confirmation

Language selectors apply only to the current query.
Run domain en, domain zh, or domain auto-lang on its own to save the default.
Run domain --help for more options.
`, `直接向注册局查询域名注册信息，始终不跟随注册商。

用法: domain [en|zh|auto-lang] [域名|URL|update|setup|uninstall]

  domain example.com          使用默认语言
  domain .com                 向 IANA 查询顶级域名
  domain .github.io           忽略域名开头的一个点
  domain -en example.com      本次查询使用英文
  domain zh example.com       本次查询使用简体中文
  domain auto-lang            跟随设备语言
  domain update               立即更新 IANA RDAP 和 WHOIS 数据
  domain setup                设置本机网页版查询
  domain uninstall            确认后卸载程序

查询时的语言参数仅本次有效。
单独运行 domain en、domain zh 或 domain auto-lang 才会保存默认设置。
运行 domain --help 查看更多选项。
`)
}

func help(lang output.Language) string {
	return output.Text(lang, `Domain Check from Registries

Usage: domain [en|zh|auto-lang] [domain|URL|update|setup|uninstall]

  domain example.com          Use the default language
  domain .com                 Query a TLD at IANA
  domain .github.io           A leading dot is ignored
  domain -en example.com      English for this query
  domain zh example.com       Simplified Chinese for this query
  domain auto-lang            Follow the device language
  domain update               Refresh IANA RDAP and WHOIS data now
  domain setup                Set up the local web interface
  domain uninstall            Uninstall after confirmation
  domain https://example.com/path

Language selectors accept zero, one, or two leading hyphens and may appear
before or after the domain. The last selector wins for the current command.
Run domain en, domain zh, or domain auto-lang on its own to save the default.
With a domain, URL, help, version, update, setup, or uninstall, language selection is temporary.
Use auto-lang, -auto-lang, or --auto-lang to follow the device language.
Other device languages use English. update, setup, uninstall, help, and version also accept
zero, one, or two leading hyphens. -h and -v are shorthand for help and version.

Queries go directly to registry RDAP, or registry WHOIS on TCP port 43 when
IANA lists no RDAP service. RDAP-enabled domains can use registry WHOIS
after RDAP failure. .tw is WHOIS-only.
Delegated .ac.uk and .gov.uk names use their namespace registries directly.
Registrar referrals are never followed.
Automatic IANA refresh is checked during lookups, once per calendar month.
Use domain update to refresh dns.json and tlds.json immediately, regardless of cache age.
Single-label TLDs are queried directly at IANA RDAP. A leading dot is ignored.
Bare command names take priority; prefix a dot to query a matching TLD.
URLs are reduced to their registrable domain; the URL itself is never fetched.
Dates use YYYY-MM-DD; timestamps use YYYY-MM-DD HH:MM:SS.
Zoned timestamps are converted to UTC and carry a UTC suffix.
`, `Domain Check from Registries

用法: domain [en|zh|auto-lang] [域名|URL|update|setup|uninstall]

  domain example.com          使用默认语言
  domain .com                 向 IANA 查询顶级域名
  domain .github.io           忽略域名开头的一个点
  domain -en example.com      本次查询使用英文
  domain zh example.com       本次查询使用简体中文
  domain auto-lang            跟随设备语言
  domain update               立即更新 IANA RDAP 和 WHOIS 数据
  domain setup                设置本机网页版查询
  domain uninstall            确认后卸载程序
  domain https://example.com/path

语言参数允许不加横线、一个或两个横线，可置于域名前后；最后一个参数决定本次语言。
单独运行 domain en、domain zh 或 domain auto-lang 才会保存默认设置。
与域名、URL、help、version、update、setup 或 uninstall 一起使用时，语言选择仅本次有效。
使用 auto-lang、-auto-lang 或 --auto-lang 跟随设备语言。
其他设备语言使用英文。update、setup、uninstall、help、version 也允许不加横线、一个或两个横线。
-h 和 -v 分别是帮助和版本命令的简写。

直接查询注册局 RDAP；IANA 没有列出 RDAP 时，通过 TCP 43 查询注册局 WHOIS。
支持 RDAP 的域名在查询失败后可回退至注册局 WHOIS。
.tw 归为仅 WHOIS，始终不跟随注册商。
.ac.uk 和 .gov.uk 下的域名直接查询各自的命名空间注册局。
IANA 数据的自动更新只在查询时检查，每个日历月周期更新一次。
运行 domain update 可立即更新 dns.json 和 tlds.json，不受缓存时间限制。
单标签的顶级域名直接查询 IANA RDAP。忽略域名开头的一个点。
不带点的命令名称优先作为命令；加上前导点可查询同名顶级域名。
URL 会自动识别可注册域名，不访问 URL 页面本身。
中文日期使用 YYYY 年 M 月 D 日，有时间时保留 HH:MM:SS。
带时区的时间转换为 UTC，并添加 UTC 后缀。
`)
}
