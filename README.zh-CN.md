# CPAMP 主题工作室

[English](README.md) · [Linux bootstrap](docs/BOOTSTRAP.zh-CN.md) · [部署文档](docs/DEPLOYMENT.zh-CN.md) · [Agent 部署手册](docs/AGENT_DEPLOYMENT.md) · [v0.2.0 发布说明](docs/RELEASE_NOTES_v0.2.0.md)

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
- 六档圆角、三档界面密度，以及可选的 CPAMP 默认、内置 JetBrains Mono 或系统无衬线字体。
- 在隔离的主题抽屉内复用 CPAMP 插件宿主的滚动条语言：Firefox 使用细滚动条，Chromium 使用透明轨道与胶囊形滑块。
- 铺满或居中的桌面内容布局。
- 完整视觉效果或性能优先模式。
- 即时预览、本地持久化、旧偏好迁移、一键重置。
- 简体中文、繁体中文、英语、俄语。
- Shadow DOM 隔离、键盘焦点处理、减少动画支持和移动端布局。
- 复用 CPAMP 右上角原生“主题”控件作为唯一入口，不再生成悬浮按钮或插件页/侧栏菜单。
- 在 Theme Studio 原生三点菜单的“重新安装”上方提供带二次确认的“重新启动”项；一次 Linux bootstrap 后，该操作和后续市场安装可以在不经过部署 Agent 的情况下自动重启、验证并失败回滚。
- SPA 重入和 CPAMP 更新覆盖面板后的幂等运行时恢复与重注入。
- 热停用时确定性清理；进程正常关闭时仅尽力清理。

## 部署模式

| 模式 | 支持情况 | 要求 |
| --- | --- | --- |
| CPA `:8317` 托管的 CPAMP 轻量面板 | 支持 | CPA 需要可写的 `static/management.html` |
| 使用外置 `PANEL_PATH` 的 CPAMP Manager Server | 有条件支持 | CPA 与 Manager Server 必须共享同一个可写文件，配置插件 `panel_path`，并向 bootstrap 提供准确的公网 `--panel-url` |
| 只有内嵌面板的 Manager Server | 可由 Linux bootstrap 外置 | 提供活动公网 `--panel-url`；bootstrap 会创建并绑定可写 `PANEL_PATH` |

插件运行在 CPA 内，不能直接改写另一个进程的内嵌面板。Linux bootstrap 可以先下载活动面板，让两个服务绑定到外置文件，再验证公网结果。

## 推荐安装：CPAMP 插件市场

项目使用本仓库提供的社区商店源，不需要向 CPAMP 或 CPA 上游提交 PR。Linux/systemd 直接部署建议先执行管理员核对过的[一次性 bootstrap](docs/BOOTSTRAP.zh-CN.md)。市场插件不能安全地自行获得 root；但完成 bootstrap 后，后续安装/升级和卡片确认重启都由事务化流程自动完成。

未安装 bootstrap 时，把以下字段合并到 CPA 的有效 `config.yaml`，并准备手工重启 CPA；`dir` 建议使用绝对路径，避免 systemd 工作目录不同导致插件装到了错误位置：

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
3. 选择 Latest 或 `0.2.0` 并安装；CPAMP/CPA 会下载清单固定的平台 Release 压缩包，校验商店清单携带的 SHA-256，把带版本号的动态库写到 `<dir>/<goos>/<goarch>/`，并创建启用配置。
4. 已安装 bootstrap 时，市场写入本身就会触发绑定到 CPA unit 的 systemd broker。它等待动态库/配置稳定，重启 CPA，验证目标版本与真实面板后才接受；失败时删除未通过的部署并恢复上一个 accepted 插件/配置。Theme Studio 三点菜单中带确认的“重新启动”项复用同一个 broker。未安装 bootstrap 或其他平台仍需手工重启实际 CPA 服务。
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
| `restart_mode` | `auto` | Linux bootstrap 后为 `broker`；其他情况可用 `auto`、`disabled`、`systemd` 或 `self-exit` |
| `restart_service` | 空 | 可选 systemd 服务覆盖；只有其 `MainPID` 等于当前 CPA PID 时才接受 |
| `restart_request` | 内部字段 | 已认证卡片控件写入的一次性值；不要手工修改 |

