from __future__ import annotations

import html
import json
import re
import shutil
import tarfile
import tempfile
import urllib.error
import urllib.parse
import urllib.request
import zipfile
from dataclasses import dataclass
from datetime import date
from html.parser import HTMLParser
from pathlib import Path
from typing import Any, List, Optional, Set, Tuple
from xml.sax.saxutils import escape as xml_escape

import jinja2
import markdown
import yaml
from markupsafe import Markup


KNOWN_BLOCK_TYPES = {"markdown", "html", "image", "code", "math"}
SLUG_SEGMENT_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9-]*$")
ADSENSE_CLIENT_RE = re.compile(r"^ca-pub-\d{10,30}$")
ISO_DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
SITEMAP_CHANGEFREQ_VALUES = {
    "always",
    "hourly",
    "daily",
    "weekly",
    "monthly",
    "yearly",
    "never",
}
RESERVED_PAGE_DATA_KEYS = {
    "page",
    "meta",
    "content",
    "scripts",
    "packages",
    "post",
    "ads",
    "sitemap",
    "integrations",
    "robots",
    "rss",
    "structured_data",
}


class KilnError(Exception):
    """Raised for user-facing Kiln errors."""


@dataclass(frozen=True)
class BuildWarning:
    message: str


@dataclass(frozen=True)
class PageSource:
    source_path: Path
    data: dict[str, Any]


@dataclass(frozen=True)
class SitePaths:
    site_root: Path
    content_dir: Path
    output_dir: Path
    static_dir: Path
    template_dir: Path


@dataclass(frozen=True)
class PackageSpec:
    vendor_dir: Path
    entrypoint: Path
    output_dir: Path
    script_tag: str


@dataclass(frozen=True)
class IntegrationSpec:
    provider: str
    mode: str
    script_template: str


PACKAGE_REGISTRY = {
    "mathjax": PackageSpec(
        vendor_dir=Path("vendor/mathjax"),
        entrypoint=Path("vendor/mathjax/es5/tex-mml-chtml.js"),
        output_dir=Path("vendor/mathjax"),
        script_tag='<script defer src="{base_path}/vendor/mathjax/es5/tex-mml-chtml.js"></script>',
    )
}

INTEGRATION_REGISTRY = {
    "adsense_auto": IntegrationSpec(
        provider="adsense",
        mode="auto",
        script_template=(
            '<script async src="https://pagead2.googlesyndication.com/pagead/js/'
            'adsbygoogle.js?client={client}" crossorigin="anonymous"></script>'
        ),
    )
}

PACKAGE_TARBALL_REGISTRY = {
    "mathjax": "https://registry.npmjs.org/mathjax/-/mathjax-{version}.tgz",
}
DEFAULT_PACKAGE_VERSIONS = {
    "mathjax": "3.2.2",
}
DOWNLOAD_TIMEOUT_SECONDS = 30


class AssetPolicyParser(HTMLParser):
    def __init__(self, allowed_external_script_urls: Optional[Set[str]] = None) -> None:
        super().__init__()
        self.allowed_external_script_urls = allowed_external_script_urls or set()
        self.errors: List[str] = []

    def handle_starttag(self, tag: str, attrs: List[Tuple[str, Optional[str]]]) -> None:
        attr = {key.lower(): value for key, value in attrs if value is not None}
        if tag.lower() == "script":
            src = attr.get("src", "")
            if is_external_url(src) and src not in self.allowed_external_script_urls:
                self.errors.append(f"external script URL is not allowed: {src}")

        if tag.lower() == "link":
            rel = attr.get("rel", "")
            href = attr.get("href", "")
            if "stylesheet" in rel.lower() and is_external_url(href):
                self.errors.append(f"external stylesheet URL is not allowed: {href}")


def init_site(path: Path) -> None:
    if path.exists() and any(path.iterdir()):
        raise KilnError(f"target directory exists and is not empty: {path}")

    path.mkdir(parents=True, exist_ok=True)
    (path / "contents").mkdir()
    (path / "templates").mkdir()
    (path / "static").mkdir()
    (path / "public").mkdir()
    (path / "vendor" / "mathjax").mkdir(parents=True)

    (path / "site.yml").write_text(STARTER_SITE_YML, encoding="utf-8")
    (path / "contents" / "index.yml").write_text(STARTER_INDEX_YML, encoding="utf-8")
    (path / "templates" / "default.html").write_text(
        STARTER_TEMPLATE_HTML, encoding="utf-8"
    )
    (path / "templates" / "post.html").write_text(
        STARTER_POST_TEMPLATE_HTML, encoding="utf-8"
    )
    (path / "vendor" / "mathjax" / "README.md").write_text(
        MATHJAX_VENDOR_README, encoding="utf-8"
    )


def validate_site(path: Path) -> List[BuildWarning]:
    site, site_errors = load_site_config(path)
    if site_errors:
        raise KilnError("\n".join(site_errors))

    paths = site_paths(path, site)
    pages = load_pages(paths.content_dir)
    warnings: List[BuildWarning] = []
    errors: List[str] = []

    allowed_packages = allowed_js_packages(site)

    for page_source in pages:
        page_errors, page_warnings = validate_page(
            paths, site, page_source, allowed_packages
        )
        errors.extend(page_errors)
        warnings.extend(page_warnings)

    errors.extend(validate_unique_output_paths(paths, pages))

    if errors:
        raise KilnError("\n".join(errors))

    return warnings


def build_site(path: Path, clean: bool = True) -> List[BuildWarning]:
    warnings = validate_site(path)
    site, _ = load_site_config(path)
    paths = site_paths(path, site)
    pages = load_pages(paths.content_dir)

    if clean:
        clean_site(path)
    else:
        assert_safe_output_dir(path, paths.output_dir)
        paths.output_dir.mkdir(parents=True, exist_ok=True)

    copy_static(paths.static_dir, paths.output_dir)
    copy_requested_packages(paths, pages)

    for page_source in pages:
        html_text = render_page_html(paths, site, page_source, pages)
        policy_errors = find_asset_policy_errors(
            html_text,
            allowed_external_script_urls=integration_allowed_script_urls(site, page_source.data),
        )
        if policy_errors:
            raise KilnError(
                "\n".join(f"{page_source.source_path}: {err}" for err in policy_errors)
            )

        page = page_source.data["page"]
        out_file = output_file_for_page(paths.output_dir, page["path"])
        out_file.parent.mkdir(parents=True, exist_ok=True)
        out_file.write_text(html_text, encoding="utf-8")

    write_sitemap(paths, site, pages)
    write_robots(paths, site)

    return warnings


def clean_site(path: Path) -> None:
    site, errors = load_site_config(path)
    if errors:
        raise KilnError("\n".join(errors))

    paths = site_paths(path, site)
    assert_safe_output_dir(path, paths.output_dir)

    if paths.output_dir.exists():
        shutil.rmtree(paths.output_dir)
    paths.output_dir.mkdir(parents=True, exist_ok=True)


def new_page(path: Path, slug: str) -> Path:
    segments = validate_page_slug(slug)
    site, errors = load_site_config(path)
    if errors:
        raise KilnError("\n".join(errors))

    paths = site_paths(path, site)
    page_file = paths.content_dir.joinpath(*segments).with_suffix(".yml")
    resolved_file = page_file.resolve(strict=False)
    resolved_content_dir = paths.content_dir.resolve(strict=False)
    if not is_relative_to(resolved_file, resolved_content_dir):
        raise KilnError(f"page file resolves outside content directory: {slug}")
    if page_file.exists():
        raise KilnError(f"page already exists: {page_file}")

    title = title_for_slug(segments)
    page_path = path_for_slug(segments)
    page_file.parent.mkdir(parents=True, exist_ok=True)
    page_file.write_text(page_yaml(title, page_path), encoding="utf-8")
    return page_file


