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
	domain  string
	lang    output.Language
	help    bool
	version bool
}

func parseArgs(args []string, lang output.Language) (options, error) {
	o := options{lang: lang}
	for _, arg := range args {
		switch strings.ToLower(arg) {
		case "en", "-en", "--en":
			o.lang = output.English
		case "zh", "-zh", "--zh":
			o.lang = output.Chinese
		case "help", "-h", "--help":
			o.help = true
		case "version", "-v", "--version":
			o.version = true
		default:
			if strings.HasPrefix(arg, "-") || o.domain != "" {
				return o, fmt.Errorf("unexpected argument: %s", registry.Clean(arg))
			}
			o.domain = arg
		}
	}
	if o.help || o.version {
		return o, nil
	}
	if o.domain == "" {
		return o, fmt.Errorf("missing domain")
	}
	domain, err := registry.NormalizeDomain(o.domain)
	if err != nil {
		return o, err
	}
	o.domain = domain
	return o, nil
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer, version string) int {
	o, err := parseArgs(args, preferredLanguage())
	if err != nil {
		fmt.Fprintln(stderr, output.Text(o.lang, "Usage: domain [en|zh] <domain>", "用法: domain [en|zh] <域名>"))
		fmt.Fprintln(stderr, "domain: "+output.Text(o.lang, "Invalid arguments or domain", "参数或域名无效")+": "+registry.Clean(err.Error()))
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
	client := registry.NewClient(registry.DefaultCacheDir())
	client.Warn = func(code string) {
		message := output.Text(o.lang, "Cannot save the local cache; continuing online.", "无法保存本地缓存，继续使用在线数据。")
		if code == "stale-bootstrap" || code == "stale-whois" {
			message = output.Text(o.lang, "Cannot refresh IANA data; using a previously validated cache.", "无法更新 IANA 数据，使用已验证的旧缓存。")
		}
		fmt.Fprintln(stderr, "domain: "+message)
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
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
		"bootstrap-failed":    {"Cannot load IANA RDAP bootstrap data", "无法读取 IANA RDAP 引导数据"},
		"iana-whois-failed":   {"Cannot discover the registry WHOIS server from IANA", "无法通过 IANA 获取注册局 WHOIS 服务器"},
		"unsupported-suffix":  {"Unknown or unsupported domain suffix", "未知或不支持的域名后缀"},
		"no-registry-service": {"No public registry RDAP or WHOIS service", "注册局没有公开的 RDAP 或 WHOIS 服务"},
		"rdap-failed":         {"Registry RDAP query failed", "注册局 RDAP 查询失败"},
		"whois-failed":        {"Registry WHOIS query failed", "注册局 WHOIS 查询失败"},
		"invalid-response":    {"Invalid registry response", "注册局响应无效"},
		"redirect-blocked":    {"Redirect blocked: queries stay within IANA-listed registry endpoints", "已阻止重定向：仅查询 IANA 列出的注册局端点"},
		"rate-limited":        {"Registry rate limit reached; try again later", "已触发注册局查询频率限制，请稍后重试"},
		"access-denied":       {"Registry denied access", "注册局拒绝访问"},
		"invalid-domain":      {"Invalid domain", "域名无效"},
	}
	message, ok := messages[query.Code]
	if !ok {
		return registry.Clean(query.Error())
	}
	return output.Text(lang, message[0], message[1]) + ": " + query.Detail
}

func help(lang output.Language) string {
	return output.Text(lang, `Domain Check from Registries

Usage: domain [en|zh] <domain>

  domain example.com          Use the device language
  domain -en example.com      English
  domain zh example.cn        Simplified Chinese

Language selectors accept zero, one, or two leading hyphens and may appear
before or after the domain. The last selector wins. Other device languages
use English. --help and --version display help and the build version.

Queries go directly to registry RDAP, or registry WHOIS on TCP port 43 when
IANA lists no RDAP service. Registrar referrals are never followed.
IANA data is refreshed at most once per calendar month, only during a lookup.
Zoned timestamps are displayed in UTC; unzoned WHOIS values are preserved.
`, `Domain Check from Registries

用法: domain [en|zh] <域名>

  domain example.com          按设备语言输出
  domain -en example.com      英文
  domain zh example.cn        简体中文

语言参数允许不加横线、一个或两个横线，可置于域名前后；最后一个参数生效。
其他设备语言使用英文。--help 显示帮助，--version 显示版本。

直接查询注册局 RDAP；IANA 没有列出 RDAP 时，通过 TCP 43 查询注册局 WHOIS。
始终不跟随注册商。IANA 数据只在查询时检查，每个日历月周期更新一次。
带时区的时间统一显示为 UTC；未注明时区的 WHOIS 时间保留原值。
`)
}
