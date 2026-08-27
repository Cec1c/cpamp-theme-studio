# 部署、升级、回滚与卸载

[English](DEPLOYMENT.md) · [Linux bootstrap](BOOTSTRAP.zh-CN.md) · [Agent 部署手册](AGENT_DEPLOYMENT.md)

本文将 CPAMP 主题工作室部署到现有 CPA。它不会安装 CPA/CPAMP，也不会开放新的管理端口。

## 1. 收集参数

修改前先确认：

- CPA 主目录/工作目录。
- CPA `config.yaml` 路径。
- CPA 插件目录，默认是 `plugins`。
- 系统与架构：`windows`、`linux`、`darwin`；`amd64` 或 `arm64`。
- 面板模式：CPA 轻量面板、使用外置 `PANEL_PATH` 的 Manager Server，或准备由 Linux bootstrap 外置的 Manager 内嵌面板。
- 可写 `management.html` 的准确路径。
- 需要部署的插件版本。

不要输出或提交 CPA Management Key；插件不需要它。

## 2. 部署前检查

1. CPA 为 v7.2.138，或已在本环境验证过的版本。
2. 面板为 CPAMP v1.12.2，或已在本环境验证过的版本。
3. `management.html` 是可写的普通文件，不是符号链接，且不超过 64 MiB。
4. 首次部署前备份 CPA `config.yaml`、当前插件动态库和面板文件。Linux bootstrap 会自动创建限定范围的备份，但仍应保留运维人员自己的独立备份。
5. 测试期间让 CPA 只监听本机。

可选下载代理：

```powershell
$env:HTTP_PROXY = 'http://127.0.0.1:7890'
$env:HTTPS_PROXY = 'http://127.0.0.1:7890'
$env:ALL_PROXY = 'http://127.0.0.1:7890'
```

```bash
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890
export ALL_PROXY=http://127.0.0.1:7890
```

只有执行下载的机器确实运行该代理时才设置。

## 3. 通过 CPAMP 插件市场安装（推荐）

CPAMP 当前使用 CPA 的插件商店接口完成发现、下载、校验、落盘和启用。本项目的主要功能是前端代码，但要进入 CPAMP 商店仍需要随 Release 分发最小 CPA 原生桥。清单采用 schema v2 直链资产，为每个平台固定 URL、字节数与 SHA-256，因此安装不消耗 GitHub REST API 的匿名配额。

Linux/systemd 直接部署应先执行[一次性自动重启 bootstrap](BOOTSTRAP.zh-CN.md)。市场包不能自行获取 root；经管理员核对的 bootstrap 会把一个窄权限 broker 固定绑定到唯一 CPA unit。此后市场写入会自动重启并验证 CPA，升级失败则恢复上一个 accepted Theme Studio 动态库/配置，不再需要 Agent。其他平台或明确不安装 bootstrap 时，仍应安排手工重启 CPA。

在 CPAMP 的“配置”页加入社区源，或把以下字段最小合并到实际生效的 CPA `config.yaml`：

