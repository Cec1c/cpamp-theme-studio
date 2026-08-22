# CPAMP Theme Studio

[简体中文](README.zh-CN.md) · [Deployment](docs/DEPLOYMENT.md) · [Agent runbook](docs/AGENT_DEPLOYMENT.md) · [v0.1.5 notes](docs/RELEASE_NOTES_v0.1.5.md)

CPAMP Theme Studio is a frontend theme extension delivered through the CPAMP Plugin Store. It adds a persistent visual theme editor to a writable [CPA Manager Plus](https://github.com/seakee/CPA-Manager-Plus) panel without an upstream pull request or a long-lived CPAMP fork.

The theme, editor, and persistence logic live in the browser-side `assets/loader.js`. Because the current CPAMP store and lifecycle controls reuse CPA's plugin APIs, releases also include a minimal native bridge. The bridge registers a hidden browser-resource route and maintains one marked `<script>` block in `management.html`; it does not participate in model requests, providers, or traffic routing. Panel updates are reinjected automatically, and disabling the plugin removes only its own block.

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
- Six corner-radius levels, three density levels, and selectable CPAMP default, bundled JetBrains Mono, or system sans typography.
- CPAMP's own plugin-host scrollbar treatment inside the isolated Theme Studio drawer, including thin native scrolling in Firefox and a transparent pill track in Chromium.
- Full-width or centered desktop content.
- Full visual effects or a reduced-effects performance mode.
- Immediate preview, local persistence, legacy preference migration, and one-click reset.
- Simplified Chinese, Traditional Chinese, English, and Russian UI.
- Shadow DOM isolation, keyboard focus handling, reduced-motion support, and mobile layout.
- Reuses CPAMP's native top-right Theme control as the single entry point; no floating or sidebar duplicate.
- Adds a confirmed `Restart CPA` control to the installed-plugin row and Plugin Store card, so a hot install or upgrade can finish without leaving the retired watcher alive.
- Idempotent runtime recovery and reinjection after SPA or upstream panel updates.
- Deterministic cleanup on hot disable; normal process-shutdown cleanup is best effort.

## Deployment modes

| Mode | Support | Requirement |
| --- | --- | --- |
| CPAMP Lightweight Panel served by CPA (`:8317`) | Supported | CPA must have a writable `static/management.html` |
| CPAMP Manager Server with external `PANEL_PATH` | Supported with configuration | CPA and Manager Server must share the same writable file; set plugin `panel_path` |
| Manager Server with only its embedded panel | Not persistently injectable | The panel is compiled into another process; use an external `PANEL_PATH` first |

The plugin runs inside CPA. It cannot rewrite a panel embedded inside a separate Manager Server executable or container image.

## Recommended install: CPAMP Plugin Store

The project uses this repository as a community store source, so no CPAMP or CPA upstream PR is required. Merge these fields into the effective CPA `config.yaml`. Prefer an absolute `dir` so a different systemd working directory cannot redirect the installation:

```yaml
plugins:
  enabled: true
  dir: "/absolute/path/to/cpa/plugins"
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

After saving and reloading the configuration:

1. Open CPAMP, then Plugins → Plugin Store.
2. Confirm that the sources include `raw.githubusercontent.com` and search for `CPAMP Theme Studio`.
3. Choose Latest or `0.1.5` and install it. CPAMP/CPA downloads the matching pinned Release archive, verifies the SHA-256 carried by the store registry, writes a versioned library under `<dir>/<goos>/<goarch>/`, and creates the enabled plugin configuration.
4. After an install or upgrade, click `Restart CPA` on either the Theme Studio store card or its Installed Plugins row and confirm. The control waits for a new CPA process and a fresh panel injection before refreshing. If automatic restart is unavailable for the current service manager, restart the effective CPA service manually. Do not rely on hot reload alone: a retired plugin version can keep its panel watcher alive until the process exits.
5. Return to the CPAMP dashboard and click the existing Theme control in the top-right action row. The plugin replaces that control's behavior without adding another floating button.

The hidden read-only resource used by the injected loader and bundled fonts is:

```text
/v0/resource/plugins/cpamp-theme-studio/studio
```

If CPA needs a proxy to reach GitHub, set proxy variables in the service environment. `127.0.0.1:7890` works only when the proxy shares CPA's network namespace. See [Deployment](docs/DEPLOYMENT.md) for store validation, the manual fallback, upgrades, rollback, and uninstall.

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
| `restart_mode` | `auto` | `auto`, `disabled`, `systemd`, or `self-exit`; see the deployment guide before changing it |
| `restart_service` | empty | Optional systemd unit override; accepted only when its `MainPID` is the current CPA PID |
| `restart_request` | internal | One-time value written by the authenticated card control; do not edit it manually |

When `panel_path` is empty, the plugin checks `CPAMP_THEME_PANEL_PATH`, `MANAGEMENT_STATIC_PATH`, and `PANEL_PATH`, followed by `static/management.html` and `management.html` near CPA's working directory and executable.

## Plugin Store contract

This repository is a community source that CPAMP can add directly:

```yaml
plugins:
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

`registry.json` uses CPA schema v2 direct artifacts: every OS/architecture URL, byte size, and SHA-256 is pinned to an immutable GitHub Release. This avoids anonymous GitHub REST API rate limits during marketplace installation. CPA installs the root-level library as `cpamp-theme-studio-v<version>.<ext>`. The release workflow generates six platform archives, `checksums.txt`, and a verified candidate registry from the actual packaged bytes.

## Build from source

Requirements: Go 1.26+, a native C compiler for CGO, and optionally Node.js 24+ for the loader syntax check. CGO shared libraries are built on a native runner for each OS/architecture.

Windows PowerShell:

```powershell
.\scripts\build.ps1 -Version 0.1.5-dev
.\scripts\package.ps1 -Version 0.1.5-dev
```

Linux or macOS:

```bash
./scripts/build.sh 0.1.5-dev
./scripts/package.sh 0.1.5-dev
```

Generated libraries and archives are placed under `dist/` and are not committed.

## How it works

1. CPA loads the native library through plugin ABI v1.
2. The plugin registers one hidden read-only resource route for its loader and fonts; it publishes no CPAMP sidebar menu.
3. The injector finds a trusted writable `management.html` and inserts one unique start/end marker block before `</head>`.
4. The loader mounts a Shadow DOM editor and applies preferences through CPAMP's CSS variables and compatible local theme stores.
5. The restart control writes a random one-time request through CPAMP's authenticated configuration-save UI. On Linux/systemd the plugin validates that the selected unit owns the current CPA PID; explicit `self-exit` requires an external supervisor.
6. The watcher restores the marker after an upstream update. Hot-disable removes it deterministically; process-shutdown cleanup is best effort because CPA may terminate before an asynchronous native-plugin shutdown finishes.

The plugin does not intercept CPA requests, read or proxy credentials, expose arbitrary files, or publish an unauthenticated restart endpoint.

Before stopping CPA for an upgrade, rollback, or uninstall, hot-disable this plugin and wait for the marker block to disappear. If CPA has already exited and the markers remain, restore the panel backup or remove only this plugin's marked block as described in the deployment guide.

## Validation performed

- `go test ./...`, `go test -race ./...`, and `go vet ./...`
- `node --check assets/loader.js`
- Native Windows/amd64 DLL and Linux/amd64 shared-object builds and packages
- CPA v7.2.138 real plugin discovery, registration, hidden-menu contract, and resource responses on Windows/amd64 and Linux/amd64
- CPAMP v1.12.2 official panel checksum and injection
- Five consecutive open/change/close cycles using X, scrim, and Escape; three more cycles after reload and login
- Exact singleton mount/stage/host-control counts, body-scroll restoration, SPA route re-entry, and zero sidebar entries
- JetBrains Mono Regular/SemiBold resource responses and successful browser font loading
- CPAMP-default, JetBrains Mono, and system-sans selection plus reload persistence
- CPAMP plugin-host scrollbar parity in light and dark themes, including a 375×667 no-overflow pass
- Scrollable Theme Studio at 1280×720 and 375×667, including reachable density controls and no persistent horizontal overflow
- Palette persistence across reload and 390×844 responsive layout
- Panel overwrite followed by automatic reinjection
- Hot disable followed by exact restoration of the original CPAMP SHA-256
- Installed-row and store-card restart controls, safe cancellation, one-time request persistence, supervised process replacement, loader recovery, and automatic refresh
- Restart confirmation focus handling, 44 px controls, light/dark danger styling, reduced motion, and a 375×812 layout without horizontal overflow

## License and provenance

The project is MIT-licensed. See [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for the CPA ABI, CPAMP origin, YAML dependency, bundled JetBrains Mono license, and palette provenance. The plugin does not redistribute CPAMP's `management.html` or AGPL-licensed `new-api` code/assets.
