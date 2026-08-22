# Agent deployment runbook / Agent 部署手册

This file is designed to be handed directly to a server-side coding agent. It is bilingual where operator decisions matter. The agent must inspect the target host and substitute real paths; it must not guess them.

本文可以直接交给服务器 Agent。Agent 必须先检查目标主机并替换真实路径，不能猜测路径或服务名。

## Copyable task prompt / 可复制任务提示词

```text
Deploy Cec1c/cpamp-theme-studio into the existing CPA installation on this server.

Rules:
1. Work read-only first. Discover CPA_HOME, the effective config.yaml, plugins.dir, OS/architecture, CPA version, CPAMP panel version/source, the exact CPA PID and process/service manager, and whether the panel is CPA lightweight, Manager Server external PANEL_PATH, or embedded-only.
2. Do not display, copy, change, or commit Management Keys, API keys, auth files, or unrelated configuration.
3. Use https://github.com/Cec1c/cpamp-theme-studio as the only plugin source. Require a published GitHub Release and install through CPAMP's plugin-store flow. The registry uses pinned schema-v2 direct artifacts; do not replace them with GitHub API release discovery. If no release exists, stop; do not substitute a source build and call it a store deployment.
4. Do not modify an embedded-only Manager Server panel. Stop and report that external PANEL_PATH is required.
5. Before writes, back up only the effective CPA config, existing cpamp-theme-studio library, and external management.html. Preserve permissions and ownership.
6. Merge only plugins.enabled, an absolute plugins.dir, and this repository's registry URL into plugins.store-sources. Do not pre-copy a library or invent a plugin config before testing store discovery.
7. Use CPAMP's Plugin Store UI or the authenticated CPA plugin-store API to discover and install cpamp-theme-studio. Record the returned version and absolute install path. A file copied by hand is not a successful marketplace deployment.
8. After store installation, set absolute panel_path and host_config_path only when automatic discovery is ambiguous. Prefer the plugin card's confirmed Restart CPA control. Keep restart_mode=auto for a direct Linux systemd service; use systemd with restart_service only when MainPID can be proven to match CPA; use self-exit only when an external supervisor is proven to restart CPA; otherwise use disabled and restart manually.
9. Validate plugin loaded + registered logs, plugin listing with zero published menus, studio HTML, loader JavaScript, both JetBrains Mono resources, exactly one panel marker pair, the restart-status read-only resource, and repeated real-browser open/change/close plus reload persistence when browser access exists. Cancel the restart confirmation once before confirming it and prove that cancellation changed neither the CPA process nor config hash.
10. On any failed acceptance check, disable the plugin, wait for marker cleanup, restore backups, restart the previous version, and report the evidence.
11. Report store discovery, install API result/path, registry/release SHA-256, discovered paths and versions, backups, changed files, validation results, and anything not tested. Never print secrets.
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
CPA_PID=
RESTART_MODE=auto|systemd|self-exit|disabled
RESTART_SERVICE=optional; systemd unit whose MainPID must equal CPA_PID
SUPERVISOR_RESTART_POLICY=required before self-exit
DESIRED_VERSION=
DOWNLOAD_PROXY=optional
STORE_SOURCE=https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json
STORE_INSTALL_PATH=resolved from the install response; never guessed
```

If `PANEL_MODE=manager-embedded`, the deployment is not viable yet. The agent should explain how to provide an external `PANEL_PATH` and stop without patching binaries or container layers.

若 `PANEL_MODE=manager-embedded`，当前部署不可行。Agent 应说明如何改为外置 `PANEL_PATH`，不得修改二进制或容器层。

## Deterministic procedure / 确定性流程

