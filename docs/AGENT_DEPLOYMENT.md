# Agent deployment runbook / Agent 部署手册

This file is designed to be handed directly to a server-side coding agent. It is bilingual where operator decisions matter. The agent must inspect the target host and substitute real paths; it must not guess them.

本文可以直接交给服务器 Agent。Agent 必须先检查目标主机并替换真实路径，不能猜测路径或服务名。

## Copyable task prompt / 可复制任务提示词

```text
Deploy Cec1c/cpamp-theme-studio into the existing CPA installation on this server.

Rules:
1. Work read-only first. Discover CPA_HOME, the effective config.yaml, plugins.dir, OS/architecture, CPA version, CPAMP panel version/source, process/service manager, and whether the panel is CPA lightweight, Manager Server external PANEL_PATH, or embedded-only.
2. Do not display, copy, change, or commit Management Keys, API keys, auth files, or unrelated configuration.
3. Use https://github.com/Cec1c/cpamp-theme-studio as the only plugin source. If downloading, verify checksums.txt before extracting. If no release exists, clone the repository and build natively with its documented script.
4. Do not modify an embedded-only Manager Server panel. Stop and report that external PANEL_PATH is required.
5. Before writes, back up only the effective CPA config, existing cpamp-theme-studio library, and external management.html. Preserve permissions and ownership.
6. Merge the minimal plugin YAML block; do not rewrite unrelated YAML. Use absolute panel_path and host_config_path when service working directories are ambiguous.
7. Before stopping CPA, hot-disable only this plugin and wait for marker cleanup plus a 404 resource response. Process-shutdown cleanup is best effort. Then stop CPA before replacing a loaded library and restart only the in-scope CPA/CPAMP services.
8. Validate plugin loaded + registered logs, plugin listing, studio HTML, loader JavaScript, exactly one panel marker pair, and a real browser launch/persistence check when browser access exists.
9. On any failed acceptance check, disable the plugin, wait for marker cleanup, restore backups, restart the previous version, and report the evidence.
10. Report discovered paths and versions, backup paths, changed files, installed SHA-256, validation results, and anything not tested. Never print secrets.
```

## Required inputs / 必要参数

The agent must resolve and report these before mutation:

```text
CPA_HOME=
CPA_CONFIG=
PLUGINS_DIR=
GOOS=
GOARCH=
CPA_VERSION=
CPAMP_VERSION=
PANEL_MODE=lightweight|manager-external|manager-embedded
PANEL_PATH=
SERVICE_MANAGER=systemd|docker-compose|docker|windows-service|manual|other
DESIRED_VERSION=
DOWNLOAD_PROXY=optional
```

If `PANEL_MODE=manager-embedded`, the deployment is not viable yet. The agent should explain how to provide an external `PANEL_PATH` and stop without patching binaries or container layers.

若 `PANEL_MODE=manager-embedded`，当前部署不可行。Agent 应说明如何改为外置 `PANEL_PATH`，不得修改二进制或容器层。

## Deterministic procedure / 确定性流程

1. Inspect effective process arguments and working directory; do not rely on a similarly named inactive installation.
2. Check the listening address. Use loopback for acceptance testing.
3. Determine the plugin path from effective `plugins.dir`, then append `<goos>/<goarch>`.
4. Determine the panel path from the live service/config and verify it is a regular, writable, non-symlink `.html` file under 64 MiB.
5. Download and verify a release archive, or build on the target host:

```bash
git clone --depth 1 https://github.com/Cec1c/cpamp-theme-studio.git
cd cpamp-theme-studio
go test ./...
node --check assets/loader.js
./scripts/package.sh 0.1.0-dev
```

```powershell
git clone --depth 1 https://github.com/Cec1c/cpamp-theme-studio.git
Set-Location .\cpamp-theme-studio
go test ./...
node --check .\assets\loader.js
.\scripts\package.ps1 -Version 0.1.0-dev
```

6. Create timestamped backups beside an operator-approved backup directory. A backup is not successful until its size and SHA-256 are recorded.
7. Stop CPA, install the one matching dynamic library, minimally merge the configuration below, and start CPA:

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    cpamp-theme-studio:
      enabled: true
      priority: 10
      auto_inject: true
      panel_path: "/absolute/path/to/management.html"
      host_config_path: "/absolute/path/to/config.yaml"
      watch_seconds: 3
```

8. Verify in order:

```text
[ ] CPA process is healthy and bound only where expected
[ ] log contains plugin loaded and plugin registered for cpamp-theme-studio
[ ] authenticated CPA plugin listing says registered=true and effective_enabled=true
[ ] GET /v0/resource/plugins/cpamp-theme-studio/studio returns 200 HTML
[ ] GET .../studio?asset=loader returns 200 JavaScript with nosniff and cache headers
[ ] management.html has exactly one start marker and one end marker
[ ] CPAMP still reaches its login/dashboard and the launcher opens
[ ] one setting change persists after browser reload
```

9. Simulate an upstream update only in a disposable copy or approved maintenance window: replace the panel with the same official version and confirm reinjection within `watch_seconds`.
10. Test disable cleanup: set only this plugin's `enabled` to false, confirm the resource becomes 404 and the panel SHA-256 returns to the pre-injection value, then re-enable if deployment is accepted.

## Failure and rollback / 失败与回滚

Rollback is mandatory if CPA fails to start, registration fails, the resource is not read-only, the panel has duplicate/incomplete markers, the UI no longer logs in, or the panel cannot be restored exactly.

回滚步骤：

1. Disable only `cpamp-theme-studio` and allow cleanup when possible.
2. Stop CPA.
3. Restore the backed-up plugin and config; restore the panel only when marker cleanup did not reproduce its recorded hash.
4. Start the previous deployment and run its health checks.
5. Report the failure without deleting evidence/logs or exposing secrets.

## Final report template / 最终报告模板

```text
Outcome: success | rolled back | blocked
CPA version/path:
CPAMP version/panel mode/path:
Platform:
Plugin source/version/SHA-256:
Backups created:
Files changed:
Services restarted:
Acceptance checks:
Lifecycle checks:
Security checks:
Not tested:
Rollback status:
```
