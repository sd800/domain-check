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
		case "auto", "-auto", "--auto":
			o.lang = lang
			o.preference = automaticLanguage
		case "help", "-help", "--help", "-h":
			o.help = true
		case "version", "-version", "--version", "-v":
			o.version = true
		case "update", "-update", "--update":
			o.update = true
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
	o, err := parseArgs(args, preferredLanguage())
	if o.preference == "" {
		if saved, loadErr := loadLanguagePreference(); loadErr != nil {
			fmt.Fprintln(stderr, "domain: "+output.Text(o.lang, "Cannot read the saved language; using the device language", "无法读取已保存的语言，使用设备语言")+": "+registry.Clean(loadErr.Error()))
		} else if saved == output.English || saved == output.Chinese {
			o.lang = saved
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, output.Text(o.lang, "Usage: domain [en|zh|auto] [domain|URL|update]", "用法: domain [en|zh|auto] [域名|URL|update]"))
		fmt.Fprintln(stderr, "domain: "+output.Text(o.lang, "Invalid arguments, domain, or URL", "参数、域名或 URL 无效")+": "+registry.Clean(err.Error()))
		return 2
	}
	preferenceSaved := false
	if o.preference != "" {
		if err := saveLanguagePreference(o.preference); err != nil {
			fmt.Fprintln(stderr, "domain: "+output.Text(o.lang, "Cannot save the language preference", "无法保存语言偏好")+": "+registry.Clean(err.Error()))
		} else {
			preferenceSaved = true
		}
	}
	if o.help {
		fmt.Fprint(stdout, help(o.lang))
		return 0
	}
	if o.version {
		fmt.Fprintln(stdout, "Domain Check from Registries "+version)
		return 0
	}
	if o.intro {
		var message string
		if o.preference != "" {
			if !preferenceSaved {
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
		if code == "stale-bootstrap" || code == "stale-whois" {
			message = output.Text(o.lang, "Cannot refresh IANA data; using a previously validated cache.", "无法更新 IANA 数据，使用已验证的旧缓存。")
		}
		fmt.Fprintln(stderr, "domain: "+message)
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if o.update {
		if err := client.UpdateBootstrap(ctx); err != nil {
			fmt.Fprintln(stderr, "domain: "+errorText(o.lang, err))
			return 3
		}
		if _, err := fmt.Fprintln(stdout, output.Text(o.lang, "IANA DNS bootstrap data updated.", "IANA DNS 引导数据已更新。")); err != nil {
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
		"edu-cn-failed":           {"Domain query failed", "域名查询失败"},
		"dns-failed":              {"DNS nameserver query failed", "DNS 域名服务器查询失败"},
		"bootstrap-failed":        {"Cannot load IANA RDAP bootstrap data", "无法读取 IANA RDAP 引导数据"},
		"bootstrap-update-failed": {"Cannot update IANA DNS bootstrap data", "无法更新 IANA DNS 引导数据"},
		"bootstrap-cache-failed":  {"Cannot save the IANA DNS bootstrap cache", "无法保存 IANA DNS 引导数据缓存"},
		"iana-whois-failed":       {"Cannot discover the registry WHOIS server from IANA", "无法通过 IANA 获取注册局 WHOIS 服务器"},
		"unsupported-suffix":      {"Unknown or unsupported domain suffix", "未知或不支持的域名后缀"},
		"no-registry-service":     {"No public registry RDAP or WHOIS service", "注册局没有公开的 RDAP 或 WHOIS 服务"},
		"rdap-failed":             {"Registry RDAP query failed", "注册局 RDAP 查询失败"},
		"whois-failed":            {"Registry WHOIS query failed", "注册局 WHOIS 查询失败"},
		"invalid-response":        {"Invalid registry response", "注册局响应无效"},
		"redirect-blocked":        {"Redirect blocked: queries stay within IANA-listed registry endpoints", "已阻止重定向：仅查询 IANA 列出的注册局端点"},
		"rate-limited":            {"Registry rate limit reached; try again later", "已触发注册局查询频率限制，请稍后重试"},
		"access-denied":           {"Registry denied access", "注册局拒绝访问"},
		"invalid-domain":          {"Invalid domain", "域名无效"},
	}
	message, ok := messages[query.Code]
	if !ok {
		return registry.Clean(query.Error())
	}
	return output.Text(lang, message[0], message[1]) + ": " + query.Detail
}

func introduction(lang output.Language, version string) string {
	return "Domain Check from Registries\nversion " + version + "\n© 2026 Songming.org\n\n" + output.Text(lang, `Query domain registration information directly from registries.
Registrar referrals are never followed.

Usage: domain [en|zh|auto] [domain|URL|update]

  domain example.com          Use the default language
  domain -en example.com      English
  domain zh example.com       Simplified Chinese
  domain auto                 Follow the device language
  domain update               Refresh IANA DNS bootstrap data now

Language choices are saved for future commands.
Run domain --help for more options.
`, `直接向注册局查询域名注册信息，始终不跟随注册商。

用法: domain [en|zh|auto] [域名|URL|update]

  domain example.com          使用默认语言
  domain -en example.com      英文
  domain zh example.com       简体中文
  domain auto                 跟随设备语言
  domain update               立即更新 IANA DNS 引导数据

语言选择会保存，后续调用自动沿用。
运行 domain --help 查看更多选项。
`)
}

func help(lang output.Language) string {
	return output.Text(lang, `Domain Check from Registries

Usage: domain [en|zh|auto] [domain|URL|update]

  domain example.com          Use the default language
  domain -en example.com      English
  domain zh example.com       Simplified Chinese
  domain auto                 Follow the device language
  domain update               Refresh IANA DNS bootstrap data now
  domain https://example.com/path

Language selectors accept zero, one, or two leading hyphens and may appear
before or after the domain. The last selector wins and is saved for future
commands. Use auto, -auto, or --auto to restore the device language.
Other device languages use English. update, help, and version also accept
zero, one, or two leading hyphens. -h and -v are shorthand for help and version.

Queries go directly to registry RDAP, or registry WHOIS on TCP port 43 when
IANA lists no RDAP service. Registrar referrals are never followed.
Automatic IANA refresh is checked during lookups, once per calendar month.
Use domain update to refresh dns.json immediately, regardless of cache age.
URLs are reduced to their registrable domain; the URL itself is never fetched.
Dates use YYYY-MM-DD; timestamps use YYYY-MM-DD HH:MM:SS.
Zoned timestamps are converted to UTC and carry a UTC suffix.
`, `Domain Check from Registries

用法: domain [en|zh|auto] [域名|URL|update]

  domain example.com          使用默认语言
  domain -en example.com      英文
  domain zh example.com       简体中文
  domain auto                 跟随设备语言
  domain update               立即更新 IANA DNS 引导数据
  domain https://example.com/path

语言参数允许不加横线、一个或两个横线，可置于域名前后；最后一个参数会保存。
使用 auto、-auto 或 --auto 恢复跟随设备语言。
其他设备语言使用英文。update、help、version 也允许不加横线、一个或两个横线。
-h 和 -v 分别是帮助和版本命令的简写。

直接查询注册局 RDAP；IANA 没有列出 RDAP 时，通过 TCP 43 查询注册局 WHOIS。
始终不跟随注册商。IANA 数据的自动更新只在查询时检查，每个日历月周期更新一次。
运行 domain update 可立即更新 dns.json，不受缓存时间限制。
URL 会自动识别可注册域名，不访问 URL 页面本身。
日期使用 YYYY-MM-DD，时间使用 YYYY-MM-DD HH:MM:SS。
带时区的时间转换为 UTC，并添加 UTC 后缀。
`)
}