1. Inspect effective process arguments and working directory; do not rely on a similarly named inactive installation.
2. Check the listening address. Use loopback for acceptance testing.
3. Resolve effective `plugins.dir` to an absolute path. If it is relative, resolve it against the live process working directory, not the shell's current directory.
4. Determine the panel path from the live service/config and verify it is a regular, writable, non-symlink `.html` file under 64 MiB.
5. Confirm the requested GitHub Release exists and contains the matching platform ZIP plus `checksums.txt`; confirm the custom registry's matching artifact URL, size, and SHA-256 agree with those bytes. If any item is missing or disagrees, stop.
6. Create timestamped backups beside an operator-approved backup directory. A backup is not successful until its size and SHA-256 are recorded.
7. Minimally merge the store bootstrap configuration. Preserve unrelated plugin sources and configurations:

```yaml
plugins:
  enabled: true
  dir: "/absolute/path/to/cpa/plugins"
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

8. Reload the effective configuration, then verify the CPAMP Plugin Store lists `cpamp-theme-studio` from that source as install type `direct`, with the current OS/architecture present and no error for this custom source. An unrelated official-source error must be reported but does not invalidate a healthy custom source.
9. Install the requested version through CPAMP or `POST /v0/management/plugin-store/cpamp-theme-studio/install?source=<resolved-source-id>&version=<version>`. Use the deployment's existing authenticated management path without printing the key. Record the response `path` and confirm it is under the resolved `<plugins.dir>/<goos>/<goarch>/`.
10. If panel discovery is ambiguous, update only this plugin's configuration through CPAMP with `auto_inject: true`, absolute `panel_path`, absolute `host_config_path`, and `watch_seconds: 3`. Do not set `restart_request`; it is an internal one-time value written by the authenticated card control. Select restart behavior from this evidence:

```text
systemd direct service: restart_mode=auto, or systemd + an explicit unit after MainPID validation
Docker/other supervisor: restart_mode=self-exit only after proving that exit code 75 causes that exact CPA process to be restarted
manual/no supervisor/uncertain wrapper: restart_mode=disabled and perform an operator-approved manual restart
```

The card does not expose a restart POST endpoint or read the Management Key. It opens CPAMP's native Edit configuration drawer and saves a random `restart_request` through the existing authenticated session.
11. Verify in order:

```text
[ ] CPAMP store lists the plugin from the configured community source
[ ] store install response reports the requested version and expected absolute path
[ ] CPA process is healthy and bound only where expected
[ ] log contains plugin loaded and plugin registered for cpamp-theme-studio
[ ] authenticated CPA plugin listing says registered=true and effective_enabled=true
[ ] authenticated plugin listing reports menus=[] for cpamp-theme-studio
[ ] GET /v0/resource/plugins/cpamp-theme-studio/studio returns 200 HTML
[ ] GET .../studio?asset=loader returns 200 JavaScript with nosniff and cache headers
[ ] GET .../studio?asset=font-regular and font-semibold return 200 font/woff2
[ ] GET .../studio?asset=restart-status returns read-only JSON with no PID or secret
[ ] management.html has exactly one start marker and one end marker
[ ] Theme Studio's installed row and store card each contain one confirmed Restart CPA control
[ ] canceling the confirmation leaves CPA_PID, process_instance, config SHA-256, and panel usability unchanged
[ ] after any install or upgrade, the effective CPA process was restarted; process_instance changed; and the loader marker v= value exactly matches the installed plugin version
[ ] CPAMP still reaches its login/dashboard, has no floating/sidebar Theme Studio item, and exposes the studio through the existing top-right Theme control
[ ] top-right Theme control -> change theme -> X/scrim/Escape -> reopen succeeds repeatedly without duplicate mounts or stuck body overflow
[ ] JetBrains Mono is loaded in the browser and one setting change persists after browser reload
```

12. Simulate an upstream update only in a disposable copy or approved maintenance window: replace the panel with the same official version and confirm reinjection within `watch_seconds`.
13. Test disable cleanup: set only this plugin's `enabled` to false, confirm the resource becomes 404 and the panel SHA-256 returns to the pre-injection value, then re-enable if deployment is accepted.

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
Store source/discovery/install response path:
Backups created:
Files changed:
Services restarted:
Restart mode/service/supervisor evidence:
Restart cancellation/confirmation evidence:
Acceptance checks:
Lifecycle checks:
Security checks:
Not tested:
Rollback status:
```