def new_post(path: Path, slug: str) -> Path:
    segments = validate_page_slug(slug)
    site, errors = load_site_config(path)
    if errors:
        raise KilnError("\n".join(errors))

    paths = site_paths(path, site)
    page_file = paths.content_dir.joinpath("posts", *segments).with_suffix(".yml")
    resolved_file = page_file.resolve(strict=False)
    resolved_content_dir = paths.content_dir.resolve(strict=False)
    if not is_relative_to(resolved_file, resolved_content_dir):
        raise KilnError(f"post file resolves outside content directory: {slug}")
    if page_file.exists():
        raise KilnError(f"post already exists: {page_file}")

    title = title_for_slug(segments)
    post_path = "/posts/" + "/".join(segments) + "/"
    page_file.parent.mkdir(parents=True, exist_ok=True)
    page_file.write_text(post_yaml(title, post_path, date.today().isoformat()), encoding="utf-8")
    return page_file


def package_site(
    path: Path,
    output: Optional[Path] = None,
    build: bool = True,
    clean: bool = True,
    force: bool = False,
) -> Path:
    validate_site(path)
    site, _ = load_site_config(path)
    paths = site_paths(path, site)
    assert_safe_output_dir(path, paths.output_dir)

    archive_path = output or (path / f"{path.resolve(strict=False).name}.zip")
    assert_safe_package_output(paths.output_dir, archive_path)
    if archive_path.exists() and archive_path.is_dir():
        raise KilnError(f"output archive path is a directory: {archive_path}")
    if archive_path.parent.exists() and not archive_path.parent.is_dir():
        raise KilnError(f"output archive parent is not a directory: {archive_path.parent}")
    if archive_path.exists() and not force:
        raise KilnError(f"output archive already exists: {archive_path}")

    if build:
        build_site(path, clean=clean)
    if not paths.output_dir.exists():
        raise KilnError(f"output directory does not exist: {paths.output_dir}")

    archive_path.parent.mkdir(parents=True, exist_ok=True)
    try:
        with zipfile.ZipFile(archive_path, "w", compression=zipfile.ZIP_DEFLATED) as archive:
            for source in sorted(paths.output_dir.rglob("*")):
                if source.is_dir():
                    continue
                arcname = source.relative_to(paths.output_dir).as_posix()
                if arcname.startswith("/") or ".." in Path(arcname).parts:
                    raise KilnError(f"unsafe archive path: {arcname}")
                archive.write(source, arcname)
    except (OSError, zipfile.BadZipFile) as err:
        raise KilnError(f"could not create output archive {archive_path}: {err}") from err

    return archive_path


def vendor_package(site_root: Path, package_name: str, source_dir: Path, force: bool = False) -> Path:
    if package_name not in PACKAGE_REGISTRY:
        raise KilnError(f"unknown package: {package_name}")

    site, errors = load_site_config(site_root)
    if errors:
        raise KilnError("\n".join(errors))
    site_paths(site_root, site)

    if not source_dir.exists():
        raise KilnError(f"source directory does not exist: {source_dir}")
    if not source_dir.is_dir():
        raise KilnError(f"source path is not a directory: {source_dir}")

    spec = PACKAGE_REGISTRY[package_name]
    relative_entrypoint = spec.entrypoint.relative_to(spec.vendor_dir)
    source_entrypoint = source_dir / relative_entrypoint
    if not source_entrypoint.is_file():
        raise KilnError(
            f"source does not look like {package_name}: missing {relative_entrypoint}"
        )
    reject_symlinks(source_dir, f"source package {package_name}")

    destination = site_root / spec.vendor_dir
    resolved_root = site_root.resolve(strict=False)
    resolved_source = source_dir.resolve(strict=False)
    resolved_destination = destination.resolve(strict=False)
    if not is_relative_to(resolved_destination, resolved_root):
        raise KilnError(f"vendor destination resolves outside site root: {destination}")
    if resolved_source == resolved_destination:
        raise KilnError("source and destination are the same directory")
    if is_relative_to(resolved_source, resolved_destination):
        raise KilnError("source must not be inside the vendor destination")
    if is_relative_to(resolved_destination, resolved_source):
        raise KilnError("destination must not be inside the source directory")
    if destination.exists() and any(destination.iterdir()) and not force:
        if not is_vendor_placeholder_dir(destination):
            raise KilnError(f"vendor destination already exists: {destination}")

    destination.parent.mkdir(parents=True, exist_ok=True)
    if destination.exists():
        shutil.rmtree(destination)
    shutil.copytree(source_dir, destination, symlinks=True)
    return destination


def is_vendor_placeholder_dir(destination: Path) -> bool:
    if not destination.is_dir():
        return False
    entries = list(destination.iterdir())
    return len(entries) == 1 and entries[0].name == "README.md" and entries[0].is_file()


def download_vendor_package(
    site_root: Path,
    package_name: str,
    version: Optional[str] = None,
    force: bool = False,
) -> Path:
    if package_name not in PACKAGE_REGISTRY:
        raise KilnError(f"unknown package: {package_name}")
    if package_name not in PACKAGE_TARBALL_REGISTRY:
        raise KilnError(f"package does not support downloads: {package_name}")

    package_version = version or DEFAULT_PACKAGE_VERSIONS[package_name]
    tarball_url = package_tarball_url(package_name, package_version)

    with tempfile.TemporaryDirectory(prefix=f"kiln-{package_name}-") as temp_name:
        temp_dir = Path(temp_name)
        tarball_path = temp_dir / f"{package_name}.tgz"
        extract_dir = temp_dir / "extract"
        extract_dir.mkdir()

        download_tarball(tarball_url, tarball_path)
        safe_extract_tarball(tarball_path, extract_dir)
        package_dir = extracted_package_dir(extract_dir, package_name)
        return vendor_package(site_root, package_name, package_dir, force=force)


def package_tarball_url(package_name: str, version: str) -> str:
    try:
        template = PACKAGE_TARBALL_REGISTRY[package_name]
    except KeyError as err:
        raise KilnError(f"package does not support downloads: {package_name}") from err
    return template.format(version=version)


def download_tarball(
    url: str,
    destination: Path,
    timeout: int = DOWNLOAD_TIMEOUT_SECONDS,
) -> None:
    try:
        with urllib.request.urlopen(url, timeout=timeout) as response:
            with destination.open("wb") as file:
                shutil.copyfileobj(response, file)
    except (urllib.error.URLError, TimeoutError, OSError) as err:
        raise KilnError(f"could not download package from {url}: {err}") from err


