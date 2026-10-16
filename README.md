<div align="center">

# CPAMP Theme Studio

[![Release](https://img.shields.io/github/v/release/Cec1c/cpamp-theme-studio?style=flat-square&label=Release)](https://github.com/Cec1c/cpamp-theme-studio/releases/latest)
[![CI](https://img.shields.io/github/actions/workflow/status/Cec1c/cpamp-theme-studio/ci.yml?branch=main&style=flat-square&label=CI)](https://github.com/Cec1c/cpamp-theme-studio/actions/workflows/ci.yml)
[![CPA](https://img.shields.io/badge/CPA-v7.2.138-2563EB?style=flat-square)](https://github.com/router-for-me/CLIProxyAPI)
[![CPAMP](https://img.shields.io/badge/CPAMP-v1.12.2-0EA5E9?style=flat-square)](https://github.com/seakee/CPA-Manager-Plus)
[![License](https://img.shields.io/badge/License-MIT-16A34A?style=flat-square)](LICENSE)
[![LINUX DO](https://img.shields.io/badge/LINUX.DO-%E5%8F%8B%E9%93%BE-111827?style=flat-square)](https://linux.do/)

Install from the CPA/CPAMP Plugin Store, then open the panel's top-right Theme control to adjust colors, fonts, radii, and layout. Changes preview immediately, preferences stay in your browser, and the studio supports multiple languages.

[简体中文](README.zh-CN.md) ｜ [Quick start](#quick-start) ｜ [Linux bootstrap](docs/BOOTSTRAP.md) ｜ [Deployment guide](docs/DEPLOYMENT.md)

</div>

## What can Theme Studio do?

- **Restyle the panel at any time.** Use CPAMP Blue, nine additional palettes, a custom accent, light/dark modes, fonts, radii, density, layout, and visual effects.
- **Open it from the native Theme control.** The studio connects to the panel's existing top-right entry point.
- **Keep preferences stable.** Every change previews immediately and persists in the browser, with legacy migration and one-click reset.
- **Survive panel updates.** If upstream replaces `management.html`, the watcher restores one marked loader block; disabling the plugin removes only that block.
- **Complete Linux updates safely.** After one bootstrap, marketplace installs, upgrades, and confirmed restarts can restart CPA, accept the new version, or roll back a failed deployment.

## Demo

<table>
  <tr>
    <td align="center">
      <strong>Live palette switching and preview</strong><br>
      <img src="docs/theme-studio-demo.gif" alt="CPAMP Theme Studio palette switching demo" width="714">
    </td>
  </tr>
</table>

## Quick start

### Linux + systemd (recommended)

Run bootstrap once to prepare paths, automatic restarts, and failed-update rollback, then install Theme Studio from the CPAMP Plugin Store. You can also install the plugin directly and restart CPA manually after each install or upgrade.

The example below uses v0.2.2. Download and verify the script, then preview the deployment plan:

```bash
version=0.2.2
base="https://github.com/Cec1c/cpamp-theme-studio/releases/download/v${version}"

curl -fLO "${base}/bootstrap-linux.sh"
curl -fLO "${base}/checksums.txt"
grep ' bootstrap-linux.sh$' checksums.txt | sha256sum -c -
chmod 0755 bootstrap-linux.sh

./bootstrap-linux.sh \
  --bootstrap-version "${version}" \
  --panel-url https://example.com/management.html
```

Review the resolved CPA unit, effective config, plugin directory, and active panel. Then apply the same plan as root:

```bash
sudo ./bootstrap-linux.sh \
  --bootstrap-version "${version}" \
  --apply \
  --panel-url https://example.com/management.html
```

If discovery finds multiple candidates, use `--service`, `--config`, and `--panel-path` to select the instance shown in the preview. Manager Server deployments require the exact public `--panel-url`; supplying it for the Lightweight Panel also lets bootstrap compare the local file with the served page.

After bootstrap:

1. Open CPAMP → Plugins → Plugin Store.
2. Search for `CPAMP Theme Studio`, choose Latest, and install.
3. Wait for CPA to restart, return to the dashboard, and click the existing top-right Theme control.

Bootstrap adds this project's community source to CPA's effective configuration.

### Windows, macOS, or non-systemd

Add the community source to CPA's effective `config.yaml`:

```yaml
plugins:
  enabled: true
  dir: "/absolute/path/to/cpa/plugins"
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

Install from the Plugin Store, then restart the CPA process or service that actually serves the panel. Prefer an absolute `dir`. If CPA needs a proxy to reach GitHub, configure proxy variables in the CPA service environment.

## Store, bootstrap, and broker

| Component | Responsibility |
| --- | --- |
| CPAMP/CPA Plugin Store | Downloads the manifest-pinned platform ZIP, verifies SHA-256, writes the versioned native library, and updates `store.version` |
| `bootstrap-linux.sh` | Downloads and verifies the Release, but extracts and runs only the bootstrap executable; it does not install the Theme Studio `.so` as a plugin |
| bootstrap `--apply` | Backs up config/panel state, merges trusted plugin/store/panel settings, installs the root-owned systemd broker, binds one CPA unit, restarts, and verifies |
| broker | Watches marketplace writes or restart requests, waits for stable files, restarts only the bound CPA unit, verifies PID/service/resource/panel state, and restores the last accepted version on failure |
| Theme Studio plugin | Serves browser resources and maintains one loader block in `management.html`; it does not read model credentials or participate in provider/request routing |

The Plugin Store downloads, verifies, and installs each version. The broker restarts the bound CPA unit, checks the result, and rolls back failures. The broker operates on that service as root. The browser and plugin can only submit a fixed-format request file; the administrator binds the target service during bootstrap.

Run bootstrap once, then use the Plugin Store for later updates.

## Panel support

| Deployment | Support | Requirement |
| --- | --- | --- |
| Original CPAMP Lightweight Panel served by CPA (`:8317`) | **Directly supported** | CPA can write the active `static/management.html` |
| CPAMP Manager Server with external `PANEL_PATH` | **Supported with configuration** | CPA and Manager Server share one writable file, and bootstrap receives the exact public `--panel-url` |
| Manager Server with an embedded-only panel | **Externalized by Linux bootstrap** | Direct Linux/systemd deployment and an active public `--panel-url` |

The CPA Lightweight Panel connects directly. With an empty `panel_path`, the plugin checks explicit environment paths and `static/management.html` near CPA's working directory, executable, or config. It injects only when there is one candidate; set an explicit path if several are found.

For an embedded Manager Server page, Linux bootstrap downloads the active page into an external `PANEL_PATH`, binds CPA and Manager Server to that file, and verifies the served result.

## Compatibility baseline

| Component | Validated version | Status |
| --- | --- | --- |
| CLIProxyAPI / CPA | v7.2.138 | ABI, registration, resource route, reload, and disable lifecycle validated |
| CPA Manager Plus / CPAMP | v1.12.2 | Official `management.html` injection and real-browser behavior validated |
| Windows | amd64 | Real DLL load and browser tests passed |
| Linux | amd64 | Real `.so` load, resources, injection, bootstrap, and failed-update rollback passed |
| Linux | arm64 | Native Release target; review the release workflow before production use |
| macOS | amd64, arm64 | Native CI/Release targets |
| Windows | arm64 | Native Release target; review the release workflow before production use |

When upgrading CPA or CPAMP, use the deployment guide to check the plugin ABI, single-file panel structure, and target deployment mode.

## How it works

1. CPA loads a minimal native bridge through plugin ABI v1.
2. The bridge registers one hidden, read-only loader/font resource route and publishes no CPAMP sidebar menu.
3. The injector adds one marked loader block to a trusted writable `management.html`.
4. The loader mounts Theme Studio in Shadow DOM and applies preferences through CPAMP CSS variables and compatible local theme stores.
5. The watcher restores the loader after a panel overwrite; hot-disable removes it deterministically.

## Development and validation

Requirements: Go 1.26+, a native CGO compiler, and optionally Node.js 24+ for loader syntax checks.

```bash
go test ./...
go vet ./...
node --check assets/loader.js
```

Windows packaging:

```powershell
.\scripts\build.ps1 -Version 0.2.2-dev
.\scripts\package.ps1 -Version 0.2.2-dev
```

Linux/macOS packaging:

```bash
./scripts/build.sh 0.2.2-dev
./scripts/package.sh 0.2.2-dev
```

## Documentation

| Document | Covers |
| --- | --- |
| [Linux bootstrap](docs/BOOTSTRAP.md) | Dry-run, apply, broker, acceptance, diagnostics, and rollback |
| [Deployment guide](docs/DEPLOYMENT.md) | Store install, panel modes, upgrades, fallback, and uninstall |
| [Agent runbook](docs/AGENT_DEPLOYMENT.md) | Safety boundaries and checklist for automated deployment |
| [v0.2.0 release notes](docs/RELEASE_NOTES_v0.2.0.md) | Background of the transactional Linux update chain |
| [Security policy](SECURITY.md) | Vulnerability reporting and security notes |

## License and provenance

[MIT](LICENSE). See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for the CPA ABI, CPAMP origin, YAML dependency, JetBrains Mono license, and palette provenance. This project does not redistribute CPAMP's `management.html` or include AGPL `new-api` code or assets.
