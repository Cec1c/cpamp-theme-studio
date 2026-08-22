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
4. On a direct Linux/systemd host, install the release's one-time bootstrap before the market install. Download both bootstrap-linux.sh and checksums.txt, verify the script, run a no-write dry run, review every resolved service/path, then run the same pinned command with sudo and --apply. This is the only authorized root step. Do not pipe an unverified moving URL directly into a root shell.
5. An embedded Manager Server panel may be externalized only by that bootstrap with the active public --panel-url and an inspected Manager systemd unit. For containers/non-systemd, stop and report that a manually shared external PANEL_PATH is required.
6. Before writes, create an independent operator backup of only the effective CPA config, existing cpamp-theme-studio library, and external management.html. Preserve permissions and ownership. Also record the bootstrap's own backup ID.
7. Let bootstrap minimally merge plugins.enabled, an absolute plugins.dir, this repository's registry URL, absolute panel/config paths, and restart_mode=broker. Do not pre-copy a library before testing store discovery.
8. Use CPAMP's Plugin Store UI or the authenticated CPA plugin-store API to discover and install cpamp-theme-studio. Record the returned version and absolute install path. The market write must trigger broker restart and verification automatically; a file copied by hand is not a successful marketplace deployment.
9. Validate plugin loaded + registered logs, plugin listing with zero published menus, studio HTML, loader JavaScript, both JetBrains Mono resources, exactly one panel marker pair, the restart-status read-only resource, and repeated real-browser open/change/close plus reload persistence when browser access exists. Cancel the restart confirmation once before confirming it and prove that cancellation changed neither the CPA process nor config hash.
10. Simulate a failed broker upgrade only in a disposable fixture or explicitly approved maintenance window. Prove that the unaccepted library/config is removed, the previous accepted version returns, and the public panel remains usable.
11. On any failed acceptance check not already recovered by the broker, disable the plugin, wait for marker cleanup, restore backups, restart the previous version, and report the evidence.
12. Report bootstrap script/release SHA-256, dry-run plan, backup ID, active broker units, store discovery/install path, accepted version, validation results, and anything not tested. Never print secrets.
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
RESTART_MODE=broker|auto|systemd|self-exit|disabled
RESTART_SERVICE=optional; systemd unit whose MainPID must equal CPA_PID
SUPERVISOR_RESTART_POLICY=required before self-exit
DESIRED_VERSION=
BOOTSTRAP_VERSION=same pinned release as DESIRED_VERSION
BOOTSTRAP_MODE=broker|not-applicable
CPA_SYSTEMD_UNIT=required for bootstrap; auto-detect only when unique
MANAGER_SYSTEMD_UNIT=optional; required to externalize embedded Manager panel
PUBLIC_PANEL_URL=strongly recommended; required for embedded Manager externalization
DOWNLOAD_PROXY=optional
STORE_SOURCE=https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json
STORE_INSTALL_PATH=resolved from the install response; never guessed
```

If `PANEL_MODE=manager-embedded`, proceed only when CPA and Manager are direct Linux/systemd services and the pinned bootstrap dry run can safely externalize the active `PUBLIC_PANEL_URL`. Otherwise explain how to provide a shared external `PANEL_PATH` and stop without patching binaries or container layers.

若 `PANEL_MODE=manager-embedded`，只有 CPA 与 Manager 都是 Linux/systemd 直接服务，且固定版本 bootstrap dry-run 能安全外置活动 `PUBLIC_PANEL_URL` 时才能继续；否则应说明如何提供共享外置 `PANEL_PATH`，不得修改二进制或容器层。

## Deterministic procedure / 确定性流程

1. Inspect effective process arguments and working directory; do not rely on a similarly named inactive installation.
2. Check the listening address. Use loopback for acceptance testing.
3. Resolve effective `plugins.dir` to an absolute path. If it is relative, resolve it against the live process working directory, not the shell's current directory.
4. Determine the panel path from the live service/config and verify it is a regular, writable, non-symlink `.html` file under 64 MiB.
5. Confirm the requested GitHub Release exists and contains the matching platform ZIP plus `checksums.txt`; confirm the custom registry's matching artifact URL, size, and SHA-256 agree with those bytes. If any item is missing or disagrees, stop.
6. Create timestamped backups in an operator-approved backup directory. A backup is not successful until its size and SHA-256 are recorded.
7. On direct Linux/systemd, download the pinned release's `bootstrap-linux.sh` and `checksums.txt` through `DOWNLOAD_PROXY` when needed; verify the script's SHA-256. Run it once without `--apply`, record the resolved CPA PID/unit, Manager unit, config, panel, plugin directory, public URL, and externalization decision, then run the same arguments with `sudo --apply`. Follow [BOOTSTRAP.md](BOOTSTRAP.md) exactly. Record the generated backup ID and prove the `.path` and `.timer` units are active.
8. If bootstrap is not applicable, minimally merge the store configuration manually and preserve unrelated plugin sources/configurations:

```yaml
plugins:
  enabled: true
  dir: "/absolute/path/to/cpa/plugins"
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

9. Reload the effective configuration, then verify the CPAMP Plugin Store lists `cpamp-theme-studio` from that source as install type `direct`, with the current OS/architecture present and no error for this custom source. An unrelated official-source error must be reported but does not invalidate a healthy custom source.
10. Install the requested version through CPAMP or `POST /v0/management/plugin-store/cpamp-theme-studio/install?source=<resolved-source-id>&version=<version>`. Use the deployment's existing authenticated management path without printing the key. Record the response `path` and confirm it is under the resolved `<plugins.dir>/<goos>/<goarch>/`.
11. With bootstrap, do not add a manual restart between the store response and broker observation. Prove that the broker sees the file/config settle, replaces the CPA process, verifies the requested resource/marker version, and stores the version as accepted. If bootstrap is not applicable and panel discovery is ambiguous, update only this plugin's absolute paths through CPAMP and select a legacy restart behavior from this evidence:

```text
bootstrap-bound Linux systemd service: restart_mode=broker
systemd direct service without bootstrap: restart_mode=auto, or systemd + an explicit unit after MainPID validation
Docker/other supervisor: restart_mode=self-exit only after proving that exit code 75 causes that exact CPA process to be restarted
manual/no supervisor/uncertain wrapper: restart_mode=disabled and perform an operator-approved manual restart
```

The card does not expose a restart POST endpoint or read the Management Key. It opens CPAMP's native Edit configuration drawer and saves a random `restart_request` through the existing authenticated session.
12. Verify in order:

```text
[ ] CPAMP store lists the plugin from the configured community source
[ ] bootstrap script SHA-256 matched the pinned release and dry-run/apply selected the same exact paths
[ ] cpamp-theme-studio-restart.path and .timer are active and state is root-only
[ ] store install response reports the requested version and expected absolute path
[ ] store install automatically replaced CPA and bootstrap accepted the requested version
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

13. Simulate an upstream update only in a disposable copy or approved maintenance window: replace the panel with the same official version and confirm reinjection within `watch_seconds`.
14. Test disable cleanup: set only this plugin's `enabled` to false, confirm the resource becomes 404 and the panel SHA-256 returns to the pre-injection value, then re-enable if deployment is accepted.

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
Bootstrap version/script SHA-256/dry-run plan:
Bootstrap backup ID/units/accepted version:
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
