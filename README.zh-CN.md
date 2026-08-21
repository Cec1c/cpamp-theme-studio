# CPAMP 主题工作室

[English](README.md) · [部署文档](docs/DEPLOYMENT.zh-CN.md) · [Agent 部署手册](docs/AGENT_DEPLOYMENT.md)

CPAMP 主题工作室是一个独立 CPA 插件，为可写的 [CPA Manager Plus](https://github.com/seakee/CPA-Manager-Plus) 面板增加可持久化的视觉主题编辑器。功能不再依赖 CPAMP 上游 PR，也不需要长期维护 CPAMP fork。

插件向 CPA 注册浏览器资源，在 `management.html` 中注入一段带唯一标记的 `<script>`。上游面板更新覆盖文件后会自动恢复注入；停用插件时只会移除自己写入的标记块。

## 兼容状态

首个版本以以下组合为兼容基线：

| 组件 | 已验证版本 | 状态 |
| --- | --- | --- |
| CLIProxyAPI / CPA | v7.2.138 | 已验证 ABI、注册、资源路由、热重载和停用生命周期 |
| CPA Manager Plus / CPAMP | v1.12.2 | 已验证官方 `management.html` 注入和真实浏览器行为 |
| Windows | amd64 | DLL 真实加载和浏览器测试通过 |
| Linux | amd64 | `.so` 真实加载、资源和面板注入测试通过 |
| Linux | arm64 | 原生 Release 构建目标；生产使用前应确认发布工作流通过 |
| macOS | amd64、arm64 | 原生 CI/Release 构建目标 |
| Windows | arm64 | 原生 Release 构建目标；生产使用前应确认发布工作流通过 |

只要后续 CPA 的插件 ABI 和 CPAMP 的单文件面板结构保持兼容，插件通常可以继续工作；但生产升级前仍应按文档重新验证。

## 功能

- 自动、浅色、深色三种显示模式。
- CPAMP 蓝及九套独立设计的配色。
- 自定义强调色。
- 六档圆角、三档界面密度、三种字体。
- 铺满或居中的桌面内容布局。
- 完整视觉效果或性能优先模式。
- 即时预览、本地持久化、旧偏好迁移、一键重置。
- 简体中文、繁体中文、英语、俄语。
- Shadow DOM 隔离、键盘焦点处理、减少动画支持和移动端布局。
- CPAMP 更新覆盖面板后的幂等重注入。
- 热停用时确定性清理；进程正常关闭时仅尽力清理。

## 部署模式

| 模式 | 支持情况 | 要求 |
| --- | --- | --- |
| CPA `:8317` 托管的 CPAMP 轻量面板 | 支持 | CPA 需要可写的 `static/management.html` |
| 使用外置 `PANEL_PATH` 的 CPAMP Manager Server | 有条件支持 | CPA 与 Manager Server 必须共享同一个可写文件，并配置插件 `panel_path` |
| 只有内嵌面板的 Manager Server | 无法持久注入 | 面板编译在另一个进程中，必须先改用外置 `PANEL_PATH` |

插件运行在 CPA 内，不能改写另一个 Manager Server 可执行文件或容器镜像内嵌的只读面板。

## 快速安装

1. 从 GitHub Releases 下载与系统和架构匹配的压缩包，并用 `checksums.txt` 校验。
2. 升级时先热停用现有插件并等待标记消失，再停止 CPA 后替换已加载的动态库；Windows 下尤其必须这样做。
3. 将动态库解压到 CPA 插件目录对应的平台子目录：

```text
plugins/
  windows/amd64/cpamp-theme-studio.dll
  windows/arm64/cpamp-theme-studio.dll
  linux/amd64/cpamp-theme-studio.so
  linux/arm64/cpamp-theme-studio.so
  darwin/amd64/cpamp-theme-studio.dylib
  darwin/arm64/cpamp-theme-studio.dylib
```

只需要保留与宿主平台匹配的一项。

4. 在 CPA `config.yaml` 中加入：

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

5. 启动 CPA 并打开 CPAMP 面板，右下角会出现主题工作室按钮。CPA 同时公开插件菜单资源：

```text
/v0/resource/plugins/cpamp-theme-studio/studio
```

生产部署、验证命令、升级、回滚和卸载请阅读[部署文档](docs/DEPLOYMENT.zh-CN.md)。

## CPAMP 轻量面板配置

主题插件不会自行下载 CPAMP。典型 CPA 配置如下：

```yaml
remote-management:
  allow-remote: false
  secret-key: "请替换为高强度管理密钥"
  disable-control-panel: false
  disable-auto-update-panel: false
  panel-github-repository: "https://github.com/seakee/CPA-Manager-Plus"
```

管理接口应只监听本机，或放在可信反向代理/VPN 后面。不要提交真实 Management Key。

## 插件配置

| 字段 | 默认值 | 作用 |
| --- | --- | --- |
| `auto_inject` | `true` | 监控并注入可写面板文件 |
| `panel_path` | 空 | 明确指定 `management.html` 或其目录；相对路径以 CPA 工作目录为准 |
| `host_config_path` | 自动发现 | CPA `config.yaml`，用于发现热停用并立即清理 |
| `watch_seconds` | `3` | 面板检查周期，限制在 1–300 秒 |

`panel_path` 为空时，插件会依次检查 `CPAMP_THEME_PANEL_PATH`、`MANAGEMENT_STATIC_PATH`、`PANEL_PATH`，以及 CPA 工作目录和可执行文件附近的 `static/management.html`、`management.html`。

## 插件商店源

仓库产生 GitHub Release 后，可把它加入 CPA 社区插件源：

```yaml
plugins:
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

CPA 会寻找 `cpamp-theme-studio_<version>_<goos>_<goarch>.zip` 和 `checksums.txt`；仓库内的 Release 工作流会生成这些产物。

## 从源码构建

要求 Go 1.26+ 和本机 CGO C 编译器；Node.js 24+ 仅用于 loader 语法检查。CGO 动态库在每个系统/架构的原生 runner 上构建。

Windows PowerShell：

```powershell
.\scripts\build.ps1 -Version 0.1.0-dev
.\scripts\package.ps1 -Version 0.1.0-dev
```

Linux 或 macOS：

```bash
./scripts/build.sh 0.1.0-dev
./scripts/package.sh 0.1.0-dev
```

动态库和压缩包输出到 `dist/`，不会提交进 Git。

## 工作原理

1. CPA 通过插件 ABI v1 加载动态库。
2. 插件注册一个只读管理资源，并返回内嵌的主题 loader JavaScript。
3. 注入器寻找可信的可写 `management.html`，在 `</head>` 前加入唯一 start/end 标记块。
4. loader 用 Shadow DOM 挂载编辑器，通过 CPAMP CSS 变量和兼容的本地主题存储应用设置。
5. 上游更新覆盖文件后自动恢复标记；热停用会确定性移除标记。进程关闭时，CPA 可能在原生插件的异步 shutdown 完成前退出，因此清理仅为 best effort。

插件不会拦截 CPA 请求、接管进程信号、代理凭证或公开任意文件。

升级、回滚或卸载前，应先热停用本插件并等待标记消失，再停止 CPA。若 CPA 已退出但标记仍在，请按部署文档恢复面板备份，或只删除本插件的标记块。

## 已执行验证

- `go test ./...`、`go test -race ./...` 和 `go vet ./...`
- `node --check assets/loader.js`
- Windows/amd64 DLL、Linux/amd64 `.so` 原生构建与打包
- Windows/amd64 和 Linux/amd64 上的 CPA v7.2.138 真实插件发现、注册、菜单和资源响应
- CPAMP v1.12.2 官方面板校验与注入
- 浏览器启动器、对话框内容、配色切换、刷新持久化、390×844 响应式布局
- 面板被官方文件覆盖后的自动重注入
- 热停用后精确恢复原始 CPAMP SHA-256

## 许可与来源

本项目采用 MIT License。CPA ABI、CPAMP 原型、YAML 依赖与配色来源说明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。插件不分发 CPAMP `management.html`，也不包含 AGPL `new-api` 的代码或资源。