```yaml
plugins:
  enabled: true
  dir: "/absolute/path/to/cpa/plugins"
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

`plugins.dir` 强烈建议使用绝对路径。CPA v7.2.138 会把市场安装结果写到：

```text
<plugins.dir>/<goos>/<goarch>/cpamp-theme-studio-v<version>.<dll|so|dylib>
```

随后在 CPAMP 中执行：

1. 打开“插件”→“插件商店”，确认自定义来源没有错误。
2. 搜索 `CPAMP Theme Studio`，选择 Latest 或 `0.2.0` 这类明确版本。
3. 完成第三方插件确认并点击安装。
4. 在安装结果中记录版本与实际 `path`，不要假定相对目录落在 CPA 可执行文件旁。
5. 已安装 bootstrap 时，插件目录写入会自动触发 broker；等待市场流程完成并让 CPA 恢复即可。卡片的“重启 CPA”确认控件仍可用于主动重启，并复用同一个 broker。未安装 bootstrap 时，使用已配置的旧重启方式或手工重启实际 CPA 服务。主题工作室的面板 watcher 属于进程内状态；CPA 热重载可能让退役版本存活到进程退出，导致新旧 loader 缓存版本交替写入。
6. 在“已安装插件”中确认 `registered=true`、`effective_enabled=true`，再回到仪表盘点击 CPAMP 右上角原有的“主题”控件；此版本不会出现悬浮按钮或 Theme Studio 侧栏项。

只有同时满足“商店可发现、固定 SHA-256 校验通过、安装返回成功、实际文件路径正确、插件已注册、页面资源 200”才算市场链路成功。若市场安装失败，不要改用手工复制后宣称市场部署成功；先记录 CPAMP 响应与 CPA 日志，再按下一节做明确标注的手工回退。

## 4. 手工从 Release 安装（回退）

从同一个 Release 下载平台压缩包和 `checksums.txt`。命名规则：

```text
cpamp-theme-studio_<version>_<goos>_<goarch>.zip
```

解压前校验 SHA-256。

Windows PowerShell：

```powershell
$archive = 'cpamp-theme-studio_<version>_windows_amd64.zip'
$expected = (Get-Content .\checksums.txt | Where-Object { $_ -match [regex]::Escape($archive) }).Split()[0]
$actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
if ($actual -ne $expected.ToLowerInvariant()) { throw 'SHA-256 不匹配' }
Expand-Archive -LiteralPath $archive -DestinationPath .\theme-studio-package
```

Linux/macOS：

```bash
sha256sum -c checksums.txt --ignore-missing
unzip cpamp-theme-studio_<version>_<goos>_<goarch>.zip
```

先停止 CPA，再复制到对应平台目录：

```text
<CPA_HOME>/plugins/<goos>/<goarch>/cpamp-theme-studio.<dll|so|dylib>
```

CPA 也接受 `cpamp-theme-studio-v0.2.0.dll` 这类带版本名。不要保留多个无版本文件副本。

## 5. 从源码构建

只在开发、审计或操作员明确批准手工回退时使用此方式。生产市场部署必须使用已发布且带校验和的 Release；没有 Release 时应停止，而不是在服务器上构建后冒充市场安装。

Windows：

```powershell
git clone https://github.com/Cec1c/cpamp-theme-studio.git
Set-Location .\cpamp-theme-studio
go test ./...
node --check .\assets\loader.js
.\scripts\package.ps1 -Version 0.2.0-dev
```

Linux/macOS：

```bash
git clone https://github.com/Cec1c/cpamp-theme-studio.git
cd cpamp-theme-studio
go test ./...
node --check assets/loader.js
./scripts/package.sh 0.2.0-dev
```

需要 Go 1.26+ 和本机 C 编译器。插件使用 CGO `c-shared`，应在与目标相同的系统/架构上构建。

## 6. 配置 CPA

把以下内容合并进 CPA `config.yaml`，不要覆盖无关配置：

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
      restart_mode: auto
      restart_service: ""
```

标准 CPA 轻量面板位于 `<CPA_HOME>/static/management.html` 时，`panel_path` 可以留空。

CPA 由特殊启动器启动或工作目录不固定时，请使用绝对路径：

```yaml
      panel_path: "/absolute/path/to/management.html"
      host_config_path: "/absolute/path/to/config.yaml"
```

Windows 路径应加引号，并使用正斜杠或转义后的反斜杠。

### 安全重启控件

插件卡片不会调用公开重启 API，也不会读取 Management Key。它生成随机的一次性 `restart_request`，打开 CPAMP 原生“编辑配置”抽屉，再通过当前已认证的管理会话保存该值；插件会留出保存响应返回的时间后再安排重启。

| `restart_mode` | 行为 | 安全使用条件 |
| --- | --- | --- |
| `broker` | 为 root 持有的 bootstrap broker 写入请求 | 由 bootstrap 自动设置；只有 `/etc/cpamp-theme-studio/bootstrap.json` 及活动单元确实属于当前 CPA 服务时才可使用 |
| `auto` | 默认值；在 Linux 自动发现并校验当前 systemd 服务 | 只有该服务的 `MainPID` 等于当前 CPA PID 才接受 |
| `systemd` | 使用 systemd，可配合 `restart_service` | 服务名先做格式校验，且仍必须通过相同的 `MainPID` 校验 |
| `self-exit` | 让 CPA 以退出码 `75` 结束 | 只在 Docker 或其他外部 supervisor 已确认会拉起该 CPA 进程时使用 |
| `disabled` | 禁用自动重启 | 适用于手工进程或没有安全 supervisor 的部署 |

`restart_request` 是 CPA 通用插件配置表单会显示的内部一次性字段，不要预填、复用或手工修改。`broker` 只写 `/run/cpamp-theme-studio/restart.request`，插件本身不会执行 `systemctl`。在 Windows、macOS、无 systemd 的容器或由包装器启动的服务中，`auto` 提示不可用是正确结果。只有验证 supervisor 的重启策略后才能配置 `self-exit`；没有 supervisor 时，它只会停止 CPA。

## 7. 外置 PANEL_PATH 的 Manager Server

只有 CPA 与 Manager Server 能看到同一个可写文件时才支持：

1. 下载与 Manager Server 版本一致的 `management.html`。
2. 让 Manager Server 的 `PANEL_PATH` 指向该文件。
3. 将同一个文件挂载或共享到 CPA 文件系统。
4. 插件 `panel_path` 填写 CPA 看到的路径。
5. 使用 Linux bootstrap 时，必须传入准确的 Manager 公网 `--panel-url`；bootstrap 会在注入前将它与该文件比对。
6. 重启 CPA 和 Manager Server。

两个服务位于不同容器时，应把同一个宿主机文件挂载进两个容器。容器内路径可以不同，但必须对应同一个宿主文件。

