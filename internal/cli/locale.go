package cli

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/sd800/domain-check/internal/output"
)

func languageFor(locale string) output.Language {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale == "zh" || strings.HasPrefix(locale, "zh_") || strings.HasPrefix(locale, "zh-") {
		return output.Chinese
	}
	return output.English
}

func preferredLanguage() output.Language {
	// Explicit POSIX message-locale overrides take priority over device settings.
	for _, key := range []string{"LC_ALL", "LC_MESSAGES"} {
		if locale := os.Getenv(key); locale != "" {
			return languageFor(locale)
		}
	}
	if runtime.GOOS == "darwin" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		data, err := exec.CommandContext(ctx, "/usr/bin/defaults", "read", "-g", "AppleLanguages").Output()
		if err == nil {
			if language := firstAppleLanguage(string(data)); language != "" {
				return languageFor(language)
			}
		}
	}
	return languageFor(os.Getenv("LANG"))
}

func firstAppleLanguage(s string) string {
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == '(' || r == ')' || r == '"' || r == ',' || r == ' ' || r == '\n' || r == '\t' || r == '\r'
	}) {
		if part != "" {
			return part
		}
	}
	return ""
}
