from __future__ import annotations

import io
import shutil
import tarfile
import zipfile
from pathlib import Path
from unittest.mock import patch

import pytest

from kiln.cli import main
from kiln.core import KilnError, download_vendor_package, validate_site


MATHJAX_SCRIPT = '<script defer src="/vendor/mathjax/es5/tex-mml-chtml.js"></script>'
ADSENSE_CLIENT = "ca-pub-1234567890123456"
ADSENSE_URL = (
    "https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js"
    f"?client={ADSENSE_CLIENT}"
)
ADSENSE_SCRIPT = (
    f'<script async src="{ADSENSE_URL}" crossorigin="anonymous"></script>'
)


def write_minimal_site(root: Path) -> None:
    (root / "contents").mkdir()
    (root / "templates").mkdir()
    (root / "static").mkdir()
    (root / "public").mkdir()
    (root / "site.yml").write_text("site:\n  title: Test Site\n", encoding="utf-8")
    (root / "templates" / "default.html").write_text(
        "<html><body>{{ content }}{{ package_scripts_html | safe }}</body></html>",
        encoding="utf-8",
    )
    (root / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )


def allow_mathjax(root: Path) -> None:
    (root / "site.yml").write_text(
        """site:
  title: Test Site
assets:
  allowed_js_packages:
    - mathjax
""",
        encoding="utf-8",
    )


def enable_adsense(root: Path, client: str = ADSENSE_CLIENT) -> None:
    (root / "site.yml").write_text(
        f"""site:
  title: Test Site
integrations:
  ads:
    provider: adsense
    enabled: true
    mode: auto
    client: {client}
""",
        encoding="utf-8",
    )


def write_mathjax_entrypoint(root: Path) -> None:
    entrypoint = root / "vendor" / "mathjax" / "es5" / "tex-mml-chtml.js"
    entrypoint.parent.mkdir(parents=True)
    entrypoint.write_text("window.MathJax = {};", encoding="utf-8")


def make_mathjax_source(root: Path, name: str = "mathjax-src") -> Path:
    source = root / name
    entrypoint = source / "es5" / "tex-mml-chtml.js"
    entrypoint.parent.mkdir(parents=True)
    entrypoint.write_text("source MathJax", encoding="utf-8")
    (source / "README.md").write_text("mathjax source", encoding="utf-8")
    return source


def make_mathjax_tarball(path: Path, include_entrypoint: bool = True) -> Path:
    with tarfile.open(path, "w:gz") as archive:
        if include_entrypoint:
            data = b"downloaded MathJax"
            info = tarfile.TarInfo("package/es5/tex-mml-chtml.js")
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
        readme = b"mathjax package"
        readme_info = tarfile.TarInfo("package/README.md")
        readme_info.size = len(readme)
        archive.addfile(readme_info, io.BytesIO(readme))
    return path


def make_unsafe_tarball(path: Path, member_name: str, link_type=None) -> Path:
    with tarfile.open(path, "w:gz") as archive:
        info = tarfile.TarInfo(member_name)
        if link_type:
            info.type = link_type
            info.linkname = "package/es5/tex-mml-chtml.js"
            archive.addfile(info)
        else:
            data = b"bad"
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
    return path


def mock_download(monkeypatch: pytest.MonkeyPatch, tarball: Path, calls: list[str]) -> None:
    def fake_download(url: str, destination: Path, timeout: int = 30) -> None:
        calls.append(url)
        shutil.copy2(tarball, destination)

    monkeypatch.setattr("kiln.core.download_tarball", fake_download)


