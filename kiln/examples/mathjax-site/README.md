# MathJax Example

This example shows a page that requests MathJax:

```yaml
packages:
  - mathjax
```

Before building this example, vendor MathJax locally:

```sh
kiln vendor mathjax --download examples/mathjax-site
```

Kiln downloads a pinned local copy for vendoring, but generated pages still use
local files only and builds do not download MathJax automatically.

The files are installed into:

```text
vendor/mathjax/
```

The required entrypoint is:

```text
vendor/mathjax/es5/tex-mml-chtml.js
```

For offline use, copy an already-downloaded local distribution with:

```sh
kiln vendor mathjax --from /path/to/mathjax examples/mathjax-site
```