def safe_extract_tarball(tarball_path: Path, destination: Path) -> None:
    resolved_destination = destination.resolve(strict=False)
    try:
        with tarfile.open(tarball_path, "r:gz") as archive:
            members = archive.getmembers()
            for member in members:
                member_path = Path(member.name)
                if member_path.is_absolute():
                    raise KilnError(f"unsafe package archive member uses absolute path: {member.name}")
                if ".." in member_path.parts:
                    raise KilnError(f"unsafe package archive member contains '..': {member.name}")
                if member.issym():
                    raise KilnError(f"unsafe package archive member is a symlink: {member.name}")
                if member.islnk():
                    raise KilnError(f"unsafe package archive member is a hardlink: {member.name}")

                resolved_member = (destination / member.name).resolve(strict=False)
                if not is_relative_to(resolved_member, resolved_destination):
                    raise KilnError(
                        f"unsafe package archive member resolves outside extraction directory: {member.name}"
                    )

            archive.extractall(destination, members)
    except tarfile.TarError as err:
        raise KilnError(f"could not extract package archive: {err}") from err


def extracted_package_dir(extract_dir: Path, package_name: str) -> Path:
    spec = PACKAGE_REGISTRY[package_name]
    relative_entrypoint = spec.entrypoint.relative_to(spec.vendor_dir)
    candidates = [extract_dir / "package", extract_dir]
    candidates.extend(child for child in extract_dir.iterdir() if child.is_dir())

    for candidate in candidates:
        if (candidate / relative_entrypoint).is_file():
            return candidate

    raise KilnError(
        f"downloaded package does not look like a supported {package_name} "
        f"distribution: missing {relative_entrypoint}"
    )


def load_site_config(path: Path) -> Tuple[dict[str, Any], List[str]]:
    site_file = path / "site.yml"
    site = load_yaml_file(site_file)
    if site is None:
        site = {}
    if not isinstance(site, dict):
        raise KilnError(f"{site_file}: site.yml must contain a mapping")

    return site, validate_site_config(path, site)


def load_pages(contents_dir: Path) -> List[PageSource]:
    if not contents_dir.exists():
        raise KilnError(f"missing contents directory: {contents_dir}")

    page_files = sorted(contents_dir.glob("**/*.yml"))
    page_files.extend(sorted(contents_dir.glob("**/*.yaml")))
    return [
        PageSource(page_file, load_yaml_file(page_file))
        for page_file in sorted(page_files)
    ]


def validate_site_config(site_root: Path, site: dict[str, Any]) -> List[str]:
    errors: List[str] = []
    site_section = site.get("site", {})
    if site_section is None:
        site_section = {}
    if not isinstance(site_section, dict):
        errors.append(f"{site_root / 'site.yml'}: site must be a mapping")
        site_section = {}
    base_url = site_section.get("base_url")
    if base_url is not None:
        if not isinstance(base_url, str) or not is_absolute_http_url(base_url):
            errors.append(f"{site_root / 'site.yml'}: site.base_url must be an absolute http:// or https:// URL")
        elif urllib.parse.urlparse(base_url).query or urllib.parse.urlparse(base_url).fragment:
            errors.append(f"{site_root / 'site.yml'}: site.base_url must not include query strings or fragments")

    assets = site.get("assets", {})
    if assets is None:
        assets = {}
    if not isinstance(assets, dict):
        errors.append(f"{site_root / 'site.yml'}: assets must be a mapping")
        assets = {}

    allowed = assets.get("allowed_js_packages", [])
    if allowed is None:
        allowed = []
    if not isinstance(allowed, list) or not all(
        isinstance(package, str) for package in allowed
    ):
        errors.append(
            f"{site_root / 'site.yml'}: assets.allowed_js_packages must be a list of strings"
        )
        allowed = []

    for package in allowed:
        if package not in PACKAGE_REGISTRY:
            errors.append(
                f"{site_root / 'site.yml'}: unknown allowed JS package: {package}"
            )

    errors.extend(validate_integrations_config(site_root, site))
    errors.extend(validate_structured_data_config(site_root, site))
    errors.extend(validate_sitemap_config(site_root, site))
    errors.extend(validate_robots_config(site_root, site))

    build = site.get("build", {})
    if build is None:
        build = {}
    if not isinstance(build, dict):
        errors.append(f"{site_root / 'site.yml'}: build must be a mapping")
        return errors

    for key in ("output_dir", "content_dir", "static_dir", "template_dir"):
        value = build.get(key)
        if value is not None and not isinstance(value, str):
            errors.append(f"{site_root / 'site.yml'}: build.{key} must be a string")
            continue
        if isinstance(value, str):
            try:
                configured_dir = resolve_configured_dir(site_root, value)
                if key == "output_dir":
                    assert_safe_output_dir(site_root, configured_dir)
            except KilnError as err:
                errors.append(f"{site_root / 'site.yml'}: build.{key}: {err}")

    return errors


def validate_structured_data_config(site_root: Path, site: dict[str, Any]) -> List[str]:
    errors: List[str] = []
    config = site.get("structured_data", {})
    if config is None:
        config = {}
    if not isinstance(config, dict):
        return [f"{site_root / 'site.yml'}: structured_data must be a mapping"]

    enabled = config.get("enabled", False)
    if not isinstance(enabled, bool):
        errors.append(f"{site_root / 'site.yml'}: structured_data.enabled must be a boolean")
        enabled = False

    if enabled and not site_base_url(site):
        errors.append(f"{site_root / 'site.yml'}: site.base_url is required when structured_data is enabled")

    website = config.get("website")
    if website is not None:
        if not isinstance(website, dict):
            errors.append(f"{site_root / 'site.yml'}: structured_data.website must be a mapping")
        else:
            unknown = set(website) - {"name"}
            for key in sorted(unknown):
                errors.append(f"{site_root / 'site.yml'}: unknown structured_data.website field: {key}")
            if "name" in website and not isinstance(website["name"], str):
                errors.append(f"{site_root / 'site.yml'}: structured_data.website.name must be a string")

    organization = config.get("organization")
    if organization is not None:
        if not isinstance(organization, dict):
            errors.append(f"{site_root / 'site.yml'}: structured_data.organization must be a mapping")
        else:
            unknown = set(organization) - {"name", "url", "logo"}
            for key in sorted(unknown):
                errors.append(f"{site_root / 'site.yml'}: unknown structured_data.organization field: {key}")
            name = organization.get("name")
            if not isinstance(name, str) or not name:
                errors.append(f"{site_root / 'site.yml'}: structured_data.organization.name is required")
            for key in ("url", "logo"):
                value = organization.get(key)
                if value is not None and (not isinstance(value, str) or not is_absolute_http_url(value)):
                    errors.append(
                        f"{site_root / 'site.yml'}: structured_data.organization.{key} must be an absolute http:// or https:// URL"
                    )

    return errors


def validate_sitemap_config(site_root: Path, site: dict[str, Any]) -> List[str]:
    errors: List[str] = []
    sitemap = site.get("sitemap", {})
    if sitemap is None:
        sitemap = {}
    if not isinstance(sitemap, dict):
        return [f"{site_root / 'site.yml'}: sitemap must be a mapping"]

    enabled = sitemap.get("enabled", False)
    if not isinstance(enabled, bool):
        errors.append(f"{site_root / 'site.yml'}: sitemap.enabled must be a boolean")
        enabled = False

    if enabled:
        base_url = site_base_url(site)
        if not base_url:
            errors.append(f"{site_root / 'site.yml'}: site.base_url is required when sitemap is enabled")
        elif not is_absolute_http_url(base_url):
            errors.append(f"{site_root / 'site.yml'}: site.base_url must be an absolute http:// or https:// URL")

    return errors


