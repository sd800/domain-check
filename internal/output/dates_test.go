package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sd800/domain-check/internal/registry"
)

func TestChineseDatesPreservePrecisionAndZone(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"1994-04-08 04:00:00 UTC", "1994 年 4 月 8 日 04:00:00 UTC"},
		{"1994-04-08 04:00:00", "1994 年 4 月 8 日 04:00:00"},
		{"1985-05-23", "1985 年 5 月 23 日"},
		{"2024-02-29", "2024 年 2 月 29 日"},
		{"2025-02-29", "2025-02-29"},
		{"approximately 1994", "approximately 1994"},
		{"1994-04", "1994-04"},
		{"", ""},
	} {
		if got := DateText(Chinese, tc.input); got != tc.want {
			t.Errorf("%q: %q, want %q", tc.input, got, tc.want)
		}
		if got := DateText(English, tc.input); got != tc.input {
			t.Errorf("English date changed: %q", got)
		}
	}
}

func TestChineseRecordDatesAndUnchangedStatus(t *testing.T) {
	var b bytes.Buffer
	err := Format(&b, registry.Record{Domain: "example.com", Registration: "1994-04-08 04:00:00 UTC", Expiration: "2027-05-23", Changed: "2026-10-22 03:10:00", Statuses: []string{"client transfer prohibited"}}, Chinese, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"1994 年 4 月 8 日 04:00:00 UTC", "2027 年 5 月 23 日", "2026 年 10 月 22 日 03:10:00", "client transfer prohibited"} {
		if !strings.Contains(b.String(), value) {
			t.Errorf("missing %q in %s", value, b.String())
		}
	}
}
