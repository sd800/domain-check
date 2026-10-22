package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/rivo/uniseg"
	"github.com/sd800/domain-check/internal/registry"
)

type Language string

const (
	English Language = "en"
	Chinese Language = "zh"
)

var labels = map[Language][]string{
	English: {"Domain Name", "Registration Time", "Expiration Time", "Last Changed Time", "Domain Status", "Registrar", "Registrant", "Registrant Email", "Name Server", "DNSSEC", "Result", "IDN", "Address", "Registration URL", "Admin", "Organization", "Role", "Phone", "Email"},
	Chinese: {"域名", "注册时间", "过期时间", "更新时间", "状态", "注册商", "注册人", "联系邮箱", "域名服务器", "DNSSEC", "查询结果", "国际化域名", "地址", "注册网址", "管理联系人", "组织", "角色", "电话", "邮箱"},
}

type entry struct {
	index int
	value string
}

func Text(lang Language, en, zh string) string {
	if lang == Chinese {
		return zh
	}
	return en
}

func ResultText(lang Language, result registry.Result) string {
	switch result {
	case registry.NotRegistered:
		return Text(lang, "Not registered", "未注册")
	case registry.Reserved:
		return Text(lang, "Reserved", "已保留")
	case registry.NotAllowed:
		return Text(lang, "Registration not allowed", "不允许注册")
	case registry.Restricted:
		return Text(lang, "Registration restricted (online registration unavailable)", "注册受限（无法在线注册）")
	case registry.Unavailable:
		return Text(lang, "Not available for registration", "不可注册")
	default:
		return Text(lang, "No registry record", "注册局无记录")
	}
}

func Format(w io.Writer, record registry.Record, lang Language, width int) error {
	if lang != Chinese {
		lang = English
	}
	names := labels[lang]
	longest := 0
	for i, label := range names {
		if lang == Chinese && i == 9 { // DNSSEC does not set the Chinese content column.
			continue
		}
		longest = max(longest, uniseg.StringWidth(label))
	}
	column := longest + 4 // colon plus three spaces after the longest label
	var b strings.Builder
	row := func(index int, value string) {
		label := names[index]
		padding := column - uniseg.StringWidth(label) - 1
		prefix := label + ":" + strings.Repeat(" ", padding)
		lines := []string{value}
		if index == 12 { // Postal address lines share one label and content column.
			lines = strings.Split(value, "\n")
		}
		first := true
		for _, value := range lines {
			if index != 11 { // The locally validated IDN may contain emoji ZWJ.
				value = registry.Clean(value)
			}
			if index >= 1 && index <= 3 {
				value = DateText(lang, value)
			}
			for _, line := range wrap(value, max(2, width-column)) {
				if first {
					b.WriteString(prefix)
					first = false
				} else {
					b.WriteString(strings.Repeat(" ", column))
				}
				b.WriteString(line)
				b.WriteByte('\n')
			}
		}
	}
	row(0, record.Domain)
	row(11, unicodeDomain(record.Domain))
	if record.Result != "" {
		result := ResultText(lang, record.Result)
		if record.TLD && record.Result == registry.NoRecord {
			result = Text(lang, "No IANA TLD record", "IANA 无此顶级域名记录")
		}
		row(10, result)
		_, err := io.WriteString(w, b.String())
		return err
	}
	// Blank lines separate only nonempty groups, preserving the compact layout.
	groups := [][]entry{
		{{1, record.Registration}, {2, record.Expiration}, {3, record.Changed}},
	}
	var registrarGroup []entry
	if record.TLD {
		groups[0] = []entry{{1, record.Registration}, {3, record.Changed}}
		for _, status := range record.Statuses {
			groups[0] = append(groups[0], entry{4, status})
		}
	} else {
		registrarGroup = append(registrarGroup, entry{5, record.Registrar})
		for _, status := range record.Statuses {
			registrarGroup = append(registrarGroup, entry{4, status})
		}
	}
	contactGroup := []entry{{6, record.Registrant}}
	if record.TLD {
		contactGroup = append(contactGroup, entry{12, strings.Join(record.RegistrantAddress, "\n")}, entry{13, record.RegistrationURL})
	} else {
		groups = append(groups, registrarGroup)
		for _, email := range record.RegistrantEmail {
			contactGroup = append(contactGroup, entry{7, email})
		}
	}
	groups = append(groups, contactGroup)
	if record.TLD {
		admin := []entry{{14, record.Admin}, {15, record.AdminOrganization}, {16, record.AdminRole}}
		for _, phone := range record.AdminPhones {
			admin = append(admin, entry{17, phone})
		}
		for _, email := range record.AdminEmails {
			admin = append(admin, entry{18, email})
		}
		groups = append(groups, admin)
	}
	var nsGroup []entry
	for _, ns := range record.Nameservers {
		nsGroup = append(nsGroup, entry{8, ns})
	}
	nsGroup = append(nsGroup, entry{9, record.DNSSEC})
	groups = append(groups, nsGroup)
	for _, group := range groups {
		nonempty := false
		for _, entry := range group {
			nonempty = nonempty || registry.Clean(entry.value) != ""
		}
		if !nonempty {
			continue
		}
		b.WriteByte('\n')
		for _, entry := range group {
			row(entry.index, entry.value)
		}
	}
	_, err := fmt.Fprint(w, b.String())
	return err
}

// Prefer word boundaries; split long tokens by grapheme cluster, never bytes.
func wrap(value string, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(value) {
		if line != "" && uniseg.StringWidth(line)+1+uniseg.StringWidth(word) <= width {
			line += " " + word
			continue
		}
		if line != "" {
			lines = append(lines, line)
			line = ""
		}
		clusters := uniseg.NewGraphemes(word)
		for clusters.Next() {
			cluster := clusters.Str()
			if line != "" && uniseg.StringWidth(line)+uniseg.StringWidth(cluster) > width {
				lines = append(lines, line)
				line = ""
			}
			line += cluster
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
