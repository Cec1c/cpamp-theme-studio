# v0.1.4 — Safe restart from the plugin card

## English

This release closes the hot-upgrade lifecycle gap: Theme Studio can now request a safe CPA restart directly from its CPAMP plugin card.

- Adds a `Restart CPA` control to both the installed-plugin row and the Theme Studio card in the Plugin Store.
- Requires an accessible second confirmation before interrupting CPA. Cancel is focused by default; Escape, the scrim, and the Cancel button all dismiss safely.
- Reuses CPAMP's authenticated Edit configuration → Save flow and never reads, captures, stores, or forwards the Management Key.
- Publishes no unauthenticated restart endpoint. The browser can only write a one-time `restart_request` through CPAMP's existing authenticated management UI.
- On Linux/systemd, auto-discovery accepts a service only when `systemctl show ... MainPID` matches the currently running CPA process.
- Supports an explicit `self-exit` mode with exit code `75` for Docker or another external supervisor. Do not enable it unless that supervisor is configured to restart CPA.
- Supports `disabled` mode and an explicit systemd service override.
- Waits for a new process instance and for the injected panel loader to return before refreshing the page.
- Shows non-blocking, screen-reader-announced error feedback on failure or timeout.
- Keeps 44 px touch targets, keyboard focus trapping, reduced-motion handling, light/dark styling, and a 375 px layout without horizontal overflow.
- Includes Simplified Chinese, Traditional Chinese, English, and Russian restart copy.

Validated with CPA v7.2.138 and CPAMP v1.12.2 in an isolated supervised environment. Cancellation preserved the CPA PID and exact configuration SHA-256. Confirmation wrote one request, exited CPA with code `75`, started a new CPA process, changed the opaque `process_instance`, restored the injected loader, and refreshed to a usable panel.

Upgrade through the CPAMP Plugin Store to `0.1.4`. Once the new card control appears, choose `Restart CPA` and confirm. If the deployment reports that automatic restart is unavailable, configure a validated systemd unit, explicitly use `self-exit` under a real supervisor, or restart CPA manually.

## 简体中文

此版本补齐热升级后的生命周期缺口：主题工作室现在可以从 CPAMP 插件卡片安全请求重启 CPA。

- 在“已安装插件”行和插件商店的 Theme Studio 卡片上新增“重启 CPA”控件。
- 中断 CPA 前必须进行无障碍二次确认；默认焦点位于“取消”，Escape、遮罩和取消按钮都能安全退出。
- 复用 CPAMP 已认证的“编辑配置 → 保存”链路，不读取、截获、保存或转发 Management Key。
- 不发布未认证的重启接口；浏览器只能通过 CPAMP 现有认证管理界面写入一次性 `restart_request`。
- Linux/systemd 自动发现只接受 `systemctl show ... MainPID` 与当前 CPA 进程一致的服务。
- 为 Docker 或其他外部 supervisor 提供显式 `self-exit` 模式，退出码为 `75`；只有 supervisor 已配置自动拉起 CPA 时才能启用。
- 支持 `disabled` 模式和显式 systemd 服务覆盖。
- 自动等待新的进程实例和面板 loader 重新注入成功，再刷新页面。
- 失败或超时时使用不阻塞操作、可由屏幕阅读器播报的错误提示。
- 保持 44 px 触控区、键盘焦点循环、减少动画、深浅色适配，以及 375 px 下无横向溢出。
- 重启文案支持简体中文、繁体中文、英语和俄语。

已在 CPA v7.2.138、CPAMP v1.12.2 的隔离 supervisor 环境验证。取消操作不会改变 CPA PID，也不会改变配置文件 SHA-256；确认后只写入一次请求，CPA 以退出码 `75` 结束，由 supervisor 拉起新进程，匿名 `process_instance` 随之变化，loader 恢复注入后页面自动刷新并可继续使用。

通过 CPAMP 插件市场升级到 `0.1.4`。新卡片控件出现后点击“重启 CPA”并确认。如果页面提示当前部署无法自动重启，请配置经过 PID 校验的 systemd 服务、在真实 supervisor 下显式使用 `self-exit`，或手工重启 CPA。
