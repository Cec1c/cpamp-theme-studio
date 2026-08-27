# Linux 自动重启 bootstrap

[English](BOOTSTRAP.md) · [部署文档](DEPLOYMENT.zh-CN.md) · [Agent 部署手册](AGENT_DEPLOYMENT.md)

CPAMP 可以安装插件压缩包，但运行在 CPA 进程内的市场插件不应该自行获取 root 权限，也不能安全地重启承载自己的 systemd 服务。因此，Theme Studio 在 Linux 上使用一次由管理员明确批准的 bootstrap。完成这一次操作后，后续市场安装、升级和插件卡片确认重启都不再需要部署 Agent。

bootstrap 不是插件运行的硬性条件；未安装时，只需由运维人员手工重启 CPA。

## 会安装什么

bootstrap 会解析正在运行的 CPA 进程，并把后续操作固定绑定到唯一的 systemd 服务。它将：

- 在 `/var/lib/cpamp-theme-studio/backups/` 创建带时间戳的备份；
- 向 CPA 的有效 `config.yaml` 最小合并插件绝对目录、社区市场源、面板路径和 `restart_mode: broker`；
- 把 root 持有的 broker 安装到 `/usr/local/libexec/cpamp-theme-studio/cpamp-theme-bootstrap`；
- 安装 `cpamp-theme-studio-restart.path`、`.timer` 和 `.service`；
- 只允许已识别的 CPA 服务账户创建 `/run/cpamp-theme-studio/restart.request`；
- 在提供活动 `--panel-url` 时，可把 Manager Server 的内嵌面板安全外置；
- 重启并验证 CPA，通过后才记录 bootstrap 状态。

浏览器不能向 broker 传入服务名。状态文件只允许 root 读取，systemd 服务不会执行 shell 或 `sudo`，可写范围也被限制为选定的配置、面板、插件、状态和运行时目录。

## 要求与边界

- Linux、systemd、`curl`、`unzip`、`sha256sum`。
- 正在运行且可识别的 CPA systemd 服务；可执行文件名应以 `cli-proxy-api` 开头，或显式传入 `--service`。
- 只有 `--apply` 和 `--rollback` 需要 root；可以先无写入地执行 dry-run。
- 可写的 CPAMP 轻量面板；若使用 Manager Server，则必须提供准确的公网 `--panel-url`。无论复用已有 Manager `PANEL_PATH` 还是外置内嵌面板都需要该 URL。
- CPA 服务本身仍要能够下载市场资产。`--download-proxy` 会用于本次 bootstrap 进程的 Release 与公网页面下载，但不会修改 CPA 的 systemd 代理环境。

此 bootstrap 不处理 Docker、Windows 服务、launchd 或手工启动的 CPA；这些环境应使用相应平台的 supervisor，或手工重启。

## 校验后安装

应固定 Release 版本，不要直接执行未固定版本的远程脚本。下面示例使用 7890 代理；可直连时删除 `--proxy` 和 `--download-proxy`：

```bash
version=0.2.0
proxy=http://127.0.0.1:7890
base="https://github.com/Cec1c/cpamp-theme-studio/releases/download/v${version}"

curl --proxy "${proxy}" -fLO "${base}/bootstrap-linux.sh"
curl --proxy "${proxy}" -fLO "${base}/checksums.txt"
grep ' bootstrap-linux.sh$' checksums.txt | sha256sum -c -
chmod 0755 bootstrap-linux.sh
```

先检查自动解析出的方案；自动发现有歧义时必须提供明确路径：

```bash
./bootstrap-linux.sh \
  --bootstrap-version "${version}" \
  --download-proxy "${proxy}" \
  --service cli-proxy-api.service \
  --config /opt/cpa/cpa/config.yaml \
  --panel-path /opt/cpa/cpa/static/management.html \
  --panel-url https://example.com/management.html
```

dry-run 会打印 CPA 服务/PID、Manager 服务、有效配置、活动面板、公网页面、平台插件目录，以及是否需要外置面板。逐项核对后，使用同一组参数加 `sudo` 和 `--apply`：

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

若主机上恰好只有一个 CPA systemd 服务和一个面板文件，可以省略 `--service`、`--config`、`--panel-path`。轻量面板强烈建议提供 `--panel-url`；所有 Manager Server 部署都必须提供该参数，以便在接受补丁前证明公网页面与选中的本地文件一致。有多个候选时 bootstrap 会停止，不会猜测。

## bootstrap 后的市场流程

1. 打开 CPAMP → 插件 → 插件商店，安装或升级 Theme Studio。
2. CPA 将带版本号的动态库和 `store.version` 写入有效配置。
3. path 单元发现插件目录变化；15 秒 timer 用来覆盖 path 单元重新布防时的窄窗口。
4. broker 等待动态库与配置稳定，只重启已绑定的 CPA unit，并检查新进程、健康状态、插件资源/版本和活动面板标记。
5. 成功后记录新文件哈希与插件配置为 accepted 状态。若重启或验收失败，只删除未通过的 Theme Studio 动态库，只恢复 Theme Studio 的 accepted 配置子树，再次重启并验证旧版本。

插件卡片也使用同一个 broker。确认后插件只写入一个很小的请求文件；它不会调用 `systemctl`、不会公开重启 API，也不会读取 Management Key。

## 验证与诊断

```bash
systemctl is-active cpamp-theme-studio-restart.path
systemctl is-active cpamp-theme-studio-restart.timer
systemctl status cpamp-theme-studio-restart.service --no-pager
journalctl -u cpamp-theme-studio-restart.service -n 100 --no-pager
sudo cat /etc/cpamp-theme-studio/bootstrap.json
```

还要验证真实主页，不能只看 systemd：

```bash
curl -fsS http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio >/dev/null
test "$(grep -c 'cpamp-theme-studio:start' /absolute/path/to/management.html)" -eq 1
test "$(grep -c 'cpamp-theme-studio:end' /absolute/path/to/management.html)" -eq 1
```

状态文件包含服务名、路径、accepted 插件哈希和本插件配置快照，不包含 Management Key 或 API Key，但仍应保持仅 root 可读。

## 回滚

apply 成功时会打印备份 ID。要把 bootstrap 文件、有效配置、面板和 systemd 单元恢复到该快照：

```bash
sudo /usr/local/libexec/cpamp-theme-studio/cpamp-theme-bootstrap \
  --rollback <backup-id>
```

回滚会停用 broker 单元，按原权限与属主恢复记录文件，重载 systemd，并重启备份中记录的 Manager/CPA 服务。备份证据会继续保留在 `/var/lib/cpamp-theme-studio/`；应在单独的运维复核后再决定是否删除。
