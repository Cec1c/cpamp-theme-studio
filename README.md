# CPAMP Theme Studio

[简体中文](README.zh-CN.md) · [Deployment](docs/DEPLOYMENT.md) · [Agent runbook](docs/AGENT_DEPLOYMENT.md)

CPAMP Theme Studio is a standalone CPA plugin that adds a persistent visual theme editor to a writable [CPA Manager Plus](https://github.com/seakee/CPA-Manager-Plus) panel. It keeps the feature outside CPAMP itself, so no upstream pull request or long-lived CPAMP fork is required.

The plugin registers a browser resource with CPA, injects one marked `<script>` block into `management.html`, and restores that block whenever a panel update replaces the file. Disabling the plugin removes only its own marked block.

## Compatibility status

The initial release baseline is:

| Component | Validated version | Status |
| --- | --- | --- |
| CLIProxyAPI / CPA | v7.2.138 | Plugin ABI, registration, resource route, reload, and disable lifecycle validated |
| CPA Manager Plus / CPAMP | v1.12.2 | Official `management.html` injection and browser behavior validated |
| Windows | amd64 | Real DLL load and browser test passed |
| Linux | amd64 | Real `.so` load, resource, and panel-injection tests passed |
| Linux | arm64 | Native release target; validate on the release workflow before production use |
| macOS | amd64, arm64 | Native CI/release targets |
| Windows | arm64 | Native release target; validate on the release workflow before production use |

Future CPA and CPAMP versions are expected to work while their plugin ABI and single-file panel structure remain compatible, but each new upstream version should be tested before production rollout.

## Features

- Automatic, light, and dark display modes.
- CPAMP Blue plus nine independently designed palettes.
- Custom accent color.
- Six corner-radius levels, three density levels, and three font choices.
- Full-width or centered desktop content.
- Full visual effects or a reduced-effects performance mode.
- Immediate preview, local persistence, legacy preference migration, and one-click reset.
- Simplified Chinese, Traditional Chinese, English, and Russian UI.
- Shadow DOM isolation, keyboard focus handling, reduced-motion support, and mobile layout.
- Idempotent reinjection after upstream panel updates.
- Deterministic cleanup on hot disable; normal process-shutdown cleanup is best effort.

## Deployment modes

| Mode | Support | Requirement |
| --- | --- | --- |
| CPAMP Lightweight Panel served by CPA (`:8317`) | Supported | CPA must have a writable `static/management.html` |
| CPAMP Manager Server with external `PANEL_PATH` | Supported with configuration | CPA and Manager Server must share the same writable file; set plugin `panel_path` |
| Manager Server with only its embedded panel | Not persistently injectable | The panel is compiled into another process; use an external `PANEL_PATH` first |

The plugin runs inside CPA. It cannot rewrite a panel embedded inside a separate Manager Server executable or container image.

## Quick start

1. Download the archive for your OS and architecture from GitHub Releases and verify it against `checksums.txt`.
2. For an upgrade, hot-disable the existing plugin and wait for its markers to disappear. Then stop CPA before replacing a loaded library, especially on Windows.
3. Extract the library into the platform directory under CPA's plugin root:

```text
plugins/
  windows/amd64/cpamp-theme-studio.dll
  windows/arm64/cpamp-theme-studio.dll
  linux/amd64/cpamp-theme-studio.so
  linux/arm64/cpamp-theme-studio.so
  darwin/amd64/cpamp-theme-studio.dylib
  darwin/arm64/cpamp-theme-studio.dylib
```

Only the entry matching the host platform is required.

4. Add this configuration to CPA `config.yaml`:

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    cpamp-theme-studio:
      enabled: true
      priority: 10
      auto_inject: true
      panel_path: ""
      host_config_path: ""
      watch_seconds: 3
```

5. Start CPA and open its CPAMP panel. A Theme Studio launcher appears in the lower-right corner. CPA also publishes the plugin menu resource at:

```text
/v0/resource/plugins/cpamp-theme-studio/studio
```

For a complete production procedure, verification commands, upgrades, rollback, and uninstall steps, see [Deployment](docs/DEPLOYMENT.md).

## CPAMP Lightweight Panel configuration

Theme Studio does not download CPAMP itself. A typical CPA configuration for the CPAMP Lightweight Panel is:

```yaml
remote-management:
  allow-remote: false
  secret-key: "replace-with-a-strong-management-key"
  disable-control-panel: false
  disable-auto-update-panel: false
  panel-github-repository: "https://github.com/seakee/CPA-Manager-Plus"
```

Keep management access bound to localhost or protect it with a trusted reverse proxy/VPN. Never commit a real Management Key.

## Plugin configuration

| Field | Default | Meaning |
| --- | --- | --- |
| `auto_inject` | `true` | Watch and patch a writable panel file |
| `panel_path` | empty | Explicit `management.html` file or directory; relative paths use CPA's working directory |
| `host_config_path` | auto-detected | CPA `config.yaml`, used to notice a hot disable and clean up immediately |
| `watch_seconds` | `3` | Panel check interval, clamped to 1–300 seconds |

When `panel_path` is empty, the plugin checks `CPAMP_THEME_PANEL_PATH`, `MANAGEMENT_STATIC_PATH`, and `PANEL_PATH`, followed by `static/management.html` and `management.html` near CPA's working directory and executable.

## Plugin Store source

After a GitHub Release exists, this repository can be added as a CPA community registry:

```yaml
plugins:
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

CPA expects release archives named `cpamp-theme-studio_<version>_<goos>_<goarch>.zip` plus `checksums.txt`. The included release workflow generates those assets.

## Build from source

Requirements: Go 1.26+, a native C compiler for CGO, and optionally Node.js 24+ for the loader syntax check. CGO shared libraries are built on a native runner for each OS/architecture.

Windows PowerShell:

```powershell
.\scripts\build.ps1 -Version 0.1.0-dev
.\scripts\package.ps1 -Version 0.1.0-dev
```

Linux or macOS:

```bash
./scripts/build.sh 0.1.0-dev
./scripts/package.sh 0.1.0-dev
```

Generated libraries and archives are placed under `dist/` and are not committed.

## How it works

1. CPA loads the native library through plugin ABI v1.
2. The plugin registers one read-only management resource and returns its embedded loader JavaScript.
3. The injector finds a trusted writable `management.html` and inserts one unique start/end marker block before `</head>`.
4. The loader mounts a Shadow DOM editor and applies preferences through CPAMP's CSS variables and compatible local theme stores.
5. The watcher restores the marker after an upstream update. Hot-disable removes it deterministically; process-shutdown cleanup is best effort because CPA may terminate before an asynchronous native-plugin shutdown finishes.

The plugin does not intercept CPA requests, handle process signals, proxy credentials, or expose arbitrary files.

Before stopping CPA for an upgrade, rollback, or uninstall, hot-disable this plugin and wait for the marker block to disappear. If CPA has already exited and the markers remain, restore the panel backup or remove only this plugin's marked block as described in the deployment guide.

## Validation performed

- `go test ./...`, `go test -race ./...`, and `go vet ./...`
- `node --check assets/loader.js`
- Native Windows/amd64 DLL and Linux/amd64 shared-object builds and packages
- CPA v7.2.138 real plugin discovery, registration, and menu/resource responses on Windows/amd64 and Linux/amd64
- CPAMP v1.12.2 official panel checksum and injection
- Browser launch, dialog content, palette change, persistence across reload, and 390×844 responsive layout
- Panel overwrite followed by automatic reinjection
- Hot disable followed by exact restoration of the original CPAMP SHA-256

## License and provenance

The project is MIT-licensed. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for the CPA ABI, CPAMP origin, YAML dependency, and palette provenance. The plugin does not redistribute CPAMP's `management.html` or AGPL-licensed `new-api` code/assets.
