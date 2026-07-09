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
kiln new post hello-world
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
  base_url: https://example.com
build:
  output_dir: public
  content_dir: contents
  static_dir: static
  template_dir: templates
assets:
  allowed_js_packages:
    - mathjax
sitemap:
  enabled: true
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
sitemap:
  changefreq: weekly
  priority: 0.8
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

## Custom Page Data

Unknown top-level fields in a content YAML file are available to templates
under `data`. Kiln excludes its own reserved fields such as `page`, `meta`,
`content`, `packages`, `post`, `ads`, `sitemap`, `integrations`, and `robots`.

Example directory page data:

```yaml
page:
  title: "Restaurants in Amherst, MA"
  path: "/restaurants/amherst/"
  layout: "directory"
meta:
  description: "Restaurants in Amherst, MA."
content:
  - type: markdown
    value: |
      # Restaurants
restaurants:
  - name: "Bueno y Sano"
    cuisine:
      - "Mexican"
```

Template usage:

```html
{% for restaurant in data.restaurants %}
  <h2>{{ restaurant.name }}</h2>
{% endfor %}
```

This is useful for simple directories and structured one-off pages before
Kiln has formal custom collections. Kiln does not validate custom data schemas
in this pass.

## Posts

Create a post source file with:

```sh
kiln new post hello-world
kiln new post notes/hello-world
```

Posts are ordinary pages with a top-level `post:` object. `kiln new post`
creates files under `contents/posts/` and uses the `post` layout:

```yaml
page:
  title: "Hello World"
  path: "/posts/hello-world/"
  layout: "post"
post:
  date: "2026-07-09"
  draft: false
  tags: []
meta:
  description: ""
content:
  - type: markdown
    value: |
      # Hello World
```

`post.date` is required and must use `YYYY-MM-DD`. `post.draft` is optional
and defaults to `false`. `post.tags` is optional and defaults to `[]`; when
present it must be a list of strings.

Kiln exposes posts to all templates as `collections.posts`:

```yaml
collections:
  posts:
    - title: "Hello World"
      path: "/posts/hello-world/"
      url: "/posts/hello-world/"
      date: "2026-07-09"
      tags: []
      draft: false
      meta: {}
```

Draft posts are still built, but they are excluded from `collections.posts`.
Posts are sorted newest first by `post.date`, then title ascending.

A simple blog index template can render links to posts:

```html
<ul>
{% for post in collections.posts %}
  <li><a href="{{ post.url }}">{{ post.title }}</a> {{ post.date }}</li>
{% endfor %}
</ul>
```

## Templates

Templates are Jinja2 files in the configured template directory. A minimal
template:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ page.title }} - {{ site.site.title }}</title>
  {{ head_integrations_html | safe }}
</head>
<body>
  <main>
    {{ content }}
  </main>
  {{ package_scripts_html | safe }}
</body>
</html>
```

`head_integrations_html` contains validated head scripts for enabled
integrations such as AdSense Auto ads. `package_scripts_html` contains local
script tags for requested vendored packages such as MathJax.

## URL Prefixes

If a site is deployed under a subdirectory, include the path in
`site.base_url`:

```yaml
site:
  title: Amherst Area
  base_url: "https://cw-complex.com/amherst-area"
```

Kiln exposes `site.base_path` to templates. For the example above it is
`/amherst-area`; for `https://cw-complex.com` it is empty.

Use the `url()` helper for internal links and assets:

```html
<a href="{{ url('/') }}">Home</a>
<link rel="stylesheet" href="{{ url('/css/site.css') }}">
<a href="{{ url('/restaurants/amherst/') }}">Restaurants</a>
```

This renders correctly both at the domain root and under a path prefix. Existing
hardcoded root-relative links still render as written, but they may not work
when deployed under a subdirectory.

Markdown content may use normal root-relative links and images:

```markdown
[Amherst](/restaurants/amherst/)
![Logo](/images/logo.png)
```

