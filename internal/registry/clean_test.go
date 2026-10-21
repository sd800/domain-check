package registry

import (
	"strings"
	"testing"
	"unicode"
)

func FuzzCleanPreservesSanitization(f *testing.F) {
	for _, seed := range []string{
		"", "Domain Name", "Example Organization", "上海溪诚信息科技有限公司",
		"  repeated   spaces  ", "tabs\tand\nnewlines", "\x1b]2;title\a",
		"left\u200bright", "\u00a0word\u3000", "e\u0301 \U0001f600", string([]byte{0xff, ' ', 0xfe}),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		want := strings.Join(strings.FieldsFunc(input, func(r rune) bool {
			return unicode.IsSpace(r) || unicode.IsControl(r) || unicode.In(r, unicode.Cf)
		}), " ")
		if got := Clean(input); got != want {
			t.Fatalf("sanitization changed: got %q; want %q", got, want)
		}
	})
}
