# v0.1.1 — Reliable re-entry and JetBrains Mono

## English

This release fixes the interaction failures reported after the first Theme Studio session.

- Keeps the lower-right launcher as the single entry point and removes the redundant `Theme Studio` sidebar page.
- Rebuilds stale or detached Shadow DOM mounts after CPAMP SPA changes.
- Uses the real stage visibility as the source of truth, so X, scrim, Escape, theme controls, and repeated reopening remain usable.
- Uses `composedPath()` for Shadow DOM event routing and restores focus and body scrolling after every close.
- Bundles JetBrains Mono 2.304 Regular and SemiBold; Chinese text falls back to PingFang SC or Microsoft YaHei.
- Retains existing saved palettes and other visual preferences. Legacy font-choice values are ignored because typography is now fixed to JetBrains Mono.

Validated locally with CPA v7.2.138 and CPAMP v1.12.2: five consecutive open/change/close cycles, three more after reload and login, X/scrim/Escape closing, SPA navigation, singleton mount checks, font loading, hidden-menu API response, and hot-disable cleanup.

Upgrade through the CPAMP Plugin Store to `0.1.1`, restart CPA if prompted, and refresh the panel once. The left sidebar item should be gone; use the lower-right launcher.

## 中文

本版本修复主题工作室首次使用后再次进入时出现的交互失效。

- 右下角启动器成为唯一入口，移除重复的 `Theme Studio` 左侧栏页面。
- CPAMP SPA 变更导致 Shadow DOM 节点失效或脱离时，运行时会自动重建挂载。
- 以 stage 的真实可见状态为准，X、遮罩、Escape、主题按钮和反复重开不再依赖容易失真的布尔标记。
- Shadow DOM 点击改用 `composedPath()` 路由，每次关闭都会恢复焦点和页面滚动。
- 内置 JetBrains Mono 2.304 Regular/SemiBold；中文回退到苹方或微软雅黑。
- 已保存的配色与其他视觉偏好继续保留；旧字体选项会被忽略，因为字体现已固定为 JetBrains Mono。

本地已在 CPA v7.2.138、CPAMP v1.12.2 上验证：连续 5 轮打开/换主题/关闭，刷新并重新登录后再做 3 轮，覆盖 X、遮罩、Escape、SPA 导航、单例挂载、字体加载、隐藏菜单接口和热停用清理。

请通过 CPAMP 插件市场升级到 `0.1.1`，按提示重启 CPA 后刷新一次面板。左侧入口应消失，后续统一使用右下角启动器。
