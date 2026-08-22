# Deployment, upgrade, rollback, and uninstall

[简体中文](DEPLOYMENT.zh-CN.md) · [Agent runbook](AGENT_DEPLOYMENT.md)

This guide deploys CPAMP Theme Studio into an existing CPA installation. It does not install CPA or CPAMP and it does not expose a management port.

## 1. Collect inputs

Record these values before changing anything:

- CPA home/working directory.
- CPA `config.yaml` path.
- CPA plugin directory (`plugins` by default).
- Host OS and architecture (`windows`, `linux`, or `darwin`; `amd64` or `arm64`).
- Panel mode: CPA Lightweight Panel or Manager Server with an external `PANEL_PATH`.
- Exact writable `management.html` path.
- Desired plugin version.

Do not print or commit the CPA Management Key. The plugin does not need it.

## 2. Preflight

1. Confirm CPA is v7.2.138 or a version already validated in your environment.
2. Confirm the panel is CPAMP v1.12.2 or a version already validated in your environment.
3. Confirm `management.html` is a regular writable file, not a symbolic link, and is no larger than 64 MiB.
4. Back up CPA `config.yaml`, the current plugin library, and the panel file before the first deployment.
5. Keep CPA bound to localhost while testing.

Optional download proxy:

```bash
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890
export ALL_PROXY=http://127.0.0.1:7890
```

```powershell
$env:HTTP_PROXY = 'http://127.0.0.1:7890'
$env:HTTPS_PROXY = 'http://127.0.0.1:7890'
$env:ALL_PROXY = 'http://127.0.0.1:7890'
```

Use the proxy only when it exists on the machine performing the download.

## 3. Install through the CPAMP Plugin Store (recommended)

CPAMP currently uses CPA's plugin-store APIs for discovery, download, verification, installation, and lifecycle control. The product feature is browser-side code, but a minimal CPA native bridge is still shipped so the extension can enter the CPAMP store and page container. This registry uses schema-v2 direct artifacts with a pinned URL, byte size, and SHA-256 for each platform, so installation does not consume anonymous GitHub REST API quota.

Add the community source on CPAMP's Configuration page, or minimally merge these fields into the effective CPA `config.yaml`:

```yaml
plugins:
  enabled: true
  dir: "/absolute/path/to/cpa/plugins"
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

An absolute `plugins.dir` is strongly recommended. CPA v7.2.138 writes a store installation to:

```text
<plugins.dir>/<goos>/<goarch>/cpamp-theme-studio-v<version>.<dll|so|dylib>
```

Then use CPAMP:

1. Open Plugins → Plugin Store and confirm the custom source has no error.
2. Search for `CPAMP Theme Studio` and choose Latest or a specific version such as `0.1.3`.
3. Complete the third-party confirmation and install it.
4. Record the returned version and actual `path`; never assume a relative directory is beside the CPA executable.
5. Restart the effective CPA service after every install or upgrade. Theme Studio's panel watcher is process-local; CPA hot reload can leave the retired version alive until process exit, causing old and new loader cache keys to alternate.
6. Confirm `registered=true` and `effective_enabled=true` under Installed Plugins, then return to the dashboard and click CPAMP's existing top-right Theme control. No floating or sidebar Theme Studio entry is expected.

The store path passes only when discovery, pinned SHA-256 verification, installation, target path, registration, and the HTTP 200 resource check all succeed. If store installation fails, do not manually copy a library and report a successful store deployment. Capture the CPAMP response and CPA logs first; use the next section only as an explicitly reported manual fallback.

## 4. Install manually from a release (fallback)

Download both the platform archive and `checksums.txt` from the same release. Archive names follow:

```text
cpamp-theme-studio_<version>_<goos>_<goarch>.zip
```

Verify SHA-256 before extracting.

Linux/macOS:

```bash
sha256sum -c checksums.txt --ignore-missing
unzip cpamp-theme-studio_<version>_<goos>_<goarch>.zip
```

Windows PowerShell:

```powershell
$archive = 'cpamp-theme-studio_<version>_windows_amd64.zip'
$expected = (Get-Content .\checksums.txt | Where-Object { $_ -match [regex]::Escape($archive) }).Split()[0]
$actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
if ($actual -ne $expected.ToLowerInvariant()) { throw 'SHA-256 mismatch' }
Expand-Archive -LiteralPath $archive -DestinationPath .\theme-studio-package
```

Stop CPA, then copy the library into the matching platform directory:

```text
<CPA_HOME>/plugins/<goos>/<goarch>/cpamp-theme-studio.<dll|so|dylib>
```

Versioned names such as `cpamp-theme-studio-v0.1.3.dll` are also accepted by CPA. Do not keep multiple unversioned copies.

## 5. Build from source

Use this only for development, audit, or an operator-approved manual fallback. A production store deployment must use a published release with checksums. If no release exists, stop instead of building on the server and presenting it as a store installation.

```bash
git clone https://github.com/Cec1c/cpamp-theme-studio.git
cd cpamp-theme-studio
go test ./...
node --check assets/loader.js
./scripts/package.sh 0.1.3-dev
```

Windows:

```powershell
git clone https://github.com/Cec1c/cpamp-theme-studio.git
Set-Location .\cpamp-theme-studio
go test ./...
node --check .\assets\loader.js
.\scripts\package.ps1 -Version 0.1.3-dev
```

The build requires Go 1.26+ and a native C compiler. Build on the same OS/architecture as the target because the plugin uses CGO `c-shared` mode.

## 6. Configure CPA

Merge this block into CPA `config.yaml`; do not replace unrelated settings:

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

For the standard CPA Lightweight Panel, leave `panel_path` empty if the panel is at `<CPA_HOME>/static/management.html`.

If CPA is started with an unusual working directory or wrapper, set absolute paths:

```yaml
      panel_path: "/absolute/path/to/management.html"
      host_config_path: "/absolute/path/to/config.yaml"
