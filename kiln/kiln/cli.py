from __future__ import annotations

import argparse
import errno
import functools
import http.server
import sys
from pathlib import Path
from socketserver import TCPServer
from typing import List, Optional

from .core import (
    KilnError,
    build_site,
    clean_site,
    download_vendor_package,
    init_site,
    load_site_config,
    new_page,
    new_post,
    package_site,
    site_paths,
    validate_site,
    vendor_package,
)


class ReusableTCPServer(TCPServer):
    allow_reuse_address = True


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="kiln",
        description="Build, validate, serve, and package Kiln static sites.",
    )
    sub = parser.add_subparsers(dest="command", required=True)

    p_init = sub.add_parser("init", help="Create a starter Kiln site.")
    p_init.add_argument("path", nargs="?", default=".", help="Target directory.")

    p_build = sub.add_parser("build", help="Validate and build a Kiln site.")
    p_build.add_argument("path", nargs="?", default=".", help="Site directory.")
    p_build.add_argument(
        "--clean",
        dest="clean",
        action="store_true",
        default=True,
        help="Clean the output directory before building.",
    )
    p_build.add_argument(
        "--no-clean",
        dest="clean",
        action="store_false",
        help="Preserve existing output files before building.",
    )

    p_clean = sub.add_parser("clean", help="Clean a Kiln site's output directory.")
    p_clean.add_argument("path", nargs="?", default=".", help="Site directory.")

    p_package = sub.add_parser(
        "package",
        help="Create a deployable zip archive. Default output: SITE_ROOT/SITE_NAME.zip.",
        description="Validate, optionally build, and zip the generated output directory contents.",
    )
    p_package.add_argument("path", nargs="?", default=".", help="Site directory.")
    p_package.add_argument(
        "--output",
        "-o",
        type=Path,
        help="Output zip path. Defaults to SITE_ROOT/SITE_NAME.zip.",
    )
    p_package.add_argument(
        "--no-build",
        action="store_true",
        help="Skip building before packaging.",
    )
    p_package.add_argument(
        "--clean",
        dest="clean",
        action="store_true",
        default=True,
        help="Clean the output directory before building.",
    )
    p_package.add_argument(
        "--no-clean",
        dest="clean",
        action="store_false",
        help="Preserve existing output files when building.",
    )
    p_package.add_argument(
        "--force",
        action="store_true",
        help="Overwrite an existing output archive.",
    )

    p_new = sub.add_parser("new", help="Create new Kiln source files.")
    new_sub = p_new.add_subparsers(dest="new_command", required=True)
    p_new_page = new_sub.add_parser("page", help="Create a new page.")
    p_new_page.add_argument("slug", help="Page slug, such as about or posts/hello-world.")
    p_new_page.add_argument("path", nargs="?", default=".", help="Site directory.")
    p_new_post = new_sub.add_parser("post", help="Create a new post.")
    p_new_post.add_argument("slug", help="Post slug, such as hello-world or notes/hello-world.")
    p_new_post.add_argument("path", nargs="?", default=".", help="Site directory.")

    p_vendor = sub.add_parser("vendor", help="Install vendor packages into a site.")
    vendor_sub = p_vendor.add_subparsers(dest="vendor_package", required=True)
    p_vendor_mathjax = vendor_sub.add_parser(
        "mathjax",
        help="Vendor MathJax from a local directory or pinned download.",
    )
    p_vendor_mathjax.add_argument(
        "--from",
        dest="source",
        type=Path,
        help="Local MathJax distribution directory.",
    )
    p_vendor_mathjax.add_argument(
        "--download",
        action="store_true",
        help="Download the pinned MathJax package tarball and vendor it locally.",
    )
    p_vendor_mathjax.add_argument(
        "--version",
        help="MathJax version to download. Only valid with --download. Default: 3.2.2.",
    )
    p_vendor_mathjax.add_argument("--force", action="store_true", help="Overwrite existing vendor files.")
    p_vendor_mathjax.add_argument("path", nargs="?", default=".", help="Site directory.")

    p_validate = sub.add_parser("validate", help="Validate a Kiln site without building.")
    p_validate.add_argument("path", nargs="?", default=".", help="Site directory.")

    p_serve = sub.add_parser("serve", help="Build and serve only the output directory locally.")
    p_serve.add_argument("path", nargs="?", default=".", help="Site directory.")
    p_serve.add_argument("--host", default="127.0.0.1", help="Host to bind.")
    p_serve.add_argument("--port", type=int, default=8000, help="Port to bind.")

    return parser


def serve_directory(directory: Path, host: str, port: int) -> None:
    handler = functools.partial(
        http.server.SimpleHTTPRequestHandler,
        directory=str(directory),
    )
    try:
        with ReusableTCPServer((host, port), handler) as server:
            try:
                server.serve_forever()
            except KeyboardInterrupt:
                print("Stopped server.")
            finally:
                server.server_close()
    except OSError as err:
        if err.errno == errno.EADDRINUSE:
            raise KilnError(f"could not bind {host}:{port}; address already in use") from err
        raise KilnError(f"could not bind {host}:{port}; {err}") from err


def main(argv: Optional[List[str]] = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    path = Path(args.path)

    try:
        if args.command == "init":
            init_site(path)
            print(f"Initialized Kiln site at {path}")
            return 0

        if args.command == "validate":
            warnings = validate_site(path)
            for warning in warnings:
                print(f"warning: {warning.message}", file=sys.stderr)
            print("Site is valid.")
            return 0

        if args.command == "build":
            warnings = build_site(path, clean=args.clean)
            for warning in warnings:
                print(f"warning: {warning.message}", file=sys.stderr)
            site, _ = load_site_config(path)
            print(f"Built site into {site_paths(path, site).output_dir}")
            return 0

        if args.command == "clean":
            clean_site(path)
            site, _ = load_site_config(path)
            print(f"Cleaned {site_paths(path, site).output_dir}")
            return 0

        if args.command == "package":
            archive_path = package_site(
                path,
                output=args.output,
                build=not args.no_build,
                clean=args.clean,
                force=args.force,
            )
            print(f"Packaged site into {archive_path}")
            return 0

        if args.command == "new":
            if args.new_command == "page":
                page_file = new_page(path, args.slug)
                print(f"Created {page_file}")
                return 0
            if args.new_command == "post":
                page_file = new_post(path, args.slug)
                print(f"Created {page_file}")
                return 0

        if args.command == "vendor":
            if args.vendor_package == "mathjax":
                if bool(args.source) == bool(args.download):
                    raise KilnError("exactly one of --from or --download is required")
                if args.version and not args.download:
                    raise KilnError("--version is only valid with --download")

                if args.download:
                    version = args.version or "3.2.2"
                    print(f"Downloading mathjax {version}...")
                    destination = download_vendor_package(
                        path, "mathjax", version=version, force=args.force
                    )
                    print(f"Vendored mathjax {version} into {destination}")
                    return 0

                destination = vendor_package(path, "mathjax", args.source, force=args.force)
                print(f"Vendored mathjax into {destination}")
                return 0

        if args.command == "serve":
            warnings = validate_site(path)
            for warning in warnings:
                print(f"warning: {warning.message}", file=sys.stderr)
            build_site(path)
            site, _ = load_site_config(path)
            output_dir = site_paths(path, site).output_dir
            print(f"Serving {output_dir} at http://{args.host}:{args.port}/")
            serve_directory(output_dir, args.host, args.port)
            return 0

    except KilnError as err:
        print(f"error: {err}", file=sys.stderr)
        return 1

    parser.error(f"unknown command: {args.command}")
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