插件本身不能注入只有内嵌面板的 Manager Server。Linux/systemd 直接部署可以由 bootstrap 外置：传入活动公网 `--panel-url`，让 bootstrap 生成 `/var/lib/cpamp-theme-studio/panel/management.html`，并检查它创建的 Manager 服务 `PANEL_PATH` drop-in。该流程会备份、重启并验证两个服务。容器或非 systemd Manager 仍需手工配置共享的外置面板。

## 8. 验证

CPA 启动日志应同时出现：

```text
pluginhost: plugin loaded plugin_id=cpamp-theme-studio
pluginhost: plugin registered plugin_id=cpamp-theme-studio
```

验证公开的只读插件资源：

```bash
curl -fsS http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio >/dev/null
curl -fsS 'http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio?asset=loader' >/dev/null
curl -fsS 'http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio?asset=font-regular' >/dev/null
curl -fsS 'http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio?asset=font-semibold' >/dev/null
curl -fsS 'http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio?asset=restart-status' >/dev/null
```

再用浏览器确认：

- CPAMP 右上角原有的“主题”控件被标记为主题工作室对话框入口，并保留原生按钮样式。
- 左侧没有 `Theme Studio` 菜单，也不会创建悬浮启动器。
- 编辑器能反复打开、切换主题，并通过 X、遮罩和 Escape 关闭，键盘焦点能够恢复。
- 切换配色后刷新仍保持。
- 浏览器计算字体以 `JetBrains Mono` 开头，两个内置字重都加载成功。
- 每轮都只有一个 mount、宿主“主题”控件和 stage，关闭后 `body` overflow 恢复。
- 浏览器开发者工具没有主题插件报错。
- Theme Studio 的已安装行和商店卡片各出现一个“重启 CPA”控件；取消二次确认不会改变 CPA 进程或配置。
- 已配置自动重启时，确认后匿名 `process_instance` 会变化，loader 恢复后页面自动刷新到可用登录页/主页。不能只看 PID 或资源状态，还要检查真实面板。
- 文件中 start/end 标记各且仅有一个。

```text
<!-- cpamp-theme-studio:start -->
<!-- cpamp-theme-studio:end -->
```

不能只看标记就判定成功；插件注册、资源响应和真实浏览器执行都要通过。

## 9. 升级

1. 下载并校验新压缩包，保留旧压缩包/动态库。
2. CPA 仍在运行时，设置 `plugins.configs.cpamp-theme-studio.enabled: false`，等待标记消失且插件资源返回 404。
3. 停止 CPA；Windows 下已加载 DLL 不能安全覆盖。
4. 替换插件动态库，并重新启用插件配置。
5. 启动 CPA，重新执行全部验证。
6. 确认当前 CPAMP 面板仍只有一个标记块。

安装 bootstrap 后升级到 `0.2.0` 时，市场写入应自动重启 CPA。确认注入 loader URL 以 `v=0.2.0` 结尾，broker 状态把新版本记录为 accepted，浏览器主题偏好继续保留，CPAMP 右上角原生“主题”控件仍是唯一编辑器入口，“字体”和“界面密度”仍可滚动访问。未安装 bootstrap 时应手工重启 CPA，或使用单独验证过的 supervisor。

CPAMP 更新面板不需要重装插件。`management.html` 被覆盖后，watcher 应在 `watch_seconds` 内恢复 loader。

## 10. 回滚

bootstrap broker 已记录旧 accepted 版本时，若升级无法通过进程、资源版本或面板标记验证，会自动回滚：只删除未通过的 Theme Studio 文件，只恢复其插件配置子树，重启 CPA，再验证旧部署。采取后续动作前先检查 `journalctl -u cpamp-theme-studio-restart.service`。

手工部署的回滚步骤：

1. 在 CPA 配置中停用 `cpamp-theme-studio`，等待标记消失。
2. 停止 CPA。
3. 恢复旧插件动态库和旧插件配置。
4. 启动 CPA 并验证旧版本。

热停用时清理是确定性的。进程正常关闭时，CPA 可能在原生插件的异步 shutdown 完成前退出，因此清理仅为 best effort。若 CPA 已退出或崩溃导致未清理，可恢复面板备份，或只删除从 start 到 end 的三行。不要删除 `management.html` 中其他脚本。

若要移除整个 bootstrap 并恢复首次 apply 前快照，请使用 apply 输出的备份 ID，参见[Linux bootstrap 回滚](BOOTSTRAP.zh-CN.md#回滚)。

## 11. 卸载

1. 设置 `plugins.configs.cpamp-theme-studio.enabled: false`。
2. 等待标记消失且插件资源返回 404。
3. 停止 CPA。
4. 只删除 `cpamp-theme-studio` 动态库。
5. 如有需要再删除插件配置块，然后启动 CPA。

浏览器本地偏好可以保留，不会影响安全，重装后还能恢复。只有需要彻底重置偏好时才手工清除。