When `site.base_url` contains a path prefix, Kiln rewrites Markdown-generated
internal `href` and `src` attributes automatically, for example to
`/amherst-area/restaurants/amherst/`. External URLs, anchors, `mailto:`, and
`tel:` links are left unchanged.

## Static Assets

Files under the configured `static_dir` are copied into the configured
`output_dir`.

Example:

```text
static/css/site.css -> public/css/site.css
```

Static files are copied as output assets. Kiln does not rewrite URLs.

## Sitemap

Kiln can generate `sitemap.xml` in the configured output directory:

```yaml
site:
  title: Example Site
  base_url: https://example.com
sitemap:
  enabled: true
```

`site.base_url` is required when sitemap generation is enabled and must be an
absolute `http://` or `https://` URL. Kiln normalizes it by removing a trailing
slash, so page path `/about/` becomes `https://example.com/about/`.

Pages are included by default. Exclude a page with:

```yaml
sitemap:
  enabled: false
```

Optional page sitemap metadata:

```yaml
sitemap:
  enabled: true
  changefreq: weekly
  priority: 0.8
```

`changefreq` may be `always`, `hourly`, `daily`, `weekly`, `monthly`,
`yearly`, or `never`. `priority` must be a number from `0.0` to `1.0`.
Static files and vendored package assets are not included.

## Robots

Kiln can generate `robots.txt` in the configured output directory:

```yaml
site:
  title: Example Site
  base_url: https://example.com
robots:
  enabled: true
  allow_all: true
  sitemap: true
```

`robots.enabled` defaults to `false`. `robots.allow_all` defaults to `true`;
when true, Kiln emits:

```text
User-agent: *
Allow: /
```

Set `allow_all: false` to emit:

```text
User-agent: *
Disallow: /
```

`robots.sitemap` defaults to `true` and appends:

```text
Sitemap: https://example.com/sitemap.xml
```

When `robots.sitemap` is true, `site.base_url` is required and must be an
absolute `http://` or `https://` URL. This does not automatically enable
sitemap generation, but most sites should enable both `robots.sitemap: true`
and `sitemap.enabled: true`.

## No External CDN Policy

Kiln rejects generated or raw HTML containing external script or stylesheet
URLs:

- `http://...`
- `https://...`
- `//...`

Inline scripts and local scripts are allowed.

The one first-class external script exception is Google AdSense Auto ads, and
only when enabled through the structured `integrations.ads` config below. Kiln
still rejects arbitrary external scripts, including other
`googlesyndication.com` URLs.

## AdSense Auto Ads

Enable Google AdSense Auto ads in `site.yml`:

```yaml
integrations:
  ads:
    provider: adsense
    enabled: true
    mode: auto
    client: ca-pub-1234567890123456
```

When enabled, Kiln emits the official AdSense Auto ads script into each page
head:

```html
<script async src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-1234567890123456" crossorigin="anonymous"></script>
```

This is an explicit exception to Kiln's no-arbitrary-external-script policy.
The generated site will load Google's AdSense script at runtime. Kiln does not
vendor, download, or rewrite this script during build.

Ads are disabled by default. The configured client must use the conservative
`ca-pub-` plus digits format. Only `provider: adsense` and `mode: auto` are
supported in this first pass.

When site ads are enabled, every page gets Auto ads by default. Disable ads on
individual pages with page-level YAML:

```yaml
ads:
  enabled: false
```

This is useful for pages where ads do not belong, such as privacy, about,
contact, or legal pages. Page-level `ads.enabled: true` is allowed but
redundant; it cannot enable ads if site-wide ads are disabled.

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

If `site.base_url` contains a path prefix, Kiln prefixes this local script URL
the same way as `url('/vendor/mathjax/es5/tex-mml-chtml.js')`.

For v0.1.0, Kiln rejects symlinks inside `vendor/mathjax/` during build.
Static assets under `static_dir` are treated as trusted local project files.

## Build, Clean, Serve, Package

`kiln build` validates the site, cleans the output directory by default, copies
static assets and requested vendored packages, renders pages, and writes
optional generated files such as `sitemap.xml` and `robots.txt`.

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