def validate_robots_config(site_root: Path, site: dict[str, Any]) -> List[str]:
    errors: List[str] = []
    robots = site.get("robots", {})
    if robots is None:
        robots = {}
    if not isinstance(robots, dict):
        return [f"{site_root / 'site.yml'}: robots must be a mapping"]

    enabled = robots.get("enabled", False)
    allow_all = robots.get("allow_all", True)
    include_sitemap = robots.get("sitemap", True)

    if not isinstance(enabled, bool):
        errors.append(f"{site_root / 'site.yml'}: robots.enabled must be a boolean")
        enabled = False
    if not isinstance(allow_all, bool):
        errors.append(f"{site_root / 'site.yml'}: robots.allow_all must be a boolean")
    if not isinstance(include_sitemap, bool):
        errors.append(f"{site_root / 'site.yml'}: robots.sitemap must be a boolean")
        include_sitemap = False

    if enabled and include_sitemap:
        base_url = site_base_url(site)
        if not base_url:
            errors.append(f"{site_root / 'site.yml'}: site.base_url is required when robots.sitemap is true")
        elif not is_absolute_http_url(base_url):
            errors.append(f"{site_root / 'site.yml'}: site.base_url must be an absolute http:// or https:// URL")

    return errors


def validate_integrations_config(site_root: Path, site: dict[str, Any]) -> List[str]:
    errors: List[str] = []
    integrations = site.get("integrations", {})
    if integrations is None:
        integrations = {}
    if not isinstance(integrations, dict):
        return [f"{site_root / 'site.yml'}: integrations must be a mapping"]

    ads = integrations.get("ads")
    if ads is None:
        return errors
    if not isinstance(ads, dict):
        return [f"{site_root / 'site.yml'}: integrations.ads must be a mapping"]

    enabled = ads.get("enabled", False)
    if not isinstance(enabled, bool):
        errors.append(f"{site_root / 'site.yml'}: integrations.ads.enabled must be a boolean")
        enabled = False

    provider = ads.get("provider")
    mode = ads.get("mode")
    client = ads.get("client")

    if provider is not None and provider != "adsense":
        errors.append(f"{site_root / 'site.yml'}: integrations.ads.provider must be 'adsense'")
    if mode is not None and mode != "auto":
        errors.append(f"{site_root / 'site.yml'}: integrations.ads.mode must be 'auto'")

    if enabled:
        if provider != "adsense":
            errors.append(f"{site_root / 'site.yml'}: integrations.ads.provider is required and must be 'adsense'")
        if mode != "auto":
            errors.append(f"{site_root / 'site.yml'}: integrations.ads.mode is required and must be 'auto'")
        if not isinstance(client, str) or not client:
            errors.append(f"{site_root / 'site.yml'}: integrations.ads.client is required when ads are enabled")
        elif not ADSENSE_CLIENT_RE.match(client):
            errors.append(
                f"{site_root / 'site.yml'}: integrations.ads.client must match ca-pub- followed by 10 to 30 digits"
            )
    elif client is not None and (not isinstance(client, str) or not ADSENSE_CLIENT_RE.match(client)):
        errors.append(
            f"{site_root / 'site.yml'}: integrations.ads.client must match ca-pub- followed by 10 to 30 digits"
        )

    return errors


def site_paths(site_root: Path, site: dict[str, Any]) -> SitePaths:
    build = site.get("build", {})
    if not isinstance(build, dict):
        build = {}
    return SitePaths(
        site_root=site_root,
        content_dir=resolve_configured_dir(
            site_root, config_dir_value(build, "content_dir", "contents")
        ),
        output_dir=resolve_configured_dir(
            site_root, config_dir_value(build, "output_dir", "public")
        ),
        static_dir=resolve_configured_dir(
            site_root, config_dir_value(build, "static_dir", "static")
        ),
        template_dir=resolve_configured_dir(
            site_root, config_dir_value(build, "template_dir", "templates")
        ),
    )


def config_dir_value(build: dict[str, Any], key: str, default: str) -> str:
    value = build.get(key, default)
    return value if isinstance(value, str) else default


def allowed_js_packages(site: dict[str, Any]) -> Set[str]:
    assets = site.get("assets", {})
    if not isinstance(assets, dict):
        return set()
    allowed = assets.get("allowed_js_packages", [])
    if not isinstance(allowed, list):
        return set()
    return {package for package in allowed if isinstance(package, str)}


def package_source_path(paths: SitePaths, package_name: str) -> Path:
    spec = PACKAGE_REGISTRY[package_name]
    source = paths.site_root / spec.vendor_dir
    resolved_source = source.resolve(strict=False)
    resolved_root = paths.site_root.resolve(strict=False)
    if not is_relative_to(resolved_source, resolved_root):
        raise KilnError(f"package source resolves outside site root: {package_name}")
    return source


def package_entrypoint_path(paths: SitePaths, package_name: str) -> Path:
    spec = PACKAGE_REGISTRY[package_name]
    entrypoint = paths.site_root / spec.entrypoint
    resolved_entrypoint = entrypoint.resolve(strict=False)
    resolved_root = paths.site_root.resolve(strict=False)
    if not is_relative_to(resolved_entrypoint, resolved_root):
        raise KilnError(f"package entrypoint resolves outside site root: {package_name}")
    return entrypoint


def package_output_path(paths: SitePaths, package_name: str) -> Path:
    spec = PACKAGE_REGISTRY[package_name]
    target = paths.output_dir / spec.output_dir
    resolved_target = target.resolve(strict=False)
    resolved_output = paths.output_dir.resolve(strict=False)
    if not is_relative_to(resolved_target, resolved_output):
        raise KilnError(f"package output resolves outside output_dir: {package_name}")
    return target


def requested_packages(page_data: dict[str, Any]) -> List[str]:
    packages = page_data.get("packages", [])
    if packages is None:
        return []
    if not isinstance(packages, list):
        return []
    return [package for package in packages if isinstance(package, str)]


def package_script_tags(package_names: List[str], base_path: str = "") -> str:
    seen = set()
    tags = []
    for package_name in package_names:
        if package_name in seen:
            continue
        seen.add(package_name)
        tags.append(PACKAGE_REGISTRY[package_name].script_tag.format(base_path=base_path))
    return "\n".join(tags)


def build_collections(pages: List[PageSource]) -> dict[str, Any]:
    posts = []
    for page_source in pages:
        data = page_source.data
        if not isinstance(data, dict):
            continue
        page = data.get("page")
        post = data.get("post")
        if not isinstance(page, dict) or not isinstance(post, dict):
            continue
        if post.get("draft", False):
            continue
        title = page.get("title")
        path = page.get("path")
        post_date = post.get("date")
        tags = post.get("tags", [])
        if not isinstance(title, str) or not isinstance(path, str) or not isinstance(post_date, str):
            continue
        if not isinstance(tags, list) or not all(isinstance(tag, str) for tag in tags):
            continue
        posts.append(
            {
                "title": title,
                "path": path,
                "url": path,
                "date": post_date,
                "tags": tags,
                "draft": False,
                "meta": data.get("meta", {}),
            }
        )

    posts.sort(key=lambda post: (-date_key(post["date"]), post["title"]))
    return {"posts": posts}


def date_key(value: str) -> int:
    return int(value.replace("-", ""))


def custom_page_data(page_data: dict[str, Any]) -> dict[str, Any]:
    return {
        key: value
        for key, value in page_data.items()
        if key not in RESERVED_PAGE_DATA_KEYS
    }


