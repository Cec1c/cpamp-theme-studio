# v0.2.0 — Automatic marketplace restart and rollback

## English

This release closes the remaining gap between “installed from the CPAMP Plugin Store” and “actually running the new Theme Studio version.” A marketplace plugin cannot safely grant itself root privileges, so v0.2.0 introduces a one-time, administrator-reviewed Linux bootstrap. After that bootstrap, later CPAMP marketplace installs/upgrades and confirmed card restarts no longer need a deployment Agent.

### New

- A pinned `bootstrap-linux.sh` Release asset downloads the matching Linux archive through an optional proxy and verifies it against `checksums.txt`.
- Linux archives include a statically linked `cpamp-theme-bootstrap` binary and the downloader.
- Bootstrap has a no-write dry run, explicit service/path overrides, scoped backups, root-only state, and `--rollback <backup-id>`.
- It detects the live CPA systemd service, effective config, plugin directory, and active panel; ambiguous candidates stop instead of being guessed.
- It can use an existing Manager Server `PANEL_PATH`, or externalize an embedded Manager panel from an operator-supplied active public URL.
- A root-owned `.path` broker watches marketplace writes; a 15-second timer covers the narrow path-unit re-arm window.
- Plugin card confirmation now supports `restart_mode: broker` and writes only a small request file. The plugin never receives `systemctl` or `sudo` authority.
- Every candidate install must pass process replacement, CPA health, exact plugin resource version, and singleton active-panel marker checks before it becomes the accepted state.
- A candidate that restarts but fails acceptance is removed; only Theme Studio's plugin config subtree is restored, CPA is restarted, and the previously accepted version is verified.

### Deployment boundary

The one-time bootstrap requires root on a direct Linux/systemd host. This cannot be performed by a pure marketplace/frontend package. Without bootstrap, Theme Studio remains usable after a manual or separately supervised CPA restart. Docker, Windows, macOS, and manual processes continue to use their platform-native supervisor or manual restart.

Read [Linux bootstrap](BOOTSTRAP.md) before applying it. The recommended flow downloads both `bootstrap-linux.sh` and `checksums.txt`, verifies the pinned script, reviews a dry-run plan, then repeats exactly that command with `sudo --apply`. `--download-proxy http://127.0.0.1:7890` affects Release/public-panel downloads made by this bootstrap process only; CPA itself still needs a reachable proxy in its own service/network namespace for marketplace downloads.

### Validation

Validated with a real systemd-managed CPA v7.2.138 fixture on Linux/amd64:

```text
bootstrap apply: ok
market install auto-restart: ok
confirmed card broker restart: ok
failed upgrade rollback: ok
marker count: 1/1
```

The test covered bootstrap dry-run/apply, first marketplace file/config write, loaded/registered plugin resources, an authenticated card confirmation, a deliberately damaged `.so`, automatic deletion and config restoration, previous-version recovery, and one start/end marker in the active CPAMP panel. Windows tests, Linux race tests, Go vet, loader syntax validation, native shared-library builds, and Linux ZIP/bootstrap contents also passed locally before tagging.

## 简体中文

此版本打通了“已从 CPAMP 插件市场落盘”到“新 Theme Studio 版本确实已经运行”的最后一段流程。市场插件不能安全地自行获取 root，因此 v0.2.0 新增一次由管理员核对的 Linux bootstrap。完成这一次操作后，后续 CPAMP 市场安装/升级和插件卡片确认重启不再需要部署 Agent。

### 新增

- 固定版本的 Release 资产 `bootstrap-linux.sh`：支持可选下载代理，并通过 `checksums.txt` 校验所下载的 Linux 包。
- Linux 压缩包同时包含静态链接的 `cpamp-theme-bootstrap` 二进制和下载器。
- bootstrap 提供无写入 dry-run、明确的服务/路径覆盖、限定范围备份、仅 root 可读状态，以及 `--rollback <backup-id>`。
- 自动识别活动 CPA systemd 服务、有效配置、插件目录和真实面板；存在歧义时停止，不会猜测。
- 可复用 Manager Server 已有 `PANEL_PATH`；也可根据运维人员提供的活动公网 URL，把内嵌 Manager 面板外置。
- root 持有的 `.path` broker 监听市场写入，15 秒 timer 覆盖 path 单元重新布防的窄窗口。
- 插件卡片确认新增 `restart_mode: broker`；插件只写一个很小的请求文件，不会获得 `systemctl` 或 `sudo` 权限。
- 候选安装必须通过进程替换、CPA 健康、精确插件资源版本和活动面板唯一标记验证，才会成为 accepted 状态。
- 若候选版本能够重启但验收失败，broker 会删除该候选，只恢复 Theme Studio 的插件配置子树，再重启并验证上一个 accepted 版本。

### 部署边界

一次性 bootstrap 只适用于 Linux/systemd 直接部署，并需要 root；纯市场包/前端包无法完成这一步。未安装 bootstrap 时，手工重启或由另行验证的 supervisor 重启 CPA 后，Theme Studio 仍可正常使用。Docker、Windows、macOS 和手工进程继续使用平台原生 supervisor 或手工重启。

执行前请阅读 [Linux bootstrap](BOOTSTRAP.zh-CN.md)。推荐同时下载 `bootstrap-linux.sh` 与 `checksums.txt`，校验固定版本脚本，先检查 dry-run 方案，再把完全相同的命令加 `sudo --apply`。`--download-proxy http://127.0.0.1:7890` 只影响本次 bootstrap 进程的 Release/公网页面下载；CPA 市场下载仍需要它自己的服务/网络命名空间能够访问代理。

### 验证

已在 Linux/amd64、真实 systemd 管理的 CPA v7.2.138 测试环境通过：

```text
bootstrap apply: ok
market install auto-restart: ok
confirmed card broker restart: ok
failed upgrade rollback: ok
marker count: 1/1
```

测试覆盖 bootstrap dry-run/apply、首次市场文件/配置写入、插件 loaded/registered 及资源、已认证卡片确认、故意损坏的 `.so`、自动删除与配置恢复、旧版本重新工作，以及活动 CPAMP 面板 start/end 标记各 1 个。Windows 测试、Linux race、Go vet、loader 语法、原生动态库构建和 Linux ZIP/bootstrap 内容验证也在 tag 前通过。
