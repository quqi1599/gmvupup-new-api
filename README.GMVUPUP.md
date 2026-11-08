# GMVUPUP 官网定制

GMVUPUP / gmvupup.top 的独立 New API 前端工作目录。官方基线为 `v1.0.0-rc.48`（`706144769e5bed5a47b78721e3bb11da3cf4c0b9`），官网初始分支 `codex/gmvupup-homepage`；文档开发与最终验证使用独立验证目录的 `codex/gmvupup-docs`。

GitHub 账号已有官方同源 fork `quqi1599/new-api`；`gh repo fork --fork-name gmvupup-new-api` 返回已有 fork。因此本工作目录以 `origin` 指向现有 fork、`upstream` 指向 `QuantumNous/new-api`，与其他 New API 本地目录分开。GMVUPUP 使用独立发布分支 `codex/gmvupup-release`，通过 `.github/workflows/gmvupup-production.yml` 在既有 `ci-vps` runner 检查、构建并发布 `ghcr.io/quqi1599/gmvupup-new-api:sha-<commit前12位>`。生产按 digest 部署；不更新贵数旧站的 main 分支或镜像。

## 页面

- 官网入口仍是 `/`，默认首页呈现 GMVUPUP 品牌设计；保留 New API 原有的管理员自定义首页功能。
- 暖白、炭黑、荧光绿的品牌配色，双向上箭头标识、概念视觉、六模型 Logo、三种创作场景、API 代码示例、FAQ 和移动导航。
- 模型展示按用户指定保留完整名称：GPT-image-2.6、nanobanana-2.1、seedance-2.5、MiniMax-H3、Gemini-3.8、DeepSeek-V4.1-flash。通过既有 getLobeIcon 复用 @lobehub/icons 的 OpenAI、NanoBanana、ByteDance、Minimax、Gemini、DeepSeek 标识；Seedance 使用其提供方 ByteDance 标识。展示名称不会创建模型、渠道或价格。
- 注册、登录、模型价格、密钥管理和控制台均使用 New API 真实路由；不修改认证、渠道、计费或任务处理。
- 新增文案使用 i18next，提供中文和英文；其余界面的新增文案采用英文回退，保留原有译文。
- 页面概念图不宣称来自 GMVUPUP 线上服务，不使用未经证实的客户数量、可用率、销量或模型效果。
- 保留 New API 项目链接、前端作者署名、源码许可证和第三方声明。

## API 文档

入口 `/docs`，适配 GMVUPUP 的 Base URL `https://gmvupup.top/v1`。参考 AGToken 文档的信息结构，重做为本站暖白、深绿、荧光绿的三栏排版；手机端改为抽屉目录。

- 8 篇内容：快速开始、身份验证、模型目录、图像生成、图像编辑、对话生成、视频任务、错误与重试。
- 文档搜索、Enter 打开结果、页内导航、上下篇、Base URL 复制、cURL/Python/JavaScript 切换、示例模型切换、代码复制和下载。
- 公开模型目录复用 `/api/pricing`；令牌自己的可用范围通过 `GET /v1/models` 查询。代码示例使用本站 ID `gpt-image-2.6`、`nanobanana-2.1`、`deepseek-v4.1-flash`，不直接使用上游别名。
- 图像说明采用本站同步接口。视频页明确要求先开通对应模型；示例为已有 Doubao 适配器的 `doubao-seedance-2.5`，不是 CCG 插件协议，不宣称当前已可调用。
- `/gmvupup/docs/openapi.json` 提供 OpenAPI 3.1 文件，覆盖模型列表、图像生成/编辑和对话；`/gmvupup/docs/guide.md` 提供中文接入指南。
- 首页导航与页脚的 API 文档入口均指向 `/docs`；修复文档返回首页锚点时，首页异步加载后没有滚动到对应段落的问题。

源码：`web/src/features/gmvupup-docs/`、`web/src/routes/docs/`、`web/src/styles/gmvupup-docs.css`。验证与截图：工作区 `outputs/gmvupup-docs/`，报告 `outputs/GMVUPUP-文档验收.md`。

## 本机预览

在 `web/` 执行 `bun install --frozen-lockfile`，然后用两个终端运行：

```sh
node scripts/gmvupup-preview-api.mjs
```

