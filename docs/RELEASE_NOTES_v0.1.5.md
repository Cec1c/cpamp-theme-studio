# v0.1.5 — CPAMP-native Theme Studio scrollbar

## English

This release removes the browser-default scrollbar from the Theme Studio drawer and adopts CPAMP's own plugin-host scrollbar treatment.

- Uses CPAMP's theme-aware primary/border color mix instead of a fixed scrollbar color.
- Matches CPAMP's Chromium treatment: a 10 px scrollbar box, 2 px transparent thumb border, fully rounded thumb, padding-box clipping, and transparent track.
- Uses native `scrollbar-width: thin` and matching `scrollbar-color` in Firefox.
- Keeps the styling inside Theme Studio's Shadow DOM, where CPAMP's global selectors cannot otherwise reach it.
- Adds runtime-contract assertions so future loader changes cannot silently restore the browser-default scrollbar.

Validated on the authenticated CPAMP v1.12.2 dashboard with CPA v7.2.138. Light and dark themes produced theme-aware computed scrollbar colors; the editor scrolled from top to bottom, closed, and reopened normally. At 375×667, the drawer had no horizontal overflow, Typography and Density remained reachable, and the fixed footer stayed unobscured.

Upgrade through the CPAMP Plugin Store to `0.1.5`, restart the effective CPA process from the confirmed card control (or manually when automatic restart is unavailable), and return to the dashboard's existing top-right Theme control.

## 简体中文

此版本移除主题工作室抽屉中的浏览器默认滚动条，改为复用 CPAMP 自己的插件宿主滚动条视觉语言。

- 使用 CPAMP 的主题强调色与边框色混合，不再写死滚动条颜色。
- 与 CPAMP 的 Chromium 样式一致：10 px 滚动区域、2 px 透明滑块边框、全圆角滑块、`padding-box` 裁剪和透明轨道。
- Firefox 使用原生 `scrollbar-width: thin` 及配套 `scrollbar-color`。
- 样式被放入主题工作室的 Shadow DOM；CPAMP 全局选择器原本无法穿透该边界。
- 增加运行时契约断言，避免后续 loader 修改又悄悄退回浏览器默认滚动条。

已在 CPA v7.2.138、CPAMP v1.12.2 的登录后真实主页验证。亮色与深色都得到随主题变化的实际计算颜色；编辑器可以从顶部滚动到底部，关闭并再次打开正常。375×667 下没有横向溢出，“字体”和“界面密度”均可访问，固定底栏没有遮挡内容。

请通过 CPAMP 插件市场升级到 `0.1.5`，使用插件卡片的二次确认控件重启实际 CPA 进程；若自动重启不可用则手工重启。完成后仍从仪表盘右上角原有的“主题”控件进入。
