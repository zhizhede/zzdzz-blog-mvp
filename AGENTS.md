# zzdzz-blog 项目规则

## 浏览器自动化禁令(防止窗口被顶到最上层)

- **禁止使用 ZCode 内置可见浏览器面板**(iab 后端)打开本项目任何页面。该面板每次"绑定标签"都会强制置顶,干扰用户正常使用(2026-09-01 已实证)。
- 前端部署验证一律使用服务器端 `curl`(检查 HTML、资源指纹、接口状态码);交互逻辑验证优先在本地用单元测试覆盖。
- 仅当用户**明确要求**"截图 / 看视觉效果"时,才允许打开可见浏览器,且用完必须关闭所有标签页。
- 若必须做浏览器自动化,优先申请 headless 后端(`--browser-use=headless`,cdp),它没有可见窗口。
- 部署流程见 `deploy/README.md`;服务器前端产物路径是 `/www/wwwroot/blog-server/web`(由 Go 后端托管,`blog-ui/dist` 不生效)。

## 网站图标源图约定

- `pictures/` 恒定只放**一张**当前网站图标源图;旧图一律挪入 `pictures/bak/`(带时间戳前缀)。
- 更换源图用 `node scripts/set-icon.mjs <图片路径>`,自动备份旧图;不带参数可查看当前源图与备份列表。
- `scripts/build-favicon.mjs` 自动取 `pictures/` 内唯一图片为源,发现零张或多张会直接报错,不要往 `pictures/` 手工堆图。
- 线上自定义 favicon 走后台上传(`IconService`,存服务器 `data/icon`),与 `pictures/` 源图相互独立;`pictures/` 只影响内置回退图标的生成来源。
