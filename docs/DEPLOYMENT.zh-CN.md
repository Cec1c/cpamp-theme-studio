# 部署、升级、回滚与卸载

[English](DEPLOYMENT.md) · [Agent 部署手册](AGENT_DEPLOYMENT.md)

本文将 CPAMP 主题工作室部署到现有 CPA。它不会安装 CPA/CPAMP，也不会开放新的管理端口。

## 1. 收集参数

修改前先确认：

- CPA 主目录/工作目录。
- CPA `config.yaml` 路径。
- CPA 插件目录，默认是 `plugins`。
- 系统与架构：`windows`、`linux`、`darwin`；`amd64` 或 `arm64`。
- 面板模式：CPA 轻量面板，或使用外置 `PANEL_PATH` 的 Manager Server。
- 可写 `management.html` 的准确路径。
- 需要部署的插件版本。

不要输出或提交 CPA Management Key；插件不需要它。

## 2. 部署前检查

1. CPA 为 v7.2.138，或已在本环境验证过的版本。
2. 面板为 CPAMP v1.12.2，或已在本环境验证过的版本。
3. `management.html` 是可写的普通文件，不是符号链接，且不超过 64 MiB。
4. 首次部署前备份 CPA `config.yaml`、当前插件动态库和面板文件。
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

## 3. 从 Release 安装

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

CPA 也接受 `cpamp-theme-studio-v0.1.0.dll` 这类带版本名。不要保留多个无版本文件副本。

## 4. 从源码构建

首个 Release 之前，或需要审计服务器所部署的准确源码时，使用此方式。

Windows：

```powershell
git clone https://github.com/Cec1c/cpamp-theme-studio.git
Set-Location .\cpamp-theme-studio
go test ./...
node --check .\assets\loader.js
.\scripts\package.ps1 -Version 0.1.0-dev
```

Linux/macOS：

```bash
git clone https://github.com/Cec1c/cpamp-theme-studio.git
cd cpamp-theme-studio
go test ./...
node --check assets/loader.js
./scripts/package.sh 0.1.0-dev
```

需要 Go 1.26+ 和本机 C 编译器。插件使用 CGO `c-shared`，应在与目标相同的系统/架构上构建。

## 5. 配置 CPA

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
```

标准 CPA 轻量面板位于 `<CPA_HOME>/static/management.html` 时，`panel_path` 可以留空。

CPA 由特殊启动器启动或工作目录不固定时，请使用绝对路径：

```yaml
      panel_path: "/absolute/path/to/management.html"
      host_config_path: "/absolute/path/to/config.yaml"
```

Windows 路径应加引号，并使用正斜杠或转义后的反斜杠。

## 6. 外置 PANEL_PATH 的 Manager Server

只有 CPA 与 Manager Server 能看到同一个可写文件时才支持：

1. 下载与 Manager Server 版本一致的 `management.html`。
2. 让 Manager Server 的 `PANEL_PATH` 指向该文件。
3. 将同一个文件挂载或共享到 CPA 文件系统。
4. 插件 `panel_path` 填写 CPA 看到的路径。
5. 重启 CPA 和 Manager Server。

两个服务位于不同容器时，应把同一个宿主机文件挂载进两个容器。容器内路径可以不同，但必须对应同一个宿主文件。

只有内嵌面板的 Manager Server 无法注入，不应宣称该模式可在登录页持久生效。

## 7. 验证

CPA 启动日志应同时出现：

```text
pluginhost: plugin loaded plugin_id=cpamp-theme-studio
pluginhost: plugin registered plugin_id=cpamp-theme-studio
```

验证公开的只读插件资源：

```bash
curl -fsS http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio >/dev/null
curl -fsS 'http://127.0.0.1:8317/v0/resource/plugins/cpamp-theme-studio/studio?asset=loader' >/dev/null
```

再用浏览器确认：

- 右下角出现主题工作室按钮。
- 编辑器可打开/关闭，键盘焦点能够恢复。
- 切换配色后刷新仍保持。
- 浏览器开发者工具没有主题插件报错。
- 文件中 start/end 标记各且仅有一个。

```text
<!-- cpamp-theme-studio:start -->
<!-- cpamp-theme-studio:end -->
```

不能只看标记就判定成功；插件注册、资源响应和真实浏览器执行都要通过。

## 8. 升级

1. 下载并校验新压缩包，保留旧压缩包/动态库。
2. CPA 仍在运行时，设置 `plugins.configs.cpamp-theme-studio.enabled: false`，等待标记消失且插件资源返回 404。
3. 停止 CPA；Windows 下已加载 DLL 不能安全覆盖。
4. 替换插件动态库，并重新启用插件配置。
5. 启动 CPA，重新执行全部验证。
6. 确认当前 CPAMP 面板仍只有一个标记块。

CPAMP 更新面板不需要重装插件。`management.html` 被覆盖后，watcher 应在 `watch_seconds` 内恢复 loader。

## 9. 回滚

1. 在 CPA 配置中停用 `cpamp-theme-studio`，等待标记消失。
2. 停止 CPA。
3. 恢复旧插件动态库和旧插件配置。
4. 启动 CPA 并验证旧版本。

热停用时清理是确定性的。进程正常关闭时，CPA 可能在原生插件的异步 shutdown 完成前退出，因此清理仅为 best effort。若 CPA 已退出或崩溃导致未清理，可恢复面板备份，或只删除从 start 到 end 的三行。不要删除 `management.html` 中其他脚本。

## 10. 卸载

1. 设置 `plugins.configs.cpamp-theme-studio.enabled: false`。
2. 等待标记消失且插件资源返回 404。
3. 停止 CPA。
4. 只删除 `cpamp-theme-studio` 动态库。
5. 如有需要再删除插件配置块，然后启动 CPA。

浏览器本地偏好可以保留，不会影响安全，重装后还能恢复。只有需要彻底重置偏好时才手工清除。
