package registry

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/simplifiedchinese"
	textunicode "golang.org/x/text/encoding/unicode"
)

// This registry uses GBK/GB2312. Keep its declared UTF encodings and GB18030
// without linking the general HTML encoding index and unrelated legacy tables.
func decodeEDUCN(data []byte, contentType string) ([]byte, error) {
	name := ""
	switch {
	case bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}):
		name, data = "utf-8", data[3:]
	case bytes.HasPrefix(data, []byte{0xff, 0xfe}):
		return textunicode.UTF16(textunicode.LittleEndian, textunicode.ExpectBOM).NewDecoder().Bytes(data)
	case bytes.HasPrefix(data, []byte{0xfe, 0xff}):
		return textunicode.UTF16(textunicode.BigEndian, textunicode.ExpectBOM).NewDecoder().Bytes(data)
	default:
		name = declaredCharset(contentType)
		if name == "" {
			name = eduCNMetaCharset(data[:min(len(data), 1024)])
		}
	}
	if name == "" {
		if utf8.Valid(data) && bytes.IndexFunc(data, func(r rune) bool { return r > 127 }) >= 0 {
			name = "utf-8"
		} else {
			name = "gbk"
		}
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "utf-8", "utf8", "unicode-1-1-utf-8":
		if !utf8.Valid(data) {
			return nil, fmt.Errorf("registry response contains invalid UTF-8")
		}
		return data, nil
	case "gbk", "gb2312", "gb_2312-80", "x-gbk", "chinese", "csiso58gb231280", "iso-ir-58":
		return simplifiedchinese.GBK.NewDecoder().Bytes(data)
	case "gb18030":
		return simplifiedchinese.GB18030.NewDecoder().Bytes(data)
	case "utf-16", "utf-16le":
		return textunicode.UTF16(textunicode.LittleEndian, textunicode.IgnoreBOM).NewDecoder().Bytes(data)
	case "utf-16be":
		return textunicode.UTF16(textunicode.BigEndian, textunicode.IgnoreBOM).NewDecoder().Bytes(data)
	default:
		return nil, fmt.Errorf("registry response uses an unsupported HTML encoding")
	}
}

func declaredCharset(contentType string) string {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(params["charset"]))
}

func eduCNMetaCharset(preview []byte) string {
	z := html.NewTokenizer(bytes.NewReader(preview))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return ""
		case html.StartTagToken, html.SelfClosingTagToken:
			tag, more := z.TagName()
			if string(tag) != "meta" {
				continue
			}
			var charset, content, pragma string
			seen := uint8(0)
			for more {
				var key, value []byte
				key, value, more = z.TagAttr()
				switch string(key) {
				case "charset":
					if seen&1 == 0 {
						charset = strings.ToLower(strings.TrimSpace(string(value)))
						seen |= 1
					}
				case "content":
					if seen&2 == 0 {
						content, seen = string(value), seen|2
					}
				case "http-equiv":
					if seen&4 == 0 {
						pragma, seen = string(value), seen|4
					}
				}
			}
			if charset == "" && strings.EqualFold(pragma, "content-type") {
				charset = declaredCharset(content)
			}
			if charset != "" {
				// HTML meta declarations cannot switch ASCII markup to UTF-16.
				if strings.HasPrefix(charset, "utf-16") {
					return "utf-8"
				}
				return charset
			}
		}
	}
}

// Extract PRE text in document order without constructing a document tree.
// Tokenization still handles entities, comments and quoted tag attributes.
func eduCNPreText(decoded []byte) (string, error) {
	z := html.NewTokenizer(bytes.NewReader(decoded))
	var text strings.Builder
	depth := 0
	found, skipNewline := false, false
	suppressed := ""
	for {
		kind := z.Next()
		ignoreNewline := skipNewline
		skipNewline = false
		switch kind {
		case html.ErrorToken:
			if err := z.Err(); err != io.EOF {
				return "", err
			}
			if !found {
				return "", fmt.Errorf("registry response has no WHOIS text")
			}
			return text.String(), nil
		case html.StartTagToken, html.SelfClosingTagToken:
			tag, _ := z.TagName()
			switch string(tag) {
			case "pre":
				if depth == 0 {
					if found {
						text.WriteString("\n\n")
					}
					found = true
				}
				depth++
				skipNewline = true
			case "script", "style":
				if depth > 0 {
					suppressed = string(tag)
				}
			case "br":
				if depth > 0 && suppressed == "" {
					text.WriteByte('\n')
				}
			}
		case html.EndTagToken:
			tag, _ := z.TagName()
			if string(tag) == suppressed {
				suppressed = ""
			}
			if string(tag) == "pre" {
				depth = max(0, depth-1)
			}
		case html.TextToken:
			if depth > 0 && suppressed == "" {
				value := z.Text()
				if ignoreNewline && len(value) > 0 {
					if value[0] == '\n' {
						value = value[1:]
					}
				}
				text.Write(value)
			}
		}
	}
}