def ads_integration_config(site: dict[str, Any]) -> Optional[dict[str, Any]]:
    integrations = site.get("integrations", {})
    if not isinstance(integrations, dict):
        return None
    ads = integrations.get("ads", {})
    if not isinstance(ads, dict) or not ads.get("enabled", False):
        return None
    if (
        ads.get("provider") != "adsense"
        or ads.get("mode") != "auto"
        or not isinstance(ads.get("client"), str)
    ):
        return None
    return ads


def page_ads_enabled(page_data: dict[str, Any]) -> bool:
    ads = page_data.get("ads", {})
    if not isinstance(ads, dict):
        return True
    enabled = ads.get("enabled")
    if enabled is None:
        return True
    return bool(enabled)


def head_integrations_html(site: dict[str, Any], page_data: Optional[dict[str, Any]] = None) -> str:
    ads = ads_integration_config(site)
    if not ads or (page_data is not None and not page_ads_enabled(page_data)):
        return ""
    spec = INTEGRATION_REGISTRY["adsense_auto"]
    return spec.script_template.format(client=ads["client"])


def integration_allowed_script_urls(
    site: dict[str, Any],
    page_data: Optional[dict[str, Any]] = None,
) -> Set[str]:
    html_text = head_integrations_html(site, page_data)
    if not html_text:
        return set()
    parser = ScriptSrcParser()
    parser.feed(html_text)
    return parser.script_urls


def structured_data_enabled(site: dict[str, Any]) -> bool:
    config = site.get("structured_data", {})
    if not isinstance(config, dict):
        return False
    return config.get("enabled", False) is True


def structured_data_html(site: dict[str, Any], page_data: dict[str, Any]) -> str:
    if not structured_data_enabled(site):
        return ""
    graph = structured_data_graph(site, page_data)
    json_text = json.dumps(graph, ensure_ascii=False, separators=(",", ":"))
    json_text = json_text.replace("</", "<\\/")
    return f'<script type="application/ld+json">{json_text}</script>'


def structured_data_graph(site: dict[str, Any], page_data: dict[str, Any]) -> dict[str, Any]:
    base_url = site_base_url(site)
    if not base_url:
        raise KilnError("site.base_url is required when structured_data is enabled")

    config = site.get("structured_data", {})
    if not isinstance(config, dict):
        config = {}
    website_config = config.get("website", {})
    if not isinstance(website_config, dict):
        website_config = {}
    organization_config = config.get("organization")
    site_section = site.get("site", {})
    if not isinstance(site_section, dict):
        site_section = {}

    website = {
        "@type": "WebSite",
        "@id": f"{base_url}#website",
        "url": f"{base_url}/",
        "name": website_config.get("name") or site_section.get("title", ""),
    }
    graph: List[dict[str, Any]] = [website]

    has_organization = isinstance(organization_config, dict)
    if has_organization:
        organization = {
            "@type": "Organization",
            "@id": f"{base_url}#organization",
            "name": organization_config.get("name", ""),
        }
        if organization_config.get("url"):
            organization["url"] = organization_config["url"]
        if organization_config.get("logo"):
            organization["logo"] = organization_config["logo"]
        graph.append(organization)

    page = page_data.get("page", {})
    if not isinstance(page, dict):
        page = {}
    meta = page_data.get("meta", {})
    if not isinstance(meta, dict):
        meta = {}
    page_config = page_data.get("structured_data", {})
    if not isinstance(page_config, dict):
        page_config = {}

    page_path = page.get("path", "/")
    page_url = f"{base_url}{page_path}"
    webpage = {
        "@type": page_config.get("type", "WebPage"),
        "@id": f"{page_url}#webpage",
        "url": page_url,
        "name": page_config.get("name") or page.get("title", ""),
        "description": page_config.get("description", meta.get("description", "")),
        "isPartOf": {"@id": f"{base_url}#website"},
    }
    if has_organization:
        webpage["publisher"] = {"@id": f"{base_url}#organization"}
    graph.append(webpage)

    return {"@context": "https://schema.org", "@graph": graph}


def sitemap_enabled(site: dict[str, Any]) -> bool:
    sitemap = site.get("sitemap", {})
    if not isinstance(sitemap, dict):
        return False
    return sitemap.get("enabled", False) is True


def robots_enabled(site: dict[str, Any]) -> bool:
    robots = site.get("robots", {})
    if not isinstance(robots, dict):
        return False
    return robots.get("enabled", False) is True


def site_base_url(site: dict[str, Any]) -> Optional[str]:
    site_section = site.get("site", {})
    if not isinstance(site_section, dict):
        return None
    base_url = site_section.get("base_url")
    if not isinstance(base_url, str) or not base_url:
        return None
    return base_url.rstrip("/")


def is_absolute_http_url(value: str) -> bool:
    parsed = urllib.parse.urlparse(value)
    return parsed.scheme in {"http", "https"} and bool(parsed.netloc)


def site_base_path(site: dict[str, Any]) -> str:
    base_url = site_base_url(site)
    if not base_url:
        return ""
    path = urllib.parse.urlparse(base_url).path.rstrip("/")
    return "" if path == "/" else path


def url_for(site: dict[str, Any], path: str) -> str:
    base_path = site_base_path(site)
    if not path:
        path = "/"
    if not path.startswith("/"):
        path = "/" + path
    if path == "/":
        return base_path + "/" if base_path else "/"
    return base_path + path


def template_site(site: dict[str, Any]) -> dict[str, Any]:
    context_site = dict(site)
    context_site["base_path"] = site_base_path(site)
    return context_site


def page_sitemap_config(page_data: dict[str, Any]) -> dict[str, Any]:
    sitemap = page_data.get("sitemap", {})
    return sitemap if isinstance(sitemap, dict) else {}


def page_in_sitemap(page_data: dict[str, Any]) -> bool:
    sitemap = page_sitemap_config(page_data)
    enabled = sitemap.get("enabled")
    if enabled is None:
        return True
    return bool(enabled)


def format_sitemap_priority(value: Any) -> str:
    return f"{value:g}"


def write_sitemap(paths: SitePaths, site: dict[str, Any], pages: List[PageSource]) -> None:
    if not sitemap_enabled(site):
        return

    base_url = site_base_url(site)
    if not base_url:
        raise KilnError("site.base_url is required when sitemap is enabled")

    entries = []
    for page_source in pages:
        data = page_source.data
        if not isinstance(data, dict) or not page_in_sitemap(data):
            continue
        page = data.get("page", {})
        if not isinstance(page, dict) or not isinstance(page.get("path"), str):
            continue
        sitemap = page_sitemap_config(data)
        entries.append((page["path"], sitemap))

    lines = [
        '<?xml version="1.0" encoding="UTF-8"?>',
        '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
    ]
    for page_path, sitemap in sorted(entries, key=lambda item: item[0]):
        lines.append("  <url>")
        lines.append(f"    <loc>{xml_escape(base_url + page_path)}</loc>")
        changefreq = sitemap.get("changefreq")
        if changefreq is not None:
            lines.append(f"    <changefreq>{xml_escape(str(changefreq))}</changefreq>")
        priority = sitemap.get("priority")
        if priority is not None:
            lines.append(f"    <priority>{format_sitemap_priority(priority)}</priority>")
        lines.append("  </url>")
    lines.append("</urlset>")
    lines.append("")

    (paths.output_dir / "sitemap.xml").write_text("\n".join(lines), encoding="utf-8")