```sh
VITE_REACT_APP_SERVER_URL=http://127.0.0.1:4181 bun run dev --host 127.0.0.1 --port 4180
```

打开 http://127.0.0.1:4180/ 或 http://127.0.0.1:4180/docs 。预览 API 仅监听回环地址，返回只读样例数据，拒绝写请求，不连接生产或上游。预览的价格目录为空；账号和推理操作须在真正的 New API 环境验证。部署时正常使用 New API 后端，不使用该预览 API。

官网内容位于 `web/src/features/gmvupup/`，独立样式位于 `web/src/styles/gmvupup.css`。保留原 `features/home` 的自定义内容加载路径，仅替换默认内容分支。

## 组件复用

交互使用仓库已有 Button、Tabs、Accordion、Sheet、CopyButton 和 LanguageSwitcher。公共 PublicHeader 的现有实现未实际使用其声明的 navContent/rightContent 插槽，导航由后台配置覆盖，且没有页面内锚点的独立布局；本次只为官网组合品牌导航，不改动其他页面。Footer 的系统名称与 Logo 优先于外部 props，无法在不改变后台配置的情况下固定独立品牌；官网使用独立语义化页脚，同时保留可见的原项目归属和动态法律页面开关。

## 视觉资产

`web/public/gmvupup/creative-sneaker.webp`（233,616 bytes）由内置 imagegen 生成，再仅作 WebP 编码优化。原图保存在工作区 `outputs/gmvupup-creative-sneaker-original.png`。这是网站概念视觉，不是 CCG/Seedance 的实测输出。

生成提示词：高端广告产品摄影；无品牌的淡紫色运动鞋，荧光绿细节和半透明鞋底，斜向悬浮于淡紫色沙丘上；后方浅荧光绿圆盘与飘动的淡紫色绸缎；柔和、有方向的棚拍光，材质清晰；不添加 Logo、文字、水印、边框或 UI。

## 检查

```sh
cd web
bun run typecheck
bunx --no-install oxlint -c .oxlintrc.json src/features/gmvupup src/features/gmvupup-docs src/routes/docs src/features/home/index.tsx scripts/gmvupup-preview-api.mjs
bunx --no-install vitest run src/features/gmvupup/__tests__/homepage.test.tsx src/features/gmvupup-docs/__tests__/documentation.test.tsx
bun run build
```

验证结果见工作区 `outputs/GMVUPUP-验收.md` 和 `outputs/GMVUPUP-文档验收.md`。本次是本地前端开发；服务器版本更新按用户要求走既有自建 CI 测试、构建和固定版本制品部署。本次未提交、推送或部署。

## 验证目录

工作区 Documents 目录的多个原有文件读取曾阻塞，类型检查与构建因此未结束。已在 `/Users/qinxian/.codex/worktrees/gmvupup-validation-20261108` 的同版本隔离副本完成检查和生产前端构建，再用 SHA-256 清单核对并同步官网文件；未将文件系统等待误报为代码通过。该验证目录保留依赖和构建结果，可直接按上面的命令启动本地预览。两份源码的对应关系见工作区 `outputs/gmvupup-validation-manifest.json`；文档任务新增同步清单见 `outputs/gmvupup-docs/sync-manifest.json`。

<details>
<summary>主视觉完整生成提示词（内置 imagegen）</summary>

Use case: ads-marketing. Asset type: hero concept photograph for GMVUPUP, a premium AI creative API website. Create a single striking high-end advertising still life, square 1024 by 1024 composition. A sculptural unbranded lavender and pale lilac running sneaker with vivid acid-lime accents, translucent lime wavy sole and beautifully detailed mesh, floating diagonally from lower left to upper right above undulating pale lavender sand dunes. Behind the shoe is a very large luminous pale lime circular studio disc, mostly cropped, with hazy lavender atmospheric background. A small ripple of lilac satin flows behind the heel, suggesting subtle movement. Make the sneaker the unmistakable main subject occupying 65 percent of the image, beautifully lit editorial product photography with strong directional shadows and tactile materials, sophisticated surreal commercial art direction. Colors limited to lavender, pale orchid, acid lime, and warm white. No logos, no words, no text, no watermark, no interface, no border. It must feel like an expensive sportswear campaign, not a generic 3D illustration.

</details>
