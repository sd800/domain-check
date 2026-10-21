package registry

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	whoisGMTOffset   = regexp.MustCompile(`\(?GMT([+-])(\d{1,2})(?::(\d{2}))?\)?$`)
	whoisOrdinalDate = regexp.MustCompile(`(?i)(\d{1,2})(?:st|nd|rd|th)\b`)
	whoisILChange    = regexp.MustCompile(`\b(\d{8})(?:\s+\((?:Assigned|Changed)\))?$`)
)

func formatWHOISTime(tld, value string) string {
	value = useful(value)
	if value == "" {
		return ""
	}
	if tld == "il" {
		if match := whoisILChange.FindStringSubmatch(value); match != nil {
			value = match[1]
		}
	}
	// Numeric day-first dates are enabled only for registries whose format
	// specifies that order; generic ambiguous dates keep their original text.
	if strings.Contains(" ax hk mw mk ls ve rs sm tn il pt im ", " "+tld+" ") {
		date := value
		zone := ""
		if index := strings.Index(date, " GMT"); index >= 0 {
			date, zone = date[:index], date[index:]
		}
		for _, layout := range []string{"2.1.2006 15:04:05", "2.1.2006", "2-1-2006 15:04:05", "2-1-2006", "2/1/2006 15:04:05", "2/1/2006"} {
			if parsed, err := time.Parse(layout, date); err == nil {
				if strings.Contains(layout, "15:") {
					value = parsed.Format("2006-01-02 15:04:05") + zone
				} else {
					return parsed.Format("2006-01-02")
				}
				break
			}
		}
	}
	if tld == "gg" || tld == "je" {
		value = whoisOrdinalDate.ReplaceAllString(value, "$1")
		value = strings.ReplaceAll(value, " at ", " ")
	}
	if tld == "tr" {
		value = strings.TrimSuffix(value, ".")
	}
	// These suffixes explicitly identify a timezone; unknown abbreviations
	// are left intact instead of guessing their offset.
	for suffix, offset := range map[string]string{"(JST)": "+09:00", "(UTC)": "+00:00", "(GMT)": "+00:00"} {
		if strings.HasSuffix(value, suffix) {
			value = strings.TrimSpace(strings.TrimSuffix(value, suffix)) + " " + offset
		}
	}
	if tld == "cl" {
		for suffix, offset := range map[string]string{" CLST": " -03:00", " CLT": " -04:00"} {
			if strings.HasSuffix(value, suffix) {
				value = strings.TrimSuffix(value, suffix) + offset
			}
		}
	}
	if match := whoisGMTOffset.FindStringSubmatch(value); match != nil {
		hours, _ := strconv.Atoi(match[2])
		minutes := match[3]
		if minutes == "" {
			minutes = "00"
		}
		value = strings.TrimSpace(strings.TrimSuffix(value, match[0])) + fmt.Sprintf(" %s%02d:%s", match[1], hours, minutes)
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05 -07:00", "2006/01/02 15:04:05 -07:00",
		"2006-01-02 15:04:05.999999999 -07:00",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC().Format("2006-01-02 15:04:05") + " UTC"
		}
	}
	for _, layout := range []string{
		"20060102 15:04:05", "2006/01/02 15:04:05", "2 January 2006 15:04:05.999999999",
		"Jan 2 2006 3:04PM",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("2006-01-02 15:04:05")
		}
	}
	for _, layout := range []string{"Mon Jan 2 2006", "20060102"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	return formatTime(value)
}
