#!/usr/bin/env python3
"""Check the buildless website's offline assets and canonical installation text."""

from html.parser import HTMLParser
from pathlib import Path
import re
from urllib.parse import unquote, urlsplit


ROOT = Path(__file__).resolve().parents[1]
WEB = ROOT / "web"


class Page(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.ids = set()
        self.references = []
        self.install = []
        self.in_install = False
        self.headings = 0

    def handle_starttag(self, tag, attributes):
        attrs = dict(attributes)
        identity = attrs.get("id")
        if identity:
            assert identity not in self.ids, f"Duplicate ID: {identity}"
            self.ids.add(identity)
        assert not any(key.startswith("on") for key in attrs), "Use explicit event listeners"
        assert tag not in {"base", "iframe", "form"}, f"Unexpected interactive resource: {tag}"
        if tag == "h1":
            self.headings += 1
        if tag == "code" and identity == "install-command":
            self.in_install = True
        if tag == "script":
            assert attrs.get("type") != "module", "Classic scripts support file:// opening"
        if tag == "img":
            assert "alt" in attrs, "Images need an alt attribute"
        for key in ("src", "href"):
            if key in attrs:
                resource = tag != "a"
                self.references.append((attrs[key], resource))
        assert "srcset" not in attrs, "Check all responsive asset references explicitly"

    def handle_endtag(self, tag):
        if tag == "code":
            self.in_install = False

    def handle_data(self, data):
        if self.in_install:
            self.install.append(data)


def main():
    for path in WEB.rglob("*"):
        assert not path.is_symlink(), f"Publish regular website files only: {path}"
    html = (WEB / "index.html").read_text()
    page = Page()
    page.feed(html)
    assert page.headings == 1, "Use one primary heading"
    for reference, resource in page.references:
        target = urlsplit(reference)
        if target.scheme or target.netloc:
            assert not resource, f"Keep assets local: {reference}"
            assert target.scheme == "https", f"External links must use HTTPS: {reference}"
            continue
        assert not target.path.startswith("/"), f"Use relative paths for Pages and file://: {reference}"
        if target.path:
            path = (WEB / unquote(target.path)).resolve()
            assert path.is_relative_to(WEB), f"Asset escapes web/: {reference}"
            assert path.is_file(), f"Missing local asset: {reference}"
        elif target.fragment:
            assert unquote(target.fragment) in page.ids, f"Missing anchor: {reference}"
    readme = (ROOT / "README.md").read_text()
    commands = re.findall(r"```sh\n(.*?)\n```", readme, flags=re.S)
    canonical = next(command for command in commands if command.startswith("/bin/sh <<'SH'"))
    assert "".join(page.install).strip() == canonical, "Website installer differs from README"
    css = (WEB / "assets/site.css").read_text()
    js = (WEB / "assets/site.js").read_text()
    for path, source in (("CSS", css), ("JavaScript", js)):
        assert not re.search(r"https?://|@import", source), f"Unexpected remote resource in {path}"
    assert not re.search(r"\b(?:fetch|XMLHttpRequest|WebSocket|EventSource)\s*\(", js), "Website must work offline"
    assert not re.search(r"(?:^|\n)\s*import\b", js), "No module loading for file://"
    assert not re.search(r"letter-spacing\s*:\s*-", css), "Keep normal letter spacing"
    print("Website checked: local assets, anchors, offline scripts, and canonical installer.")


if __name__ == "__main__":
    main()