def request_mathjax(root: Path, page_path: str = "index.yml") -> None:
    (root / "contents" / page_path).parent.mkdir(parents=True, exist_ok=True)
    (root / "contents" / page_path).write_text(
        """page:
  title: Home
  path: /
  layout: default
packages:
  - mathjax
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )


def zip_names(path: Path) -> set:
    with zipfile.ZipFile(path) as archive:
        return set(archive.namelist())


def run_help(args: list) -> int:
    with patch("sys.stdout"):
        try:
            main(args)
        except SystemExit as err:
            return err.code
    return 0


def test_init_creates_right_files(tmp_path: Path) -> None:
    site = tmp_path / "site"

    assert main(["init", str(site)]) == 0

    assert (site / "site.yml").is_file()
    assert (site / "contents" / "index.yml").is_file()
    assert (site / "templates" / "default.html").is_file()
    assert (site / "static").is_dir()
    assert (site / "public").is_dir()


def test_init_creates_mathjax_vendor_readme(tmp_path: Path) -> None:
    site = tmp_path / "site"

    assert main(["init", str(site)]) == 0

    readme = site / "vendor" / "mathjax" / "README.md"
    assert readme.is_file()
    assert "vendor/mathjax/es5/tex-mml-chtml.js" in readme.read_text(
        encoding="utf-8"
    )


def test_init_refuses_non_empty_directory(tmp_path: Path) -> None:
    site = tmp_path / "site"
    site.mkdir()
    (site / "existing.txt").write_text("nope", encoding="utf-8")

    assert main(["init", str(site)]) == 1


def test_validate_catches_bad_yaml(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text("page: [", encoding="utf-8")

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_catches_missing_required_fields(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
content: []
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_catches_external_script_urls(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
content:
  - type: html
    value: '<script src="https://example.com/app.js"></script>'
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_ads_disabled_by_default_emits_no_adsense_script(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "templates" / "default.html").write_text(
        "<html><head></head><body>{{ content }}</body></html>",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert "adsbygoogle.js" not in html


def test_adsense_enabled_emits_script_in_head(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path)
    (tmp_path / "templates" / "default.html").write_text(
        "<html><head>{{ head_integrations_html | safe }}</head><body>{{ content }}</body></html>",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert ADSENSE_SCRIPT in html
    assert html.index(ADSENSE_SCRIPT) < html.index("</head>")


def test_site_ads_enabled_injects_ads_on_normal_pages(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path)

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert ADSENSE_SCRIPT in html


def test_page_ads_disabled_suppresses_ads_for_that_page(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
ads:
  enabled: false
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert ADSENSE_SCRIPT not in html


def test_page_ads_enabled_true_is_redundant_but_valid(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
ads:
  enabled: true
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert ADSENSE_SCRIPT in html


def test_page_ads_enabled_must_be_boolean(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
ads:
  enabled: "false"
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_site_ads_disabled_page_ads_enabled_true_emits_no_ads(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
ads:
  enabled: true
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert ADSENSE_SCRIPT not in html


def test_adsense_older_template_fallback_respects_page_disable(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path)
    (tmp_path / "templates" / "default.html").write_text(
        "<html><head><title>{{ page.title }}</title></head><body>{{ content }}</body></html>",
        encoding="utf-8",
    )
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
ads:
  enabled: false
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert ADSENSE_SCRIPT not in html


def test_adsense_missing_client_fails_validation(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "site.yml").write_text(
        """site:
  title: Test Site
integrations:
  ads:
    provider: adsense
    enabled: true
    mode: auto
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_adsense_invalid_client_fails_validation(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path, client="pub-123")

    assert main(["validate", str(tmp_path)]) == 1


def test_adsense_unknown_provider_fails_validation(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "site.yml").write_text(
        f"""site:
  title: Test Site
integrations:
  ads:
    provider: other
    enabled: true
    mode: auto
    client: {ADSENSE_CLIENT}
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_adsense_unknown_mode_fails_validation(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "site.yml").write_text(
        f"""site:
  title: Test Site
integrations:
  ads:
    provider: adsense
    enabled: true
    mode: manual
    client: {ADSENSE_CLIENT}
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_adsense_exact_generated_script_url_passes_validation(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        f"""page:
  title: Home
  path: /
  layout: default
content:
  - type: html
    value: '<script async src="{ADSENSE_URL}" crossorigin="anonymous"></script>'
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 0


def test_adsense_different_googlesyndication_url_fails_validation(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
content:
  - type: html
    value: '<script src="https://pagead2.googlesyndication.com/pagead/js/other.js"></script>'
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_starter_site_does_not_enable_ads(tmp_path: Path) -> None:
    site = tmp_path / "site"

    assert main(["init", str(site)]) == 0
    assert main(["build", str(site)]) == 0

    assert "integrations:" not in (site / "site.yml").read_text(encoding="utf-8")
    html = (site / "public" / "index.html").read_text(encoding="utf-8")
    assert "adsbygoogle.js" not in html


def test_adsense_injects_into_older_template_before_head_close(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path)
    (tmp_path / "templates" / "default.html").write_text(
        "<html><head><title>{{ page.title }}</title></head><body>{{ content }}</body></html>",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert ADSENSE_SCRIPT in html
    assert html.index(ADSENSE_SCRIPT) < html.index("</head>")


def test_adsense_does_not_duplicate_template_variable_output(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    enable_adsense(tmp_path)
    (tmp_path / "templates" / "default.html").write_text(
        "<html><head>{{ head_integrations_html | safe }}</head><body>{{ content }}</body></html>",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert html.count(ADSENSE_SCRIPT) == 1


def test_validate_catches_protocol_relative_script_urls(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
content:
  - type: html
    value: '<script src="//example.com/app.js"></script>'
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_catches_external_stylesheet_in_generated_html(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "templates" / "default.html").write_text(
        '<html><head><link rel="stylesheet" href="https://example.com/site.css"></head>'
        "<body>{{ content }}</body></html>",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_catches_protocol_relative_stylesheet_urls(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "templates" / "default.html").write_text(
        '<html><head><link rel="stylesheet" href="//example.com/site.css"></head>'
        "<body>{{ content }}</body></html>",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_catches_traversal_page_path(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Evil
  path: /../evil/
  layout: default
content:
  - type: markdown
    value: "# Nope"
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1
    assert main(["build", str(tmp_path)]) == 1
    assert not (tmp_path / "evil" / "index.html").exists()


def test_validate_catches_backslash_page_path(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Bad
  path: /bad\\path/
  layout: default
content:
  - type: markdown
    value: "# Nope"
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_catches_duplicate_page_output_paths(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").unlink()
    (tmp_path / "contents" / "a.yml").write_text(
        """page:
  title: A
  path: /same/
  layout: default
content:
  - type: markdown
    value: "# A"
""",
        encoding="utf-8",
    )
    (tmp_path / "contents" / "b.yml").write_text(
        """page:
  title: B
  path: /same/
  layout: default
content:
  - type: markdown
    value: "# B"
""",
        encoding="utf-8",
    )

    with pytest.raises(KilnError) as err:
        validate_site(tmp_path)

    message = str(err.value)
    assert "duplicate page.path '/same/'" in message
    assert "a.yml" in message
    assert "b.yml" in message
    assert main(["build", str(tmp_path)]) == 1
    assert not (tmp_path / "public" / "same" / "index.html").exists()


def test_validate_catches_duplicate_index_yml_and_yaml(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yaml").write_text(
        """page:
  title: Home YAML
  path: /
  layout: default
content:
  - type: markdown
    value: "# YAML"
""",
        encoding="utf-8",
    )

    with pytest.raises(KilnError) as err:
        validate_site(tmp_path)

    message = str(err.value)
    assert "duplicate page.path '/'" in message
    assert "index.yml" in message
    assert "index.yaml" in message
    assert main(["build", str(tmp_path)]) == 1


def test_validate_catches_missing_value(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
content:
  - type: markdown
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_value_payloads_render_correctly(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
content:
  - type: markdown
    value: "## Markdown Value"
  - type: html
    value: "<strong>HTML Value</strong>"
  - type: code
    language: python
    value: "print('code value')"
  - type: math
    value: "x^2"
  - type: image
    src: /img/logo.png
    alt: Logo
""",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0
    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert "<h2>Markdown Value</h2>" in html
    assert "<strong>HTML Value</strong>" in html
    assert 'class="language-python"' in html
    assert "print(&#x27;code value&#x27;)" in html
    assert '<div class="math">x^2</div>' in html
    assert '<img src="/img/logo.png" alt="Logo">' in html


def test_yaml_content_files_are_loaded(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").unlink()
    (tmp_path / "contents" / "index.yaml").write_text(
        """page:
  title: Home
  path: /
  layout: default
content:
  - type: markdown
    value: "# YAML Works"
""",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0
    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert "<h1>YAML Works</h1>" in html


def test_validate_catches_bad_allowed_package_type(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "site.yml").write_text(
        """site:
  title: Test Site
assets:
  allowed_js_packages: mathjax
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_catches_unknown_allowed_package(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "site.yml").write_text(
        """site:
  title: Test Site
assets:
  allowed_js_packages:
    - chartcdn
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_fails_when_mathjax_vendor_entrypoint_missing(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    allow_mathjax(tmp_path)
    request_mathjax(tmp_path)

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_passes_when_mathjax_vendor_entrypoint_exists(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    allow_mathjax(tmp_path)
    request_mathjax(tmp_path)
    write_mathjax_entrypoint(tmp_path)

    assert main(["validate", str(tmp_path)]) == 0


def test_validate_rejects_unknown_page_package_name(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
packages:
  - chartcdn
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_rejects_non_string_page_package_name(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
packages:
  - 123
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def test_validate_rejects_packages_that_are_not_a_list(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
packages: mathjax
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )

    assert main(["validate", str(tmp_path)]) == 1


def write_site_with_build_dir(root: Path, key: str, value: str) -> None:
    write_minimal_site(root)
    (root / "site.yml").write_text(
        f"""site:
  title: Test Site
build:
  {key}: {value}
""",
        encoding="utf-8",
    )


def test_validate_rejects_output_dir_parent_traversal(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "output_dir", "../outside")

    assert main(["validate", str(tmp_path)]) == 1
    assert main(["build", str(tmp_path)]) == 1


def test_validate_rejects_output_dir_absolute_path(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "output_dir", "/tmp/outside")

    assert main(["validate", str(tmp_path)]) == 1
    assert main(["build", str(tmp_path)]) == 1


def test_validate_rejects_content_dir_parent_traversal(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "content_dir", "../contents")

    assert main(["validate", str(tmp_path)]) == 1
    assert main(["build", str(tmp_path)]) == 1


def test_validate_rejects_template_dir_parent_traversal(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "template_dir", "../templates")

    assert main(["validate", str(tmp_path)]) == 1
    assert main(["build", str(tmp_path)]) == 1


def test_validate_rejects_static_dir_parent_traversal(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "static_dir", "../static")

    assert main(["validate", str(tmp_path)]) == 1
    assert main(["build", str(tmp_path)]) == 1


def test_validate_rejects_build_dir_backslashes(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "output_dir", "bad\\dist")

    assert main(["validate", str(tmp_path)]) == 1
    assert main(["build", str(tmp_path)]) == 1


def test_build_uses_configured_directories(tmp_path: Path) -> None:
    (tmp_path / "src_pages").mkdir()
    (tmp_path / "theme_templates").mkdir()
    (tmp_path / "assets").mkdir()
    (tmp_path / "dist").mkdir()
    (tmp_path / "site.yml").write_text(
        """site:
  title: Test Site
build:
  output_dir: dist
  content_dir: src_pages
  static_dir: assets
  template_dir: theme_templates
""",
        encoding="utf-8",
    )
    (tmp_path / "theme_templates" / "default.html").write_text(
        "<html><body>{{ content }}</body></html>", encoding="utf-8"
    )
    (tmp_path / "src_pages" / "index.yaml").write_text(
        """page:
  title: Home
  path: /
  layout: default
content:
  - type: markdown
    value: "# Custom Dirs"
""",
        encoding="utf-8",
    )
    (tmp_path / "assets" / "style.css").write_text("body {}", encoding="utf-8")

    assert main(["build", str(tmp_path)]) == 0
    assert "<h1>Custom Dirs</h1>" in (tmp_path / "dist" / "index.html").read_text(
        encoding="utf-8"
    )
    assert (tmp_path / "dist" / "style.css").read_text(encoding="utf-8") == "body {}"


def test_build_creates_public_index_html(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert "<h1>Hello</h1>" in html


def test_build_copies_static_assets(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "static" / "style.css").write_text("body {}", encoding="utf-8")

    assert main(["build", str(tmp_path)]) == 0

    assert (tmp_path / "public" / "style.css").read_text(encoding="utf-8") == "body {}"


def test_build_copies_mathjax_vendor_into_output(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    allow_mathjax(tmp_path)
    request_mathjax(tmp_path)
    write_mathjax_entrypoint(tmp_path)

    assert main(["build", str(tmp_path)]) == 0

    output_entrypoint = (
        tmp_path / "public" / "vendor" / "mathjax" / "es5" / "tex-mml-chtml.js"
    )
    assert output_entrypoint.read_text(encoding="utf-8") == "window.MathJax = {};"


def test_build_injects_mathjax_only_on_requesting_pages(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    allow_mathjax(tmp_path)
    write_mathjax_entrypoint(tmp_path)
    request_mathjax(tmp_path)
    (tmp_path / "contents" / "plain.yml").write_text(
        """page:
  title: Plain
  path: /plain/
  layout: default
content:
  - type: markdown
    value: "# Plain"
""",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0

    math_page = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    plain_page = (tmp_path / "public" / "plain" / "index.html").read_text(
        encoding="utf-8"
    )
    assert MATHJAX_SCRIPT in math_page
    assert MATHJAX_SCRIPT not in plain_page


def test_build_emits_local_mathjax_script_tag(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    allow_mathjax(tmp_path)
    request_mathjax(tmp_path)
    write_mathjax_entrypoint(tmp_path)

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert MATHJAX_SCRIPT in html
    assert "https://" not in html
    assert "http://" not in html
    assert 'src="//' not in html


def test_duplicate_mathjax_packages_emit_one_script_tag(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    allow_mathjax(tmp_path)
    write_mathjax_entrypoint(tmp_path)
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
packages:
  - mathjax
  - mathjax
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )

    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert html.count(MATHJAX_SCRIPT) == 1


def test_build_rejects_symlink_in_mathjax_vendor(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    allow_mathjax(tmp_path)
    request_mathjax(tmp_path)
    write_mathjax_entrypoint(tmp_path)
    target = tmp_path / "vendor" / "mathjax" / "real.txt"
    target.write_text("real", encoding="utf-8")
    (tmp_path / "vendor" / "mathjax" / "linked.txt").symlink_to(target)

    assert main(["build", str(tmp_path)]) == 1
    assert not (tmp_path / "public" / "vendor" / "mathjax" / "linked.txt").exists()


def test_vendor_mathjax_copies_valid_directory(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    source = make_mathjax_source(tmp_path)

    assert main(["vendor", "mathjax", "--from", str(source), str(tmp_path)]) == 0

    assert (
        tmp_path / "vendor" / "mathjax" / "es5" / "tex-mml-chtml.js"
    ).read_text(encoding="utf-8") == "source MathJax"
    assert (tmp_path / "vendor" / "mathjax" / "README.md").is_file()


def test_vendor_mathjax_refuses_missing_source_dir(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)

    assert main(["vendor", "mathjax", "--from", str(tmp_path / "missing"), str(tmp_path)]) == 1


def test_vendor_mathjax_refuses_source_without_entrypoint(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    source = tmp_path.parent / "bad-mathjax"
    source.mkdir()

    assert main(["vendor", "mathjax", "--from", str(source), str(tmp_path)]) == 1


def test_vendor_mathjax_refuses_source_with_symlink(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    source = make_mathjax_source(tmp_path)
    target = source / "real.txt"
    target.write_text("real", encoding="utf-8")
    (source / "linked.txt").symlink_to(target)

    assert main(["vendor", "mathjax", "--from", str(source), str(tmp_path)]) == 1
    assert not (tmp_path / "vendor" / "mathjax" / "linked.txt").exists()


def test_vendor_mathjax_refuses_overwrite_without_force(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    source = make_mathjax_source(tmp_path)
    (tmp_path / "vendor" / "mathjax").mkdir(parents=True)
    (tmp_path / "vendor" / "mathjax" / "existing.txt").write_text("old", encoding="utf-8")

    assert main(["vendor", "mathjax", "--from", str(source), str(tmp_path)]) == 1
    assert (tmp_path / "vendor" / "mathjax" / "existing.txt").read_text(encoding="utf-8") == "old"


def test_vendor_mathjax_overwrites_with_force(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    source = make_mathjax_source(tmp_path)
    (tmp_path / "vendor" / "mathjax").mkdir(parents=True)
    (tmp_path / "vendor" / "mathjax" / "existing.txt").write_text("old", encoding="utf-8")

    assert main(["vendor", "mathjax", "--from", str(source), "--force", str(tmp_path)]) == 0

    assert not (tmp_path / "vendor" / "mathjax" / "existing.txt").exists()
    assert (tmp_path / "vendor" / "mathjax" / "es5" / "tex-mml-chtml.js").is_file()


def test_vendor_mathjax_refuses_copying_from_destination_itself(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    write_mathjax_entrypoint(tmp_path)

    assert main(
        [
            "vendor",
            "mathjax",
            "--from",
            str(tmp_path / "vendor" / "mathjax"),
            "--force",
            str(tmp_path),
        ]
    ) == 1


def test_vendor_mathjax_destination_remains_inside_site_root(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    source = make_mathjax_source(tmp_path)

    assert main(["vendor", "mathjax", "--from", str(source), str(tmp_path)]) == 0

    destination = (tmp_path / "vendor" / "mathjax").resolve(strict=False)
    assert str(destination).startswith(str(tmp_path.resolve(strict=False)))


def test_vendor_mathjax_allows_mathjax_page_to_validate(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    allow_mathjax(tmp_path)
    request_mathjax(tmp_path)
    source = make_mathjax_source(tmp_path)

    assert main(["vendor", "mathjax", "--from", str(source), "--force", str(tmp_path)]) == 0
    assert main(["validate", str(tmp_path)]) == 0


def test_vendor_mathjax_download_installs_mathjax(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    write_minimal_site(tmp_path)
    tarball = make_mathjax_tarball(tmp_path / "mathjax.tgz")
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    assert main(["vendor", "mathjax", "--download", str(tmp_path)]) == 0

    entrypoint = tmp_path / "vendor" / "mathjax" / "es5" / "tex-mml-chtml.js"
    assert entrypoint.read_text(encoding="utf-8") == "downloaded MathJax"


def test_vendor_mathjax_download_replaces_init_placeholder(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    site = tmp_path / "site"
    assert main(["init", str(site)]) == 0
    tarball = make_mathjax_tarball(tmp_path / "mathjax.tgz")
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    assert main(["vendor", "mathjax", "--download", str(site)]) == 0

    assert (site / "vendor" / "mathjax" / "README.md").read_text(
        encoding="utf-8"
    ) == "mathjax package"
    assert (site / "vendor" / "mathjax" / "es5" / "tex-mml-chtml.js").is_file()


def test_vendor_mathjax_download_default_version_is_pinned(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    tarball = make_mathjax_tarball(tmp_path / "mathjax.tgz")
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    assert main(["vendor", "mathjax", "--download", str(tmp_path)]) == 0

    assert calls == ["https://registry.npmjs.org/mathjax/-/mathjax-3.2.2.tgz"]


def test_vendor_mathjax_download_version_changes_requested_version(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    tarball = make_mathjax_tarball(tmp_path / "mathjax.tgz")
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    assert main(["vendor", "mathjax", "--download", "--version", "3.2.1", str(tmp_path)]) == 0

    assert calls == ["https://registry.npmjs.org/mathjax/-/mathjax-3.2.1.tgz"]


def test_vendor_mathjax_requires_exactly_one_source_mode(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    source = make_mathjax_source(tmp_path)

    assert main(["vendor", "mathjax", str(tmp_path)]) == 1
    assert main(["vendor", "mathjax", "--from", str(source), "--download", str(tmp_path)]) == 1


def test_vendor_mathjax_version_without_download_fails(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    source = make_mathjax_source(tmp_path)

    assert main(["vendor", "mathjax", "--from", str(source), "--version", "3.2.2", str(tmp_path)]) == 1


def test_vendor_mathjax_download_refuses_overwrite_without_force(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    tarball = make_mathjax_tarball(tmp_path / "mathjax.tgz")
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)
    destination = tmp_path / "vendor" / "mathjax"
    destination.mkdir(parents=True)
    (destination / "existing.txt").write_text("old", encoding="utf-8")

    assert main(["vendor", "mathjax", "--download", str(tmp_path)]) == 1
    assert (destination / "existing.txt").read_text(encoding="utf-8") == "old"


def test_vendor_mathjax_download_force_overwrites_existing_vendor(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    tarball = make_mathjax_tarball(tmp_path / "mathjax.tgz")
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)
    destination = tmp_path / "vendor" / "mathjax"
    destination.mkdir(parents=True)
    (destination / "existing.txt").write_text("old", encoding="utf-8")

    assert main(["vendor", "mathjax", "--download", "--force", str(tmp_path)]) == 0

    assert not (destination / "existing.txt").exists()
    assert (destination / "es5" / "tex-mml-chtml.js").is_file()


def test_vendor_mathjax_download_missing_entrypoint_fails_clearly(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    tarball = make_mathjax_tarball(tmp_path / "mathjax.tgz", include_entrypoint=False)
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    with pytest.raises(KilnError, match="downloaded package does not look like"):
        download_vendor_package(tmp_path, "mathjax")


def test_vendor_mathjax_download_rejects_absolute_tar_member(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    tarball = make_unsafe_tarball(tmp_path / "mathjax.tgz", "/evil.js")
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    with pytest.raises(KilnError, match="absolute path"):
        download_vendor_package(tmp_path, "mathjax")


def test_vendor_mathjax_download_rejects_parent_traversal_tar_member(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    tarball = make_unsafe_tarball(tmp_path / "mathjax.tgz", "package/../evil.js")
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    with pytest.raises(KilnError, match="contains '..'"):
        download_vendor_package(tmp_path, "mathjax")


def test_vendor_mathjax_download_rejects_tar_symlink(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    tarball = make_unsafe_tarball(
        tmp_path / "mathjax.tgz", "package/linked.js", tarfile.SYMTYPE
    )
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    with pytest.raises(KilnError, match="symlink"):
        download_vendor_package(tmp_path, "mathjax")


def test_vendor_mathjax_download_rejects_tar_hardlink(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    tarball = make_unsafe_tarball(
        tmp_path / "mathjax.tgz", "package/linked.js", tarfile.LNKTYPE
    )
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    with pytest.raises(KilnError, match="hardlink"):
        download_vendor_package(tmp_path, "mathjax")


def test_vendor_mathjax_downloaded_package_allows_mathjax_page_to_validate(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    allow_mathjax(tmp_path)
    request_mathjax(tmp_path)
    tarball = make_mathjax_tarball(tmp_path / "mathjax.tgz")
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    assert main(["vendor", "mathjax", "--download", str(tmp_path)]) == 0
    assert main(["validate", str(tmp_path)]) == 0


def test_vendor_mathjax_download_generates_no_cdn_urls(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_minimal_site(tmp_path)
    allow_mathjax(tmp_path)
    request_mathjax(tmp_path)
    tarball = make_mathjax_tarball(tmp_path / "mathjax.tgz")
    calls: list[str] = []
    mock_download(monkeypatch, tarball, calls)

    assert main(["vendor", "mathjax", "--download", str(tmp_path)]) == 0
    assert main(["build", str(tmp_path)]) == 0

    html = (tmp_path / "public" / "index.html").read_text(encoding="utf-8")
    assert MATHJAX_SCRIPT in html
    assert "https://" not in html
    assert "http://" not in html
    assert 'src="//' not in html


def test_clean_removes_stale_files_from_configured_output_dir(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "output_dir", "dist")
    (tmp_path / "dist").mkdir()
    (tmp_path / "dist" / "stale.txt").write_text("old", encoding="utf-8")

    assert main(["clean", str(tmp_path)]) == 0

    assert (tmp_path / "dist").is_dir()
    assert not (tmp_path / "dist" / "stale.txt").exists()


def test_clean_recreates_output_dir(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "output_dir", "dist")

    assert main(["clean", str(tmp_path)]) == 0

    assert (tmp_path / "dist").is_dir()
    assert list((tmp_path / "dist").iterdir()) == []


def test_clean_refuses_unsafe_output_dir(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "output_dir", "../outside")

    assert main(["clean", str(tmp_path)]) == 1
    assert not (tmp_path.parent / "outside").exists()


def test_build_removes_stale_files_by_default(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "public" / "stale.txt").write_text("old", encoding="utf-8")

    assert main(["build", str(tmp_path)]) == 0

    assert not (tmp_path / "public" / "stale.txt").exists()
    assert (tmp_path / "public" / "index.html").is_file()


def test_build_no_clean_preserves_stale_files(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "public" / "stale.txt").write_text("old", encoding="utf-8")

    assert main(["build", "--no-clean", str(tmp_path)]) == 0

    assert (tmp_path / "public" / "stale.txt").read_text(encoding="utf-8") == "old"
    assert (tmp_path / "public" / "index.html").is_file()


def test_clean_does_not_remove_source_dirs_or_site_config(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "public" / "stale.txt").write_text("old", encoding="utf-8")

    assert main(["clean", str(tmp_path)]) == 0

    assert (tmp_path / "site.yml").is_file()
    assert (tmp_path / "contents" / "index.yml").is_file()
    assert (tmp_path / "templates" / "default.html").is_file()
    assert (tmp_path / "static").is_dir()
    assert not (tmp_path / "public" / "stale.txt").exists()


def test_new_page_creates_about_page(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)

    assert main(["new", "page", "about", str(tmp_path)]) == 0

    page_file = tmp_path / "contents" / "about.yml"
    assert page_file.is_file()
    text = page_file.read_text(encoding="utf-8")
    assert 'title: "About"' in text
    assert 'path: "/about/"' in text
    assert "# About" in text


def test_new_page_creates_nested_page(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)

    assert main(["new", "page", "posts/hello-world", str(tmp_path)]) == 0

    page_file = tmp_path / "contents" / "posts" / "hello-world.yml"
    assert page_file.is_file()
    text = page_file.read_text(encoding="utf-8")
    assert 'title: "Hello World"' in text
    assert 'path: "/posts/hello-world/"' in text


def test_new_page_refuses_overwrite(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)

    assert main(["new", "page", "about", str(tmp_path)]) == 0
    assert main(["new", "page", "about", str(tmp_path)]) == 1


def test_new_page_index_uses_root_path(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "contents" / "index.yml").unlink()

    assert main(["new", "page", "index", str(tmp_path)]) == 0

    text = (tmp_path / "contents" / "index.yml").read_text(encoding="utf-8")
    assert 'title: "Index"' in text
    assert 'path: "/"' in text


def test_new_page_rejects_unsafe_slugs(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    unsafe_slugs = [
        "/about",
        "about/",
        "posts//hello",
        "../evil",
        "posts/../evil",
        "bad\\path",
        "hello world",
        "hello.html",
    ]

    for slug in unsafe_slugs:
        assert main(["new", "page", slug, str(tmp_path)]) == 1


def test_new_page_respects_configured_content_dir(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "src_pages").mkdir()
    (tmp_path / "site.yml").write_text(
        """site:
  title: Test Site
build:
  content_dir: src_pages
""",
        encoding="utf-8",
    )

    assert main(["new", "page", "about", str(tmp_path)]) == 0

    assert (tmp_path / "src_pages" / "about.yml").is_file()
    assert not (tmp_path / "contents" / "about.yml").exists()


def test_new_page_generated_page_validates(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)

    assert main(["new", "page", "about", str(tmp_path)]) == 0
    assert main(["validate", str(tmp_path)]) == 0


def test_package_creates_zip(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)

    assert main(["package", str(tmp_path)]) == 0

    assert (tmp_path / f"{tmp_path.name}.zip").is_file()


def test_package_zip_contains_generated_files_without_output_prefix(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    assert main(["new", "page", "about", str(tmp_path)]) == 0
    (tmp_path / "static" / "css").mkdir()
    (tmp_path / "static" / "css" / "site.css").write_text("body {}", encoding="utf-8")

    archive_path = tmp_path / "site.zip"
    assert main(["package", "-o", str(archive_path), str(tmp_path)]) == 0

    names = zip_names(archive_path)
    assert "index.html" in names
    assert "about/index.html" in names
    assert "css/site.css" in names
    assert "public/index.html" not in names


def test_package_zip_excludes_source_files(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    archive_path = tmp_path / "site.zip"

    assert main(["package", "-o", str(archive_path), str(tmp_path)]) == 0

    names = zip_names(archive_path)
    assert "site.yml" not in names
    assert not any(name.startswith("contents/") for name in names)
    assert not any(name.startswith("templates/") for name in names)
    assert not any(name.startswith("static/") for name in names)
    assert not any(name.startswith("/") for name in names)
    assert not any(".." in Path(name).parts for name in names)


def test_package_runs_build_by_default(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    archive_path = tmp_path / "site.zip"

    assert main(["package", "-o", str(archive_path), str(tmp_path)]) == 0

    assert "index.html" in zip_names(archive_path)


def test_package_no_build_uses_existing_output_dir(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    (tmp_path / "public" / "existing.html").write_text("existing", encoding="utf-8")
    archive_path = tmp_path / "site.zip"

    assert main(["package", "--no-build", "-o", str(archive_path), str(tmp_path)]) == 0

    names = zip_names(archive_path)
    assert "existing.html" in names
    assert "index.html" not in names


def test_package_refuses_existing_zip_without_force(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    archive_path = tmp_path / "site.zip"
    archive_path.write_text("existing", encoding="utf-8")

    assert main(["package", "-o", str(archive_path), str(tmp_path)]) == 1
    assert archive_path.read_text(encoding="utf-8") == "existing"


def test_package_force_overwrites_existing_zip(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    archive_path = tmp_path / "site.zip"
    archive_path.write_text("existing", encoding="utf-8")

    assert main(["package", "--force", "-o", str(archive_path), str(tmp_path)]) == 0

    assert "index.html" in zip_names(archive_path)


def test_package_refuses_output_inside_configured_output_dir(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)

    assert main(["package", "-o", str(tmp_path / "public" / "site.zip"), str(tmp_path)]) == 1
    assert not (tmp_path / "public" / "site.zip").exists()


def test_package_refuses_output_path_that_is_directory(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    output_dir = tmp_path / "archive-dir"
    output_dir.mkdir()

    assert main(["package", "-o", str(output_dir), str(tmp_path)]) == 1
    assert output_dir.is_dir()


def test_package_refuses_output_parent_that_is_file(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)
    parent_file = tmp_path / "not-a-directory"
    parent_file.write_text("nope", encoding="utf-8")

    assert main(["package", "-o", str(parent_file / "site.zip"), str(tmp_path)]) == 1
    assert parent_file.read_text(encoding="utf-8") == "nope"


def test_package_respects_configured_output_dir(tmp_path: Path) -> None:
    (tmp_path / "contents").mkdir()
    (tmp_path / "templates").mkdir()
    (tmp_path / "static").mkdir()
    (tmp_path / "dist").mkdir()
    (tmp_path / "site.yml").write_text(
        """site:
  title: Test Site
build:
  output_dir: dist
""",
        encoding="utf-8",
    )
    (tmp_path / "templates" / "default.html").write_text(
        "<html><body>{{ content }}</body></html>", encoding="utf-8"
    )
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
content:
  - type: markdown
    value: "# From Dist"
""",
        encoding="utf-8",
    )
    archive_path = tmp_path / "site.zip"

    assert main(["package", "-o", str(archive_path), str(tmp_path)]) == 0

    assert "index.html" in zip_names(archive_path)
    assert (tmp_path / "dist" / "index.html").is_file()


def test_package_refuses_unsafe_configured_output_dir(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "output_dir", "../outside")
    archive_path = tmp_path / "site.zip"

    assert main(["package", "-o", str(archive_path), str(tmp_path)]) == 1
    assert not archive_path.exists()


def test_end_to_end_smoke_init_new_validate_build_package(tmp_path: Path) -> None:
    site = tmp_path / "smoke-site"
    archive_path = tmp_path / "smoke-site.zip"

    assert main(["init", str(site)]) == 0
    assert main(["new", "page", "about", str(site)]) == 0
    assert main(["validate", str(site)]) == 0
    assert main(["build", str(site)]) == 0
    assert main(["package", "-o", str(archive_path), str(site)]) == 0

    names = zip_names(archive_path)
    assert "index.html" in names
    assert "about/index.html" in names
    assert "site.yml" not in names
    assert not any(name.startswith("contents/") for name in names)
    assert not any(name.startswith("templates/") for name in names)
    assert not any(name.startswith("static/") for name in names)


def test_cli_help_for_required_commands() -> None:
    help_commands = [
        ["init", "-h"],
        ["new", "page", "-h"],
        ["validate", "-h"],
        ["build", "-h"],
        ["clean", "-h"],
        ["serve", "-h"],
        ["package", "-h"],
        ["vendor", "mathjax", "-h"],
    ]

    for args in help_commands:
        assert run_help(args) == 0


def test_cli_exposes_serve_command() -> None:
    with patch("sys.stdout"):
        try:
            main(["serve", "-h"])
        except SystemExit as err:
            assert err.code == 0


def test_serve_calls_validation_before_build(tmp_path: Path) -> None:
    calls = []

    def fake_validate(path: Path) -> list:
        calls.append(("validate", path))
        return []

    def fake_build(path: Path) -> list:
        calls.append(("build", path))
        return []

    write_minimal_site(tmp_path)
    with patch("kiln.cli.validate_site", side_effect=fake_validate), patch(
        "kiln.cli.build_site", side_effect=fake_build
    ), patch("kiln.cli.serve_directory") as serve:
        assert main(["serve", str(tmp_path)]) == 0

    assert calls == [("validate", tmp_path), ("build", tmp_path)]
    serve.assert_called_once()


def test_serve_uses_configured_output_dir(tmp_path: Path) -> None:
    (tmp_path / "contents").mkdir()
    (tmp_path / "templates").mkdir()
    (tmp_path / "static").mkdir()
    (tmp_path / "dist").mkdir()
    (tmp_path / "site.yml").write_text(
        """site:
  title: Test Site
build:
  output_dir: dist
""",
        encoding="utf-8",
    )
    (tmp_path / "templates" / "default.html").write_text(
        "<html><body>{{ content }}</body></html>", encoding="utf-8"
    )
    (tmp_path / "contents" / "index.yml").write_text(
        """page:
  title: Home
  path: /
  layout: default
content:
  - type: markdown
    value: "# Hello"
""",
        encoding="utf-8",
    )

    with patch("kiln.cli.serve_directory") as serve:
        assert main(["serve", str(tmp_path)]) == 0

    serve.assert_called_once_with(tmp_path / "dist", "127.0.0.1", 8000)


def test_serve_refuses_invalid_configured_output_dir(tmp_path: Path) -> None:
    write_site_with_build_dir(tmp_path, "output_dir", "../outside")

    with patch("kiln.cli.serve_directory") as serve:
        assert main(["serve", str(tmp_path)]) == 1

    serve.assert_not_called()


def test_serve_parses_host_and_port(tmp_path: Path) -> None:
    write_minimal_site(tmp_path)

    with patch("kiln.cli.serve_directory") as serve:
        assert main(["serve", "--host", "0.0.0.0", "--port", "8123", str(tmp_path)]) == 0

    serve.assert_called_once_with(tmp_path / "public", "0.0.0.0", 8123)