```

On Windows, quote paths and use either forward slashes or escaped backslashes.

## 7. Manager Server with external PANEL_PATH

This mode works only when both processes see the same writable file.

1. Download the `management.html` matching the deployed CPAMP Manager Server release.
2. Configure Manager Server `PANEL_PATH` to that file.
3. Mount or share the same file into CPA's filesystem.
4. Set the plugin `panel_path` to CPA's view of that shared file.
5. Restart CPA and Manager Server.

When the two services run in separate containers, mount one host file into both containers. The in-container paths may differ, but both mounts must refer to the same host file.

An embedded-only Manager Server panel cannot be patched. Do not claim login-page persistence in that mode.

## 8. Verify

After starting CPA, check its logs for both messages:

```text
pluginhost: plugin loaded plugin_id=cpamp-theme-studio
pluginhost: plugin registered plugin_id=cpamp-theme-studio
```

Check the public, read-only plugin page:

```bash
curl -fsS http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio >/dev/null
curl -fsS 'http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio?asset=loader' >/dev/null
curl -fsS 'http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio?asset=font-regular' >/dev/null
curl -fsS 'http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio?asset=font-semibold' >/dev/null
```

Then open `management.html` in a browser and verify:

- CPAMP's existing top-right Theme control is marked as the Theme Studio dialog trigger and keeps the native button styling.
- No `Theme Studio` item appears in the left sidebar and no floating launcher is created.
- The editor opens, changes a theme, and closes repeatedly through X, the scrim, and Escape with keyboard focus preserved.
- A palette change survives a reload.
- Browser-computed typography starts with `JetBrains Mono`; both bundled font weights load successfully.
- Each cycle leaves exactly one mount, host Theme control, and stage, and restores `body` overflow after closing.
- Browser developer tools show no Theme Studio errors.
- The file contains exactly one start marker and one end marker.

```text
<!-- cpamp-theme-studio:start -->
<!-- cpamp-theme-studio:end -->
```

Do not use the marker alone as the health check; registration, resource responses, and real browser execution must also pass.

## 9. Upgrade

1. Download and verify the new archive and keep the old archive/library.
2. Set `plugins.configs.cpamp-theme-studio.enabled: false` while CPA is still running, then wait for the marker block to disappear and the resource route to return 404.
3. Stop CPA. A loaded DLL cannot be safely overwritten on Windows.
4. Replace the plugin library with the new version and re-enable its configuration.
5. Start CPA and repeat all verification checks.
6. Confirm the current CPAMP panel still has exactly one marker block.

When upgrading to `0.1.3`, restart the effective CPA process first and then refresh the panel. Confirm that the injected loader URL ends in `v=0.1.3`; existing browser preferences are retained, the native top-right Theme control remains the single entry point, and the Typography and Density sections are reachable inside the scrolling editor.

Panel updates do not require reinstalling the plugin. If CPAMP replaces `management.html`, the watcher should restore the loader within `watch_seconds`.

## 10. Rollback

1. Disable `cpamp-theme-studio` in CPA config and wait for its marker block to disappear.
2. Stop CPA.
3. Restore the previous plugin library and previous plugin configuration.
4. Start CPA and verify the restored version.

Hot-disable cleanup is deterministic. Normal process-shutdown cleanup is best effort because CPA may exit before asynchronous native-plugin shutdown finishes. If CPA exited or crashed before cleanup, restore the backed-up panel or remove only the three-line block from the start marker through the end marker. Never delete unrelated scripts from `management.html`.

## 11. Uninstall

1. Set `plugins.configs.cpamp-theme-studio.enabled: false`.
2. Wait until the marker block is gone and the resource route returns 404.
3. Stop CPA.
4. Remove only `cpamp-theme-studio` plugin library files.
5. Remove the plugin configuration block if desired, then restart CPA.

The user's browser preferences can remain in local storage; they are harmless and allow settings to return after reinstall. Clear them manually only if a full preference reset is desired.
