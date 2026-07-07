# MathJax Example

This example shows a page that requests MathJax:

```yaml
packages:
  - mathjax
```

Kiln does not download MathJax. Before building this example, copy an
already-downloaded MathJax distribution into:

```text
vendor/mathjax/
```

The required entrypoint is:

```text
vendor/mathjax/es5/tex-mml-chtml.js
```

You can copy it with:

```sh
kiln vendor mathjax --from /path/to/mathjax examples/mathjax-site
```
