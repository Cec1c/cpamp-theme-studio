import hashlib
import json
import tempfile
import unittest
from pathlib import Path

from generate_registry import PLATFORMS, generate_registry


class GenerateRegistryTest(unittest.TestCase):
    def test_generates_verified_direct_artifacts(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            template = root / "registry.json"
            assets = root / "dist"
            assets.mkdir()
            template.write_text(
                json.dumps(
                    {
                        "schema_version": 1,
                        "plugins": [
                            {
                                "id": "cpamp-theme-studio",
                                "name": "CPAMP Theme Studio",
                                "description": "Theme editor",
                                "author": "Cec1c",
                                "version": "0.0.0",
                                "repository": "https://github.com/Cec1c/cpamp-theme-studio",
                            }
                        ],
                    }
                ),
                encoding="utf-8",
            )

            checksum_lines = []
            for goos, goarch in PLATFORMS:
                name = f"cpamp-theme-studio_0.2.0_{goos}_{goarch}.zip"
                payload = f"{goos}/{goarch}".encode()
                (assets / name).write_bytes(payload)
                checksum_lines.append(f"{hashlib.sha256(payload).hexdigest()}  {name}")
            (assets / "checksums.txt").write_text(
                "\n".join(checksum_lines) + "\n", encoding="utf-8"
            )

            registry = generate_registry(template, assets, "v0.2.0")

            self.assertEqual(registry["schema_version"], 2)
            plugin = registry["plugins"][0]
            self.assertEqual(plugin["version"], "0.2.0")
            self.assertEqual(plugin["install"]["type"], "direct")
            self.assertEqual(len(plugin["install"]["artifacts"]), 6)
            for artifact in plugin["install"]["artifacts"]:
                self.assertGreater(artifact["size"], 0)
                self.assertRegex(artifact["sha256"], r"^[0-9a-f]{64}$")
                self.assertIn("/releases/download/v0.2.0/", artifact["url"])


if __name__ == "__main__":
    unittest.main()
