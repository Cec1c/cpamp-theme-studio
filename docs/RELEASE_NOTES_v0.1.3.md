# v0.1.3 — Typography choices and reliable editor scrolling

## English

This release fixes two visible Theme Studio regressions reported from the real CPAMP dashboard.

- Adds an explicit Typography section with CPAMP default, bundled JetBrains Mono, and system sans choices.
- Keeps JetBrains Mono as the default for new Theme Studio preferences while allowing existing users to switch immediately.
- Persists the selected font with the rest of the browser-side theme state.
- Constrains the right-side editor to the dynamic viewport and gives its content area a real shrinking vertical scroll region.
- Keeps the header, close control, and reset footer visible while long settings content scrolls independently.
- Prevents persistent horizontal overflow at phone widths.
- Adds automated contracts for all three font controls and the viewport/overflow rules.

Validated on the authenticated CPAMP v1.12.2 dashboard with CPA v7.2.138 at 1280×720 and 375×667. Typography changes, Density controls, reload persistence, the native top-right entry, and X closing were exercised in the real page.

Upgrade through the CPAMP Plugin Store to `0.1.3`, restart the effective CPA process, and refresh the panel once.

## 简体中文

此版本修复了从真实 CPAMP 主页发现的两个主题工作室回归问题。

- 新增明确的“字体”区域，可选 CPAMP 默认、内置 JetBrains Mono 和系统无衬线字体。
- 新主题偏好仍默认使用 JetBrains Mono，同时允许已有用户立即切换。
- 字体选择与其他浏览器侧主题状态一起持久化。
- 将右侧编辑器约束在动态视口内，并让内容区成为可正确收缩的纵向滚动区。
- 长内容滚动时，标题、关闭按钮和底部恢复按钮保持可见。
- 避免手机宽度下出现持续横向溢出。
- 为三种字体控件和视口/滚动约束补充自动化契约测试。

已在 CPA v7.2.138、CPAMP v1.12.2 的登录后真实主页，以 1280×720 和 375×667 视口验证字体切换、界面密度、刷新持久化、右上角原生入口和 X 关闭。

通过 CPAMP 插件市场升级到 `0.1.3` 后，重启实际 CPA 进程，再刷新面板一次。