def write_robots(paths: SitePaths, site: dict[str, Any]) -> None:
    if not robots_enabled(site):
        return

    robots = site.get("robots", {})
    if not isinstance(robots, dict):
        robots = {}

    allow_all = robots.get("allow_all", True)
    include_sitemap = robots.get("sitemap", True)
    lines = ["User-agent: *"]
    lines.append("Allow: /" if allow_all else "Disallow: /")

    if include_sitemap:
        base_url = site_base_url(site)
        if not base_url:
            raise KilnError("site.base_url is required when robots.sitemap is true")
        lines.append(f"Sitemap: {base_url}/sitemap.xml")

    lines.append("")
    (paths.output_dir / "robots.txt").write_text("\n".join(lines), encoding="utf-8")


class ScriptSrcParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.script_urls: Set[str] = set()

    def handle_starttag(self, tag: str, attrs: List[Tuple[str, Optional[str]]]) -> None:
        if tag.lower() != "script":
            return
        attr = {key.lower(): value for key, value in attrs if value is not None}
        src = attr.get("src")
        if src:
            self.script_urls.add(src)


def validate_unique_output_paths(paths: SitePaths, pages: List[PageSource]) -> List[str]:
    errors = []
    seen: dict[Path, Tuple[Path, str]] = {}
    for page_source in pages:
        data = page_source.data
        if not isinstance(data, dict):
            continue
        page = data.get("page")
        if not isinstance(page, dict):
            continue
        page_path = page.get("path")
        if not isinstance(page_path, str):
            continue
        try:
            output_file = output_file_for_page(paths.output_dir, page_path)
        except KilnError:
            continue
        resolved_output = output_file.resolve(strict=False)
        if resolved_output in seen:
            first_source, first_page_path = seen[resolved_output]
            errors.append(
                f"duplicate page.path {page_path!r} renders to the same output file "
                f"as {first_page_path!r}: {first_source} and {page_source.source_path}"
            )
        else:
            seen[resolved_output] = (page_source.source_path, page_path)
    return errors


def copy_requested_packages(paths: SitePaths, pages: List[PageSource]) -> None:
    copied = set()
    for page_source in pages:
        for package_name in requested_packages(page_source.data):
            if package_name in copied:
                continue
            source = package_source_path(paths, package_name)
            reject_symlinks(source, f"vendor package {package_name}")
            target = package_output_path(paths, package_name)
            target.parent.mkdir(parents=True, exist_ok=True)
            if target.exists():
                shutil.rmtree(target)
            shutil.copytree(source, target)
            copied.add(package_name)


def reject_symlinks(root: Path, label: str) -> None:
    if root.is_symlink():
        raise KilnError(f"{label} must not contain symlinks: {root}")
    for path in root.rglob("*"):
        if path.is_symlink():
            raise KilnError(f"{label} must not contain symlinks: {path}")


def resolve_configured_dir(site_root: Path, value: str) -> Path:
    candidate = Path(value)
    if value == "":
        raise KilnError("path must not be empty")
    if candidate.is_absolute():
        raise KilnError(f"absolute paths are not allowed: {value}")
    if ".." in candidate.parts:
        raise KilnError(f"path must not contain '..': {value}")
    if "\\" in value:
        raise KilnError(f"path must not contain backslashes: {value}")

    candidate = site_root / candidate
    root = site_root.resolve(strict=False)
    resolved = candidate.resolve(strict=False)
    if not is_relative_to(resolved, root):
        raise KilnError(f"directory resolves outside site root: {value}")
    return candidate


def assert_safe_output_dir(site_root: Path, output_dir: Path) -> None:
    root = site_root.resolve(strict=False)
    output = output_dir.resolve(strict=False)
    if output == root:
        raise KilnError("output_dir must not resolve to the site root")
    if output == Path("/").resolve(strict=False):
        raise KilnError("output_dir must not resolve to filesystem root")
    if not is_relative_to(output, root):
        raise KilnError("output_dir must resolve inside the site root")


def assert_safe_package_output(output_dir: Path, archive_path: Path) -> None:
    output = output_dir.resolve(strict=False)
    archive = archive_path.resolve(strict=False)
    if is_relative_to(archive, output):
        raise KilnError("output archive must not be inside output_dir")


def validate_page_slug(slug: str) -> List[str]:
    if slug == "":
        raise KilnError("slug must not be empty")
    if slug.startswith("/") or slug.endswith("/"):
        raise KilnError(f"slug must not start or end with '/': {slug}")
    if "\\" in slug:
        raise KilnError(f"slug must not contain backslashes: {slug}")
    candidate = Path(slug)
    if candidate.is_absolute():
        raise KilnError(f"slug must not be absolute: {slug}")

    segments = slug.split("/")
    if any(segment == "" for segment in segments):
        raise KilnError(f"slug must not contain empty path components: {slug}")
    if any(segment == ".." for segment in segments):
        raise KilnError(f"slug must not contain '..': {slug}")
    for segment in segments:
        if not SLUG_SEGMENT_RE.match(segment):
            raise KilnError(f"slug contains unsupported characters: {slug}")
    return segments


def title_for_slug(segments: List[str]) -> str:
    last = segments[-1]
    return " ".join(part.capitalize() for part in last.split("-"))


def path_for_slug(segments: List[str]) -> str:
    if segments == ["index"]:
        return "/"
    return "/" + "/".join(segments) + "/"


def page_yaml(title: str, page_path: str) -> str:
    escaped_title = title.replace('"', '\\"')
    escaped_path = page_path.replace('"', '\\"')
    return f"""page:
  title: "{escaped_title}"
  path: "{escaped_path}"
  layout: "default"
meta:
  description: ""
content:
  - type: markdown
    value: |
      # {title}
"""


def post_yaml(title: str, page_path: str, post_date: str) -> str:
    escaped_title = title.replace('"', '\\"')
    escaped_path = page_path.replace('"', '\\"')
    escaped_date = post_date.replace('"', '\\"')
    return f"""page:
  title: "{escaped_title}"
  path: "{escaped_path}"
  layout: "post"
post:
  date: "{escaped_date}"
  draft: false
  tags: []
meta:
  description: ""
content:
  - type: markdown
    value: |
      # {title}
"""


def load_yaml_file(path: Path) -> Any:
    try:
        return yaml.safe_load(path.read_text(encoding="utf-8"))
    except FileNotFoundError as err:
        raise KilnError(f"missing file: {path}") from err
    except yaml.YAMLError as err:
        raise KilnError(f"{path}: invalid YAML: {err}") from err


