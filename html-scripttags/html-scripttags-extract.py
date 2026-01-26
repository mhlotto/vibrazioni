#!/usr/bin/env python3
import argparse
import sys
from html.parser import HTMLParser


class ScriptTag:
    def __init__(self, attrs):
        self.attrs = attrs
        self.content_parts = []

    def add_content(self, data):
        self.content_parts.append(data)

    @property
    def content(self):
        return "".join(self.content_parts)


class ScriptExtractor(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=False)
        self.scripts = []
        self._current = None
        self._in_script = False

    def handle_starttag(self, tag, attrs):
        if tag.lower() == "script":
            self._in_script = True
            self._current = ScriptTag(attrs)

    def handle_endtag(self, tag):
        if tag.lower() == "script" and self._in_script and self._current is not None:
            self.scripts.append(self._current)
            self._current = None
            self._in_script = False

    def handle_data(self, data):
        if self._in_script and self._current is not None:
            self._current.add_content(data)


def read_input(path):
    if path:
        with open(path, "r", encoding="utf-8", errors="replace") as f:
            return f.read()
    if sys.stdin.isatty():
        raise RuntimeError("no input provided (use --file or pipe HTML)")
    return sys.stdin.read()


def format_attrs(attrs):
    parts = []
    for name, value in attrs:
        if value is None:
            parts.append(name)
        else:
            parts.append(f'{name}="{value}"')
    if not parts:
        return ""
    return " " + " ".join(parts)


def match_tag(tag, type_value, id_value, src_value):
    attrs = {}
    for name, value in tag.attrs:
        attrs[name.lower()] = value
    if type_value is not None:
        if attrs.get("type") != type_value:
            return False
    if id_value is not None:
        if attrs.get("id") != id_value:
            return False
    if src_value is not None:
        if "src" not in attrs:
            return False
        src_attr = attrs.get("src")
        if src_attr is None:
            return src_value == ""
        if src_attr != src_value:
            return False
    return True


def extract_scripts(html, type_value, id_value, src_value):
    parser = ScriptExtractor()
    parser.feed(html)
    parser.close()
    return [
        tag
        for tag in parser.scripts
        if match_tag(tag, type_value, id_value, src_value)
    ]


def write_output(path, content):
    if path:
        with open(path, "w", encoding="utf-8") as f:
            f.write(content)
        return
    sys.stdout.write(content)


def main():
    parser = argparse.ArgumentParser(description="Extract <script> tags from HTML.")
    parser.add_argument("--file", help="Input HTML file path")
    parser.add_argument("--out", help="Write output to file")
    parser.add_argument("--type", dest="type_value", help="Match script type attribute")
    parser.add_argument("--id", dest="id_value", help="Match script id attribute")
    parser.add_argument("--src", dest="src_value", help="Match script src attribute")
    parser.add_argument(
        "--clean",
        action="store_true",
        help="Remove <script> and </script> tags from output",
    )
    args = parser.parse_args()

    html = read_input(args.file)
    tags = extract_scripts(html, args.type_value, args.id_value, args.src_value)

    outputs = []
    for tag in tags:
        if args.clean:
            outputs.append(tag.content)
        else:
            attrs = format_attrs(tag.attrs)
            outputs.append(f"<script{attrs}>{tag.content}</script>")

    output = "\n".join(outputs)
    if output and not output.endswith("\n"):
        output += "\n"
    write_output(args.out, output)


if __name__ == "__main__":
    main()
