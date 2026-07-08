# Kiln

![Kiln logo](kiln-logo.png)

Kiln is a small Python static site generator. It reads YAML page definitions,
renders them through Jinja2 templates, copies static files, and writes a
deployable static site.

Kiln is intentionally local-first:

- no web framework
- no live reload or file watching
- no deployment integration
- no network downloads during validation or build
- no CDN-generated scripts

## Local Development Install

From this `kiln/` directory:

```sh
python3 -m venv venv
venv/bin/python -m ensurepip --upgrade
venv/bin/python -m pip install --upgrade pip setuptools wheel
venv/bin/python -m pip install --no-build-isolation -e '.[test]'
venv/bin/kiln -h
venv/bin/python -m pytest
```

If you activate the virtualenv, use `kiln` directly:

```sh
. venv/bin/activate
kiln -h
```

## Quick Start

```sh
kiln init my-site
cd my-site
kiln new page about
kiln build
kiln package
kiln serve
```

`kiln package` creates a zip archive inside the site root by default, named
after the site directory, for example `my-site/my-site.zip`. The archive
contains the generated output directory contents, not the output directory
itself.

`kiln serve` builds the site and serves only the configured output directory.
By default this is `public/` at `http://127.0.0.1:8000/`. This command blocks
until you stop it with Ctrl-C.

## Project Layout

`kiln init` creates:

```text
site.yml
contents/index.yml
templates/default.html
static/
public/
vendor/
vendor/mathjax/
vendor/mathjax/README.md
```

## site.yml

Example:

```yaml
site:
  title: Example Site
build:
  output_dir: public
  content_dir: contents
  static_dir: static
  template_dir: templates
assets:
  allowed_js_packages:
    - mathjax
```

Build directories must be relative paths inside the site root. Kiln rejects
absolute paths, `..`, backslashes, and output directories that resolve to the
site root.

## Content YAML

Pages live under the configured content directory as `.yml` or `.yaml` files.

```yaml
page:
  title: "About"
  path: "/about/"
  layout: "default"
meta:
  description: "About this site."
content:
  - type: markdown
    value: |
      # About

      This page is written in Markdown.
```

Supported block types:

```yaml
content:
  - type: markdown
    value: |
      ## Markdown

      Markdown is rendered to HTML.

  - type: html
    value: |
      <p>Raw local HTML is allowed.</p>

  - type: image
    src: /images/logo.png
    alt: Logo

  - type: code
    language: python
    value: |
      print("hello")

  - type: math
    value: |
      E = mc^2
```

The `html` block may contain inline scripts or local script references, but
Kiln rejects external `http://`, `https://`, and protocol-relative `//` script
or stylesheet URLs.

## Templates

Templates are Jinja2 files in the configured template directory. A minimal
template:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ page.title }} - {{ site.site.title }}</title>
</head>
<body>
  <main>
    {{ content }}
  </main>
  {{ package_scripts_html | safe }}
</body>
</html>
```

`package_scripts_html` contains local script tags for requested vendored
packages such as MathJax.

## Static Assets

Files under the configured `static_dir` are copied into the configured
`output_dir`.

Example:

```text
static/css/site.css -> public/css/site.css
```

Static files are copied as output assets. Kiln does not rewrite URLs.

## No External CDN Policy

Kiln rejects generated or raw HTML containing external script or stylesheet
URLs:

- `http://...`
- `https://...`
- `//...`

Inline scripts and local scripts are allowed.

## MathJax

Pages can request MathJax:

```yaml
packages:
  - mathjax
```

The package must also be allowed in `site.yml`:

```yaml
assets:
  allowed_js_packages:
    - mathjax
```

The easiest way to install MathJax locally is:

```sh
kiln vendor mathjax --download
```

Kiln downloads a pinned MathJax package tarball, currently version `3.2.2` by
default, and installs it into:

```text
vendor/mathjax/
```

The required entrypoint is:

```text
vendor/mathjax/es5/tex-mml-chtml.js
```

Generated sites still use only local files from your output directory. Kiln
does not generate CDN URLs, and `kiln validate` and `kiln build` do not
download MathJax automatically. If you need a specific version:

```sh
kiln vendor mathjax --download --version 3.2.2
```

For offline or manually managed installs, copy an already-downloaded local
distribution with:

```sh
kiln vendor mathjax --from /path/to/mathjax
```

Use `--force` to replace an existing installed `vendor/mathjax/` directory.

During build, Kiln copies `vendor/mathjax/` to
`OUTPUT_DIR/vendor/mathjax/` and injects this local script tag only on pages
that request MathJax:

```html
<script defer src="/vendor/mathjax/es5/tex-mml-chtml.js"></script>
```

For v0.1.0, Kiln rejects symlinks inside `vendor/mathjax/` during build.
Static assets under `static_dir` are treated as trusted local project files.

## Build, Clean, Serve, Package

`kiln build` validates the site, cleans the output directory by default, copies
static assets and requested vendored packages, then renders pages.

```sh
kiln build
kiln build --no-clean
```

`kiln clean` deletes only the configured output directory and recreates it
empty. It refuses unsafe output paths.

```sh
kiln clean
```

`kiln serve` validates, builds, and serves only the configured output directory.

```sh
kiln serve --host 127.0.0.1 --port 8000
```

`kiln package` validates, builds by default, then zips the contents of the
configured output directory.

```sh
kiln package
kiln package -o dist/site.zip --force
kiln package --no-build
```

The zip contains paths like:

```text
index.html
about/index.html
css/site.css
```

It does not include `site.yml`, source directories, absolute paths, traversal
paths, or the output directory prefix.