`panel_path` 为空时，插件先从 `CPAMP_THEME_PANEL_PATH`、`MANAGEMENT_STATIC_PATH`、`PANEL_PATH` 中接受唯一且无冲突的显式路径；否则只接受 CPA 工作目录或可执行文件附近唯一存在的 `static/management.html` / `management.html`。若路径冲突或候选不唯一，注入会停止，不再猜测。

## 插件商店契约

本仓库是 CPAMP 可添加的社区插件源：

```yaml
plugins:
  store-sources:
    - "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"
```

`registry.json` 使用 CPA schema v2 直链资产：每个系统/架构的 URL、字节数和 SHA-256 都固定到不可变的 GitHub Release，因此市场安装不依赖 GitHub REST API 的匿名配额。CPA 会把 ZIP 根目录中的动态库安装为 `cpamp-theme-studio-v<version>.<ext>`。Release 工作流会从实际产物生成六个平台压缩包、独立的 `bootstrap-linux.sh`、`checksums.txt` 和已核验的候选商店清单；Linux 包还包含静态 bootstrap 二进制与下载器。

## 从源码构建

要求 Go 1.26+ 和本机 CGO C 编译器；Node.js 24+ 仅用于 loader 语法检查。CGO 动态库在每个系统/架构的原生 runner 上构建。

Windows PowerShell：

```powershell
.\scripts\build.ps1 -Version 0.2.0-dev
.\scripts\package.ps1 -Version 0.2.0-dev
```

Linux 或 macOS：

```bash
./scripts/build.sh 0.2.0-dev
./scripts/package.sh 0.2.0-dev
```

动态库和压缩包输出到 `dist/`，不会提交进 Git。

## 工作原理

1. CPA 通过插件 ABI v1 加载动态库。
2. 插件注册一个供 loader 和字体使用的隐藏只读资源路由，不发布 CPAMP 侧栏菜单。
3. 注入器寻找可信的可写 `management.html`，在 `</head>` 前加入唯一 start/end 标记块。
4. loader 用 Shadow DOM 挂载编辑器，通过 CPAMP CSS 变量和兼容的本地主题存储应用设置。
5. 完成一次 Linux bootstrap 后，市场文件变化与卡片确认请求由绑定到唯一已核验 CPA unit 的 root broker 处理；它执行进程、资源和面板验收，失败时恢复上一个 accepted 版本。浏览器和插件都不会获得 systemd 权限。
6. 未安装 bootstrap 时，控件仍可使用旧的 PID 校验 systemd/self-exit 模式，或明确提示需要手工重启。
7. 上游更新覆盖文件后自动恢复标记；热停用会确定性移除标记。进程关闭时，CPA 可能在原生插件的异步 shutdown 完成前退出，因此清理仅为 best effort。

插件不会拦截 CPA 请求、读取或代理凭证、公开任意文件，也不会发布未认证的重启接口。

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
- CPAMP 默认、JetBrains Mono、系统无衬线三种字体切换及刷新持久化
- 亮色/深色下与 CPAMP 插件宿主一致的滚动条，以及 375×667 无横向溢出验证
- 1280×720 与 375×667 下主题工作室可滚动，界面密度控件可达，且无持续横向溢出
- 配色刷新持久化和 390×844 响应式布局
- 面板被官方文件覆盖后的自动重注入
- 热停用后精确恢复原始 CPAMP SHA-256
- 位于“重新安装”上方的原生三点菜单重启项、安全取消、一次性请求持久化、supervisor 拉起新进程、loader 恢复与自动刷新
- 二次确认焦点处理、宿主原生菜单样式与图标、减少动画，以及 375×812 下无横向溢出
- 在真实 systemd 管理的 CPA v7.2.138 测试环境完成 Linux bootstrap dry-run/apply
- 市场文件/配置变化触发自动重启，并通过插件注册、资源版本和单一面板标记验收
- 卡片确认 broker 重启；故意损坏 `.so` 的升级验证失败后，自动恢复旧插件/配置并确认旧版本重新工作

## 许可与来源

本项目采用 MIT License。CPA ABI、CPAMP 原型、YAML 依赖、内置 JetBrains Mono 许可与配色来源说明见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。插件不分发 CPAMP `management.html`，也不包含 AGPL `new-api` 的代码或资源。
