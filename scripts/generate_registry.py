#!/usr/bin/env python3
"""Render a pinned CPA schema-v2 registry from packaged release assets."""

from __future__ import annotations

import argparse
import hashlib
import json
import re
from pathlib import Path
from urllib.parse import urlparse


PLATFORMS = (
    ("windows", "amd64"),
    ("windows", "arm64"),
    ("linux", "amd64"),
    ("linux", "arm64"),
    ("darwin", "amd64"),
    ("darwin", "arm64"),
)
VERSION_RE = re.compile(r"^[0-9][0-9A-Za-z.+-]*$")


def parse_checksums(path: Path) -> dict[str, str]:
    checksums: dict[str, str] = {}
    for line_number, raw_line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        line = raw_line.strip()
        if not line:
            continue
        parts = line.split()
        if len(parts) != 2 or not re.fullmatch(r"[0-9a-fA-F]{64}", parts[0]):
            raise ValueError(f"invalid checksum line {line_number}")
        checksums[parts[1].lstrip("*")] = parts[0].lower()
    return checksums


def generate_registry(template_path: Path, assets_dir: Path, version: str) -> dict:
    version = version.removeprefix("v")
    if not VERSION_RE.fullmatch(version):
        raise ValueError(f"invalid version: {version!r}")

    registry = json.loads(template_path.read_text(encoding="utf-8"))
    plugins = registry.get("plugins", [])
    if len(plugins) != 1:
        raise ValueError("registry template must contain exactly one plugin")

    plugin = plugins[0]
    plugin_id = str(plugin.get("id", "")).strip()
    repository = str(plugin.get("repository", "")).rstrip("/")
    parsed_repository = urlparse(repository)
    if plugin_id != "cpamp-theme-studio":
        raise ValueError(f"unexpected plugin id: {plugin_id!r}")
    if parsed_repository.scheme != "https" or parsed_repository.netloc != "github.com":
        raise ValueError("repository must be an https://github.com URL")

    checksums = parse_checksums(assets_dir / "checksums.txt")
    artifacts = []
    for goos, goarch in PLATFORMS:
        name = f"{plugin_id}_{version}_{goos}_{goarch}.zip"
        archive_path = assets_dir / name
        if not archive_path.is_file():
            raise FileNotFoundError(f"missing release archive: {archive_path}")
        expected = checksums.get(name)
        if expected is None:
            raise ValueError(f"checksums.txt is missing {name}")
        actual = hashlib.sha256(archive_path.read_bytes()).hexdigest()
        if actual != expected:
            raise ValueError(f"sha256 mismatch for {name}: {actual} != {expected}")
        artifacts.append(
            {
                "goos": goos,
                "goarch": goarch,
                "url": f"{repository}/releases/download/v{version}/{name}",
                "sha256": actual,
                "size": archive_path.stat().st_size,
            }
        )

    registry["schema_version"] = 2
    plugin["version"] = version
    plugin.pop("versions", None)
    plugin["install"] = {"type": "direct", "artifacts": artifacts}
    return registry


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--template", type=Path, default=Path("registry.json"))
    parser.add_argument("--assets-dir", type=Path, default=Path("dist"))
    parser.add_argument("--version", required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    registry = generate_registry(args.template, args.assets_dir, args.version)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(
        json.dumps(registry, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )


if __name__ == "__main__":
    main()
