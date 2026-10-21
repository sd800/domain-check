package registry

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/simplifiedchinese"
	textunicode "golang.org/x/text/encoding/unicode"
)

func TestEDUCNEncodingSelection(t *testing.T) {
	gbk := gbkBytes(t, "<pre>东北大学</pre>")
	gb18030, err := simplifiedchinese.GB18030.NewEncoder().Bytes([]byte("<pre>机构𠀀</pre>"))
	if err != nil {
		t.Fatal(err)
	}
	utf16, err := textunicode.UTF16(textunicode.LittleEndian, textunicode.UseBOM).NewEncoder().Bytes([]byte("<pre>东北大学</pre>"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, header, want string
		data               []byte
	}{
		{"default GBK", "text/html", "<pre>东北大学</pre>", gbk},
		{"header overrides meta", "text/html; charset=GBK", `<meta charset="utf-8"><pre>东北大学</pre>`, append([]byte(`<meta charset="utf-8">`), gbk...)},
		{"short meta", "text/html", `<meta charset='gb2312'><pre>东北大学</pre>`, append([]byte(`<meta charset='gb2312'>`), gbk...)},
		{"pragma meta", "text/html", `<META CONTENT="text/html; charset=gbk" HTTP-EQUIV="Content-Type"><pre>东北大学</pre>`, append([]byte(`<META CONTENT="text/html; charset=gbk" HTTP-EQUIV="Content-Type">`), gbk...)},
		{"BOM overrides header", "text/html; charset=gbk", "<pre>东北大学</pre>", append([]byte{0xef, 0xbb, 0xbf}, []byte("<pre>东北大学</pre>")...)},
		{"BOM UTF16", "text/html; charset=gbk", "<pre>东北大学</pre>", utf16},
		{"UTF8 autodetection", "text/html", "<pre>东北大学</pre>", []byte("<pre>东北大学</pre>")},
		{"GB18030 declaration", "text/html; charset=gb18030", "<pre>机构𠀀</pre>", gb18030},
		{"UTF16 meta follows HTML rules", "text/html", `<meta charset="utf-16"><pre>东北大学</pre>`, []byte(`<meta charset="utf-16"><pre>东北大学</pre>`)},
		{"meta without pragma ignored", "text/html", `<meta content="text/html; charset=utf-8"><pre>东北大学</pre>`, append([]byte(`<meta content="text/html; charset=utf-8">`), gbk...)},
	} {
		t.Run(test.name, func(t *testing.T) {
			decoded, err := decodeEDUCN(test.data, test.header)
			if err != nil || string(decoded) != test.want {
				t.Fatalf("decoded %q, %v; want %q", decoded, err, test.want)
			}
		})
	}
	for _, test := range []struct {
		data   []byte
		header string
	}{
		{gbk, "text/html; charset=utf-8"},
		{[]byte(`<meta charset="shift_jis"><pre>text</pre>`), "text/html"},
		{[]byte(`<pre>text</pre>`), "text/html; charset=unknown-encoding"},
	} {
		if _, err := decodeEDUCN(test.data, test.header); err == nil {
			t.Fatal("silently accepted an unsupported or invalid encoding")
		}
	}
}

func TestEDUCNStreamingTextMatchesDocumentSemantics(t *testing.T) {
	for _, source := range []string{
		"<PRE>\nDomain Name: example.edu.cn\r\nName Server: ns.example.test</PRE>",
		`<!-- <pre>wrong</pre> --><pre title="a > b">Example &amp; Research<br>Domain Name: <a href="https://registrar.test/">example.edu.cn</a><BR/>Name Server: ns.example.test</pre>`,
		`<script>"<pre>wrong</pre>"</script><pre>first<script>wrong</script><style>wrong</style>last</pre>`,
		"<pre>\nfirst</pre><div>ignored</div><pre>\nsecond</pre>",
		"<pre>first<pre>\nsecond</pre>third</pre>",
		"<pre><!-- comment -->\nfirst</pre>",
		"<pre><span></span>\nfirst</pre>",
		"<pre><script>ignored</script>\nfirst</pre>",
		"<pre>\n\nfirst</pre>",
		"<pre>unfinished &lt; text",
		"<pre></pre><pre>second</pre>",
	} {
		t.Run(source, func(t *testing.T) {
			want := documentPREText(t, source)
			got, err := eduCNPreText([]byte(source))
			if err != nil || got != want {
				t.Fatalf("streaming %q, %v; document %q", got, err, want)
			}
		})
	}
	if _, err := eduCNPreText([]byte(`<html><script>"<pre>wrong</pre>"</script></html>`)); err == nil {
		t.Fatal("script contents became a WHOIS text block")
	}
}

// Use the former full-document semantics as an independent test oracle. Its
// document parser is linked into tests only, never the production executable.
func documentPREText(t *testing.T, source string) string {
	t.Helper()
	root, err := html.Parse(bytes.NewBufferString(source))
	if err != nil {
		t.Fatal(err)
	}
	var blocks []string
	var text func(*html.Node, *strings.Builder)
	text = func(n *html.Node, b *strings.Builder) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		if n.Type == html.ElementNode {
			if n.Data == "script" || n.Data == "style" {
				return
			}
			if n.Data == "br" {
				b.WriteByte('\n')
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			text(child, b)
		}
	}
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "pre" {
			var b strings.Builder
			text(n, &b)
			blocks = append(blocks, b.String())
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return strings.Join(blocks, "\n\n")
}

func TestEDUCNStreamingWHOISFields(t *testing.T) {
	plain := "<pre>Example &amp; Organization<br>Domain Name: <a href='https://registrar.test/'>example.edu.cn</a><br>Name Server: ns1.example.test<br></pre>"
	r, err := parseEDUCN("example.edu.cn", []byte(plain), "text/html; charset=utf-8")
	if err != nil || r.Registrant != "Example & Organization" || !reflect.DeepEqual(r.Nameservers, []string{"ns1.example.test"}) {
		t.Fatalf("%+v, %v", r, err)
	}
}
