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

## 3. Install from a release

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

Versioned names such as `cpamp-theme-studio-v0.1.0.dll` are also accepted by CPA. Do not keep multiple unversioned copies.

## 4. Build from source

Use this path before the first release or when auditing the exact source deployed to a server.

```bash
git clone https://github.com/Cec1c/cpamp-theme-studio.git
cd cpamp-theme-studio
go test ./...
node --check assets/loader.js
./scripts/package.sh 0.1.0-dev
```

Windows:

```powershell
git clone https://github.com/Cec1c/cpamp-theme-studio.git
Set-Location .\cpamp-theme-studio
go test ./...
node --check .\assets\loader.js
.\scripts\package.ps1 -Version 0.1.0-dev
```

The build requires Go 1.26+ and a native C compiler. Build on the same OS/architecture as the target because the plugin uses CGO `c-shared` mode.

## 5. Configure CPA

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

## 6. Manager Server with external PANEL_PATH

This mode works only when both processes see the same writable file.

1. Download the `management.html` matching the deployed CPAMP Manager Server release.
2. Configure Manager Server `PANEL_PATH` to that file.
3. Mount or share the same file into CPA's filesystem.
4. Set the plugin `panel_path` to CPA's view of that shared file.
5. Restart CPA and Manager Server.

When the two services run in separate containers, mount one host file into both containers. The in-container paths may differ, but both mounts must refer to the same host file.

An embedded-only Manager Server panel cannot be patched. Do not claim login-page persistence in that mode.

## 7. Verify

After starting CPA, check its logs for both messages:

```text
pluginhost: plugin loaded plugin_id=cpamp-theme-studio
pluginhost: plugin registered plugin_id=cpamp-theme-studio
```

Check the public, read-only plugin page:

```bash
curl -fsS http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio >/dev/null
curl -fsS 'http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio?asset=loader' >/dev/null
```

Then open `management.html` in a browser and verify:

- A Theme Studio launcher appears in the lower-right corner.
- The editor opens and closes with keyboard focus preserved.
- A palette change survives a reload.
- Browser developer tools show no Theme Studio errors.
- The file contains exactly one start marker and one end marker.

```text
<!-- cpamp-theme-studio:start -->
<!-- cpamp-theme-studio:end -->
```

Do not use the marker alone as the health check; registration, resource responses, and real browser execution must also pass.

## 8. Upgrade

1. Download and verify the new archive and keep the old archive/library.
2. Set `plugins.configs.cpamp-theme-studio.enabled: false` while CPA is still running, then wait for the marker block to disappear and the resource route to return 404.
3. Stop CPA. A loaded DLL cannot be safely overwritten on Windows.
4. Replace the plugin library with the new version and re-enable its configuration.
5. Start CPA and repeat all verification checks.
6. Confirm the current CPAMP panel still has exactly one marker block.

Panel updates do not require reinstalling the plugin. If CPAMP replaces `management.html`, the watcher should restore the loader within `watch_seconds`.

## 9. Rollback

1. Disable `cpamp-theme-studio` in CPA config and wait for its marker block to disappear.
2. Stop CPA.
3. Restore the previous plugin library and previous plugin configuration.
4. Start CPA and verify the restored version.

Hot-disable cleanup is deterministic. Normal process-shutdown cleanup is best effort because CPA may exit before asynchronous native-plugin shutdown finishes. If CPA exited or crashed before cleanup, restore the backed-up panel or remove only the three-line block from the start marker through the end marker. Never delete unrelated scripts from `management.html`.

## 10. Uninstall

1. Set `plugins.configs.cpamp-theme-studio.enabled: false`.
2. Wait until the marker block is gone and the resource route returns 404.
3. Stop CPA.
4. Remove only `cpamp-theme-studio` plugin library files.
5. Remove the plugin configuration block if desired, then restart CPA.

The user's browser preferences can remain in local storage; they are harmless and allow settings to return after reinstall. Clear them manually only if a full preference reset is desired.
