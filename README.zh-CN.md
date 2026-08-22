# CPAMP 主题工作室

[English](README.md) · [部署文档](docs/DEPLOYMENT.zh-CN.md) · [Agent 部署手册](docs/AGENT_DEPLOYMENT.md) · [v0.1.2 发布说明](docs/RELEASE_NOTES_v0.1.2.md)

CPAMP 主题工作室是一个通过 CPAMP 插件市场交付的前端主题扩展，为可写的 [CPA Manager Plus](https://github.com/seakee/CPA-Manager-Plus) 面板增加可持久化的视觉主题编辑器。功能不再依赖 CPAMP 上游 PR，也不需要长期维护 CPAMP fork。

主题、编辑器和持久化逻辑都在浏览器端的 `assets/loader.js` 中。由于当前 CPAMP 的插件市场和启停复用 CPA 插件 API，发行包仍包含一个最小原生桥：它只注册隐藏的浏览器资源路由，并在 `management.html` 中维护一段带唯一标记的 `<script>`；不参与模型请求、Provider 或流量路由。上游面板更新覆盖文件后会自动恢复注入，停用插件时只移除自己的标记块。

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
- 六档圆角、三档界面密度，以及内置 JetBrains Mono 与中文回退字体。
- 铺满或居中的桌面内容布局。
- 完整视觉效果或性能优先模式。
- 即时预览、本地持久化、旧偏好迁移、一键重置。
- 简体中文、繁体中文、英语、俄语。
- Shadow DOM 隔离、键盘焦点处理、减少动画支持和移动端布局。
- 复用 CPAMP 右上角原生“主题”控件作为唯一入口，不再生成悬浮按钮或插件页/侧栏菜单。
- SPA 重入和 CPAMP 更新覆盖面板后的幂等运行时恢复与重注入。
- 热停用时确定性清理；进程正常关闭时仅尽力清理。

## 部署模式

| 模式 | 支持情况 | 要求 |
| --- | --- | --- |
| CPA `:8317` 托管的 CPAMP 轻量面板 | 支持 | CPA 需要可写的 `static/management.html` |
| 使用外置 `PANEL_PATH` 的 CPAMP Manager Server | 有条件支持 | CPA 与 Manager Server 必须共享同一个可写文件，并配置插件 `panel_path` |
| 只有内嵌面板的 Manager Server | 无法持久注入 | 面板编译在另一个进程中，必须先改用外置 `PANEL_PATH` |

插件运行在 CPA 内，不能改写另一个 Manager Server 可执行文件或容器镜像内嵌的只读面板。

## 推荐安装：CPAMP 插件市场

项目使用本仓库提供的社区商店源，不需要向 CPAMP 或 CPA 上游提交 PR。把以下字段合并到 CPA 的有效 `config.yaml`；`dir` 建议使用绝对路径，避免 systemd 工作目录不同导致插件装到了错误位置：

```yaml
plugins:
  enabled: true
  dir: "/absolute/path/to/cpa/plugins"
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

保存并重载配置后：

1. 打开 CPAMP 的“插件”→“插件商店”。
2. 确认来源中出现 `raw.githubusercontent.com`，搜索 `CPAMP Theme Studio`。
3. 选择 Latest 或 `0.1.2` 并安装；CPAMP/CPA 会下载清单固定的平台 Release 压缩包，校验商店清单携带的 SHA-256，把带版本号的动态库写到 `<dir>/<goos>/<goarch>/`，并创建启用配置。
4. 每次安装或版本升级后都要重启实际运行的 CPA 服务；然后在“已安装插件”确认 `registered` 与 `effective enabled`。不要只依赖热重载：旧插件版本的面板 watcher 可能持续到进程退出。
5. 回到 CPAMP 仪表盘，点击右上角操作区原有的“主题”控件；插件会替换该控件的点击行为，不再添加悬浮按钮。

注入 loader 和内置字体使用的隐藏只读资源为：

```text
/v0/resource/plugins/cpamp-theme-studio/studio
```

若 CPA 访问 GitHub 必须走代理，请在启动该服务的环境中设置代理变量；`127.0.0.1:7890` 只有在代理与 CPA 位于同一网络命名空间时才有效。生产部署、市场安装验证、手工回退、升级、回滚和卸载请阅读[部署文档](docs/DEPLOYMENT.zh-CN.md)。

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

## 插件商店契约

本仓库是 CPAMP 可添加的社区插件源：

```yaml
plugins:
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

`registry.json` 使用 CPA schema v2 直链资产：每个系统/架构的 URL、字节数和 SHA-256 都固定到不可变的 GitHub Release，因此市场安装不依赖 GitHub REST API 的匿名配额。CPA 会把 ZIP 根目录中的动态库安装为 `cpamp-theme-studio-v<version>.<ext>`。Release 工作流会从实际产物生成六个平台压缩包、`checksums.txt` 和已核验的候选商店清单。

## 从源码构建

要求 Go 1.26+ 和本机 CGO C 编译器；Node.js 24+ 仅用于 loader 语法检查。CGO 动态库在每个系统/架构的原生 runner 上构建。

Windows PowerShell：

```powershell
.\scripts\build.ps1 -Version 0.1.2-dev
.\scripts\package.ps1 -Version 0.1.2-dev
```

Linux 或 macOS：

```bash
./scripts/build.sh 0.1.2-dev
./scripts/package.sh 0.1.2-dev
```

动态库和压缩包输出到 `dist/`，不会提交进 Git。

## 工作原理

1. CPA 通过插件 ABI v1 加载动态库。
2. 插件注册一个供 loader 和字体使用的隐藏只读资源路由，不发布 CPAMP 侧栏菜单。
3. 注入器寻找可信的可写 `management.html`，在 `</head>` 前加入唯一 start/end 标记块。
4. loader 用 Shadow DOM 挂载编辑器，通过 CPAMP CSS 变量和兼容的本地主题存储应用设置。
5. 上游更新覆盖文件后自动恢复标记；热停用会确定性移除标记。进程关闭时，CPA 可能在原生插件的异步 shutdown 完成前退出，因此清理仅为 best effort。

插件不会拦截 CPA 请求、接管进程信号、代理凭证或公开任意文件。

升级、回滚或卸载前，应先热停用本插件并等待标记消失，再停止 CPA。若 CPA 已退出但标记仍在，请按部署文档恢复面板备份，或只删除本插件的标记块。

## 已执行验证

- `go test ./...`、`go test -race ./...` 和 `go vet ./...`
- `node --check assets/loader.js`
- Windows/amd64 DLL、Linux/amd64 `.so` 原生构建与打包
- Windows/amd64 和 Linux/amd64 上的 CPA v7.2.138 真实插件发现、注册、隐藏菜单契约和资源响应
- CPAMP v1.12.2 官方面板校验与注入
- 连续 5 轮打开、换主题、X/遮罩/Escape 关闭；刷新并重新登录后再做 3 轮
- mount、stage、宿主主题控件始终各 1 个，关闭后 body 滚动恢复，SPA 路由重入正常，侧栏入口为 0
- JetBrains Mono Regular/SemiBold 资源响应及浏览器实际字体加载
- 配色刷新持久化和 390×844 响应式布局
- 面板被官方文件覆盖后的自动重注入
- 热停用后精确恢复原始 CPAMP SHA-256

## 许可与来源

本项目采用 MIT License。CPA ABI、CPAMP 原型、YAML 依赖、内置 JetBrains Mono 许可与配色来源说明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。插件不分发 CPAMP `management.html`，也不包含 AGPL `new-api` 的代码或资源。
