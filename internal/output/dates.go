package output

import (
	"fmt"
	"time"
)

// DateText localizes normalized dates without changing source precision or time zones.
func DateText(lang Language, value string) string {
	if lang != Chinese {
		return value
	}
	var layout string
	switch len(value) {
	case 10:
		layout = "2006-01-02"
	case 19:
		layout = "2006-01-02 15:04:05"
	case 23:
		layout = "2006-01-02 15:04:05 UTC"
	default:
		return value
	}
	date, err := time.Parse(layout, value)
	if err != nil {
		return value
	}
	return fmt.Sprintf("%d 年 %d 月 %d 日%s", date.Year(), date.Month(), date.Day(), value[10:])
}
