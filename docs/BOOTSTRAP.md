# Linux restart bootstrap

[简体中文](BOOTSTRAP.zh-CN.md) · [Deployment](DEPLOYMENT.md) · [Agent runbook](AGENT_DEPLOYMENT.md)

CPAMP can install a plugin archive, but a marketplace plugin running inside CPA cannot safely grant itself root or restart its own systemd unit. Theme Studio therefore uses a one-time, administrator-approved bootstrap on Linux. After that bootstrap, marketplace installs, upgrades, and the confirmed card control can complete without a deployment Agent.

The bootstrap is optional. Without it, Theme Studio still works after an operator restarts CPA manually.

## What it installs

The bootstrap resolves the live CPA process and binds all future actions to exactly one running systemd service. It then:

- creates a timestamped backup under `/var/lib/cpamp-theme-studio/backups/`;
- minimally merges the absolute plugin directory, community store source, panel path, and `restart_mode: broker` into CPA's effective `config.yaml`;
- installs a root-owned broker at `/usr/local/libexec/cpamp-theme-studio/cpamp-theme-bootstrap`;
- installs `cpamp-theme-studio-restart.path`, `.timer`, and `.service`;
- gives only the detected CPA service account permission to create `/run/cpamp-theme-studio/restart.request`;
- optionally externalizes an embedded Manager Server panel when an active `--panel-url` is supplied; and
- restarts and verifies CPA before accepting the bootstrap state.

The broker never accepts a service name from the browser. Its state file is root-only, its systemd service has no shell or `sudo` path, and its writable paths are restricted to the selected config, panel, plugin, state, and runtime directories.

## Requirements and limits

- Linux with systemd, `curl`, `unzip`, and `sha256sum`.
- A running CPA service whose executable name starts with `cli-proxy-api` or an explicit `--service`.
- Root only for `--apply` and `--rollback`; a dry run can be performed first without writes.
- A writable CPAMP lightweight panel, an existing Manager Server `PANEL_PATH`, or `--panel-url` so the bootstrap can externalize the active Manager panel.
- The CPA service itself must be able to download marketplace assets. `--download-proxy` applies to Release and public-panel downloads made by this bootstrap process; it does not change CPA's systemd environment.

Docker, Windows services, launchd, and manually started CPA processes are not handled by this bootstrap. Use a platform-native supervisor or restart CPA manually in those environments.

## Verified installation

Pin a release instead of running an unversioned script. The following example uses a proxy on port 7890; omit both `--proxy` and `--download-proxy` when direct access works.

```bash
version=0.2.0
proxy=http://127.0.0.1:7890
base="https://github.com/Cec1c/cpamp-theme-studio/releases/download/v${version}"

curl --proxy "${proxy}" -fLO "${base}/bootstrap-linux.sh"
curl --proxy "${proxy}" -fLO "${base}/checksums.txt"
grep ' bootstrap-linux.sh$' checksums.txt | sha256sum -c -
chmod 0755 bootstrap-linux.sh
```

First inspect the resolved plan. Add explicit paths whenever automatic detection is ambiguous:

```bash
./bootstrap-linux.sh \
  --bootstrap-version "${version}" \
  --download-proxy "${proxy}" \
  --service cli-proxy-api.service \
  --config /opt/cpa/cpa/config.yaml \
  --panel-path /opt/cpa/cpa/static/management.html \
  --panel-url https://example.com/management.html
```

The dry run prints the selected CPA service/PID, Manager service, config, panel, public URL, platform plugin directory, and whether panel externalization is required. Review every path, then repeat the same command with `sudo` and `--apply`:

```bash
sudo ./bootstrap-linux.sh \
  --bootstrap-version "${version}" \
  --download-proxy "${proxy}" \
  --apply \
  --service cli-proxy-api.service \
  --config /opt/cpa/cpa/config.yaml \
  --panel-path /opt/cpa/cpa/static/management.html \
  --panel-url https://example.com/management.html
```

When exactly one CPA systemd service and one panel file exist, `--service`, `--config`, and `--panel-path` may be omitted. `--panel-url` is strongly recommended: it proves the public panel matches the selected local file, and is required when externalizing an embedded Manager Server panel. If more than one candidate exists, the bootstrap stops instead of guessing.

## Marketplace flow after bootstrap

1. Open CPAMP → Plugins → Plugin Store and install or upgrade Theme Studio.
2. CPA writes the versioned library and `store.version` into the effective config.
3. The path unit notices the plugin-directory change; a 15-second timer covers the path-unit re-arm window.
4. The broker waits for the library and config to settle, restarts only the bound CPA unit, and verifies a new process, health, plugin resource/version, and the active panel marker.
5. On success it records the new file hashes and plugin config as the accepted state. On restart or acceptance failure it removes only the unaccepted Theme Studio library, restores only Theme Studio's accepted config subtree, restarts CPA, and verifies the previous version.

The Theme Studio card uses the same broker. After confirmation, the plugin writes a small request file; it does not call `systemctl`, expose a restart endpoint, or read the Management Key.

## Verification and diagnostics

```bash
systemctl is-active cpamp-theme-studio-restart.path
systemctl is-active cpamp-theme-studio-restart.timer
systemctl status cpamp-theme-studio-restart.service --no-pager
journalctl -u cpamp-theme-studio-restart.service -n 100 --no-pager
sudo cat /etc/cpamp-theme-studio/bootstrap.json
```

Also verify the real dashboard, not only systemd:

```bash
curl -fsS http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio >/dev/null
test "$(grep -c 'cpamp-theme-studio:start' /absolute/path/to/management.html)" -eq 1
test "$(grep -c 'cpamp-theme-studio:end' /absolute/path/to/management.html)" -eq 1
```

The state file contains service names, paths, accepted plugin hashes, and a snapshot of this plugin's config. It contains no Management Key or API key, but should remain root-readable only.

## Rollback

The apply output prints a backup ID. To revert the bootstrap files, effective config, panel, and systemd units to that snapshot:

```bash
sudo /usr/local/libexec/cpamp-theme-studio/cpamp-theme-bootstrap \
  --rollback <backup-id>
```

Rollback disables the broker units, restores the recorded files with their modes and ownership, reloads systemd, and restarts the recorded Manager/CPA services. Backup evidence remains under `/var/lib/cpamp-theme-studio/` for inspection; remove it only after a separate operator review.