def validate_page(
    paths: SitePaths,
    site: dict[str, Any],
    page_source: PageSource,
    allowed_packages: Set[str],
) -> Tuple[List[str], List[BuildWarning]]:
    data = page_source.data
    errors: List[str] = []
    warnings: List[BuildWarning] = []

    if not isinstance(data, dict):
        return [f"{page_source.source_path}: page file must contain a mapping"], warnings

    allowed_external_script_urls = integration_allowed_script_urls(site, data)

    page = data.get("page")
    if not isinstance(page, dict):
        errors.append(f"{page_source.source_path}: missing required field: page")
        page = {}

    for field in ("title", "path", "layout"):
        if not page.get(field):
            errors.append(f"{page_source.source_path}: missing required field: page.{field}")
        elif not isinstance(page.get(field), str):
            errors.append(f"{page_source.source_path}: page.{field} must be a string")

    page_path = page.get("path")
    if isinstance(page_path, str) and (
        not page_path.startswith("/") or not page_path.endswith("/")
    ):
        errors.append(
            f"{page_source.source_path}: page.path must start and end with '/': {page_path}"
        )
    if isinstance(page_path, str):
        if ".." in page_path:
            errors.append(f"{page_source.source_path}: page.path must not contain '..': {page_path}")
        if "\\" in page_path:
            errors.append(f"{page_source.source_path}: page.path must not contain backslashes: {page_path}")
        try:
            output_file_for_page(paths.output_dir, page_path)
        except KilnError as err:
            errors.append(f"{page_source.source_path}: {err}")

    content = data.get("content")
    if not isinstance(content, list):
        errors.append(f"{page_source.source_path}: missing required field: content")
        content = []

    ads = data.get("ads")
    if ads is not None:
        if not isinstance(ads, dict):
            errors.append(f"{page_source.source_path}: ads must be a mapping")
        elif "enabled" in ads and not isinstance(ads["enabled"], bool):
            errors.append(f"{page_source.source_path}: ads.enabled must be a boolean")

    post = data.get("post")
    if post is not None:
        if not isinstance(post, dict):
            errors.append(f"{page_source.source_path}: post must be a mapping")
        else:
            post_date = post.get("date")
            if not post_date:
                errors.append(f"{page_source.source_path}: post.date is required")
            elif not isinstance(post_date, str) or not ISO_DATE_RE.match(post_date):
                errors.append(f"{page_source.source_path}: post.date must be an ISO date string: YYYY-MM-DD")
            draft = post.get("draft", False)
            if not isinstance(draft, bool):
                errors.append(f"{page_source.source_path}: post.draft must be a boolean")
            tags = post.get("tags", [])
            if not isinstance(tags, list) or not all(isinstance(tag, str) for tag in tags):
                errors.append(f"{page_source.source_path}: post.tags must be a list of strings")

    sitemap = data.get("sitemap")
    if sitemap is not None:
        if not isinstance(sitemap, dict):
            errors.append(f"{page_source.source_path}: sitemap must be a mapping")
        else:
            if "enabled" in sitemap and not isinstance(sitemap["enabled"], bool):
                errors.append(f"{page_source.source_path}: sitemap.enabled must be a boolean")
            changefreq = sitemap.get("changefreq")
            if changefreq is not None and changefreq not in SITEMAP_CHANGEFREQ_VALUES:
                errors.append(f"{page_source.source_path}: sitemap.changefreq is invalid: {changefreq}")
            priority = sitemap.get("priority")
            if priority is not None:
                if isinstance(priority, bool) or not isinstance(priority, (int, float)):
                    errors.append(f"{page_source.source_path}: sitemap.priority must be numeric")
                elif priority < 0 or priority > 1:
                    errors.append(f"{page_source.source_path}: sitemap.priority must be between 0.0 and 1.0")

    page_structured_data = data.get("structured_data")
    if page_structured_data is not None:
        if not isinstance(page_structured_data, dict):
            errors.append(f"{page_source.source_path}: structured_data must be a mapping")
        else:
            unknown = set(page_structured_data) - {"type", "name", "description"}
            for key in sorted(unknown):
                errors.append(f"{page_source.source_path}: unknown structured_data field: {key}")
            schema_type = page_structured_data.get("type", "WebPage")
            if schema_type != "WebPage":
                errors.append(f"{page_source.source_path}: structured_data.type must be WebPage")
            for key in ("name", "description"):
                value = page_structured_data.get(key)
                if value is not None and not isinstance(value, str):
                    errors.append(f"{page_source.source_path}: structured_data.{key} must be a string")

    for index, block in enumerate(content):
        if not isinstance(block, dict):
            errors.append(f"{page_source.source_path}: content[{index}] must be a mapping")
            continue
        block_type = block.get("type")
        if block_type not in KNOWN_BLOCK_TYPES:
            errors.append(
                f"{page_source.source_path}: unknown content block type at "
                f"content[{index}]: {block_type}"
            )
            continue
        if block_type in {"markdown", "html", "code", "math"} and "value" not in block:
            errors.append(
                f"{page_source.source_path}: content[{index}] missing required field: value"
            )
        if block_type == "image" and "src" not in block:
            errors.append(
                f"{page_source.source_path}: content[{index}] missing required field: src"
            )
        if block_type == "html":
            errors.extend(
                f"{page_source.source_path}: content[{index}]: {err}"
                for err in find_asset_policy_errors(
                    str(block.get("value", "")),
                    allowed_external_script_urls=allowed_external_script_urls,
                )
            )

    packages = data.get("packages", [])
    if packages is None:
        packages = []
    if not isinstance(packages, list):
        errors.append(f"{page_source.source_path}: packages must be a list")
        packages = []

    for package in packages:
        if not isinstance(package, str):
            errors.append(f"{page_source.source_path}: package names must be strings")
            continue
        if package not in PACKAGE_REGISTRY:
            errors.append(f"{page_source.source_path}: unknown package: {package}")
            continue
        if package not in allowed_packages:
            errors.append(
                f"{page_source.source_path}: package is not listed in "
                f"site.yml assets.allowed_js_packages: {package}"
            )
            continue
        entrypoint = package_entrypoint_path(paths, package)
        if not entrypoint.is_file():
            errors.append(
                f"{page_source.source_path}: package {package} requires local vendor "
                f"entrypoint: {entrypoint}"
            )

    layout = page.get("layout")
    if layout:
        template_name = layout if str(layout).endswith(".html") else f"{layout}.html"
        if not (paths.template_dir / template_name).exists():
            errors.append(f"{page_source.source_path}: template not found: {template_name}")

    if not errors:
        html_text = render_page_html(paths, site, page_source, [page_source])
        errors.extend(
            f"{page_source.source_path}: {err}"
            for err in find_asset_policy_errors(
                html_text,
                allowed_external_script_urls=allowed_external_script_urls,
            )
        )

    return errors, warnings


def render_page_html(
    paths: SitePaths,
    site: dict[str, Any],
    page_source: PageSource,
    pages: Optional[List[PageSource]] = None,
) -> str:
    page = page_source.data["page"]
    layout = page["layout"]
    template_name = layout if layout.endswith(".html") else f"{layout}.html"
    env = jinja2.Environment(
        loader=jinja2.FileSystemLoader(paths.template_dir),
        autoescape=jinja2.select_autoescape(["html", "xml"]),
    )
    try:
        template = env.get_template(template_name)
    except jinja2.TemplateNotFound as err:
        raise KilnError(f"{page_source.source_path}: template not found: {template_name}") from err

    rendered_blocks = render_blocks(page_source.data.get("content", []), site)
    content_html = Markup("\n".join(rendered_blocks))
    scripts = package_script_tags(
        requested_packages(page_source.data),
        base_path=site_base_path(site),
    )
    head_html = head_integrations_html(site, page_source.data)
    schema_html = structured_data_html(site, page_source.data)
    collections = build_collections(pages or [page_source])
    html_text = template.render(
        site=template_site(site),
        page=page,
        post=page_source.data.get("post", {}),
        meta=page_source.data.get("meta", {}),
        content=content_html,
        content_html=content_html,
        data=custom_page_data(page_source.data),
        collections=collections,
        url=lambda path: url_for(site, str(path)),
        packages=page_source.data.get("packages", []),
        package_scripts_html=Markup(scripts),
        head_integrations_html=Markup(head_html),
        structured_data_html=Markup(schema_html),
    )
    if head_html and head_html not in html_text:
        head_index = html_text.lower().find("</head>")
        if head_index >= 0:
            html_text = html_text[:head_index] + head_html + "\n" + html_text[head_index:]
        else:
            html_text = head_html + "\n" + html_text
    if schema_html and schema_html not in html_text:
        head_index = html_text.lower().find("</head>")
        if head_index >= 0:
            html_text = html_text[:head_index] + schema_html + "\n" + html_text[head_index:]
        else:
            html_text = schema_html + "\n" + html_text
    if scripts and scripts not in html_text:
        body_index = html_text.lower().rfind("</body>")
        if body_index >= 0:
            return html_text[:body_index] + scripts + "\n" + html_text[body_index:]
        return html_text + "\n" + scripts
    return html_text


def render_blocks(blocks: List[dict[str, Any]], site: Optional[dict[str, Any]] = None) -> List[str]:
    rendered = []
    site = site or {}
    for block in blocks:
        block_type = block["type"]
        if block_type == "markdown":
            html_text = markdown.markdown(str(block.get("value", "")))
            rendered.append(rewrite_markdown_links(html_text, site))
        elif block_type == "html":
            rendered.append(str(block.get("value", "")))
        elif block_type == "image":
            src = html.escape(str(block.get("src", "")), quote=True)
            alt = html.escape(str(block.get("alt", "")), quote=True)
            rendered.append(f'<img src="{src}" alt="{alt}">')
        elif block_type == "code":
            language = html.escape(str(block.get("language", "")), quote=True)
            code = html.escape(str(block.get("value", "")))
            class_attr = f' class="language-{language}"' if language else ""
            rendered.append(f"<pre><code{class_attr}>{code}</code></pre>")
        elif block_type == "math":
            expression = html.escape(str(block.get("value", "")))
            rendered.append(f'<div class="math">{expression}</div>')
    return rendered


def rewrite_markdown_links(html_text: str, site: dict[str, Any]) -> str:
    base_path = site_base_path(site)
    if not base_path:
        return html_text
    parser = MarkdownLinkRewriter(site)
    parser.feed(html_text)
    parser.close()
    return parser.html()


class MarkdownLinkRewriter(HTMLParser):
    def __init__(self, site: dict[str, Any]) -> None:
        super().__init__(convert_charrefs=False)
        self.site = site
        self.parts: List[str] = []

    def handle_starttag(self, tag: str, attrs: List[Tuple[str, Optional[str]]]) -> None:
        self.parts.append(self.format_tag(tag, attrs, closed=False))

    def handle_startendtag(self, tag: str, attrs: List[Tuple[str, Optional[str]]]) -> None:
        self.parts.append(self.format_tag(tag, attrs, closed=True))

    def handle_endtag(self, tag: str) -> None:
        self.parts.append(f"</{tag}>")

    def handle_data(self, data: str) -> None:
        self.parts.append(data)

    def handle_entityref(self, name: str) -> None:
        self.parts.append(f"&{name};")

    def handle_charref(self, name: str) -> None:
        self.parts.append(f"&#{name};")

    def html(self) -> str:
        return "".join(self.parts)

    def format_tag(
        self,
        tag: str,
        attrs: List[Tuple[str, Optional[str]]],
        closed: bool,
    ) -> str:
        rewritten = []
        for name, value in attrs:
            if name in {"href", "src"} and value is not None:
                value = rewrite_internal_url(value, self.site)
            rewritten.append((name, value))
        attr_text = "".join(
            f" {name}" if value is None else f' {name}="{html.escape(value, quote=True)}"'
            for name, value in rewritten
        )
        suffix = " /" if closed else ""
        return f"<{tag}{attr_text}{suffix}>"


def rewrite_internal_url(value: str, site: dict[str, Any]) -> str:
    if not should_rewrite_internal_url(value, site_base_path(site)):
        return value
    return url_for(site, value)


def should_rewrite_internal_url(value: str, base_path: str) -> bool:
    if not base_path or not value:
        return False
    lowered = value.lower()
    if (
        lowered.startswith("http://")
        or lowered.startswith("https://")
        or lowered.startswith("//")
        or lowered.startswith("#")
        or lowered.startswith("mailto:")
        or lowered.startswith("tel:")
    ):
        return False
    if not value.startswith("/"):
        return False
    return value == "/" or not (value == base_path or value.startswith(base_path + "/"))


def copy_static(static_dir: Path, public_dir: Path) -> None:
    if not static_dir.exists():
        return
    for source in static_dir.rglob("*"):
        if source.is_dir():
            continue
        relative = source.relative_to(static_dir)
        target = public_dir / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target)


def output_file_for_page(public_dir: Path, page_path: str) -> Path:
    relative = page_path.strip("/")
    target = public_dir / "index.html" if not relative else public_dir / relative / "index.html"
    resolved_public = public_dir.resolve(strict=False)
    resolved_target = target.resolve(strict=False)
    if not is_relative_to(resolved_target, resolved_public):
        raise KilnError(f"page.path resolves outside output directory: {page_path}")
    return target


def is_external_url(value: str) -> bool:
    lowered = value.strip().lower()
    return (
        lowered.startswith("https://")
        or lowered.startswith("http://")
        or lowered.startswith("//")
    )


def is_relative_to(path: Path, parent: Path) -> bool:
    try:
        path.relative_to(parent)
    except ValueError:
        return False
    return True


def find_asset_policy_errors(
    html_text: str,
    allowed_external_script_urls: Optional[Set[str]] = None,
) -> List[str]:
    parser = AssetPolicyParser(allowed_external_script_urls)
    parser.feed(html_text)
    return parser.errors


STARTER_SITE_YML = """site:
  title: Kiln Starter Site
build:
  output_dir: public
  content_dir: contents
  static_dir: static
  template_dir: templates
assets:
  allowed_js_packages:
    - mathjax
"""


STARTER_INDEX_YML = """page:
  title: Home
  path: /
  layout: default
meta:
  description: A starter Kiln site.
content:
  - type: markdown
    value: |
      # Welcome to Kiln

      This is your starter site.
"""


STARTER_TEMPLATE_HTML = """<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ page.title }} - {{ site.site.title }}</title>
  {% if meta.description %}<meta name="description" content="{{ meta.description }}">{% endif %}
  {{ head_integrations_html | safe }}
  {{ structured_data_html | safe }}
</head>
<body>
  <main>
    {{ content }}
  </main>
  {{ package_scripts_html | safe }}
</body>
</html>
"""


STARTER_POST_TEMPLATE_HTML = """<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ page.title }} - {{ site.site.title }}</title>
  {% if meta.description %}<meta name="description" content="{{ meta.description }}">{% endif %}
  {{ head_integrations_html | safe }}
  {{ structured_data_html | safe }}
</head>
<body>
  <article>
    <h1>{{ page.title }}</h1>
    {% if post.date %}<p><time datetime="{{ post.date }}">{{ post.date }}</time></p>{% endif %}
    {{ content }}
  </article>
  {{ package_scripts_html | safe }}
</body>
</html>
"""


MATHJAX_VENDOR_README = """# MathJax vendor files

Place a local MathJax distribution in this directory, or let Kiln install the
pinned local copy:

```sh
kiln vendor mathjax --download
```

For offline installs, copy an existing local distribution:

```sh
kiln vendor mathjax --from /path/to/mathjax
```

Kiln expects this entrypoint to exist before building pages that request
MathJax:

```text
vendor/mathjax/es5/tex-mml-chtml.js
```

Kiln does not download MathJax during validation or build and does not
generate CDN URLs.
"""
