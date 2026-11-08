# GMVUPUP 官网定制

GMVUPUP / gmvupup.top 的独立 New API 前端工作目录。官方基线为 `v1.0.0-rc.48`（`706144769e5bed5a47b78721e3bb11da3cf4c0b9`），官网初始分支 `codex/gmvupup-homepage`；文档开发与最终验证使用独立验证目录的 `codex/gmvupup-docs`。

GitHub 账号已有官方同源 fork `quqi1599/new-api`；`gh repo fork --fork-name gmvupup-new-api` 返回已有 fork。因此本工作目录以 `origin` 指向现有 fork、`upstream` 指向 `QuantumNous/new-api`，与其他 New API 本地目录分开。GMVUPUP 使用独立发布分支 `codex/gmvupup-release`，通过 `.github/workflows/gmvupup-production.yml` 在既有 `ci-vps` runner 检查、构建并发布 `ghcr.io/quqi1599/gmvupup-new-api:sha-<commit前12位>`。生产按 digest 部署；不更新贵数旧站的 main 分支或镜像。

## 2026-11-09 首页与控制台简化（当前本地版）

本轮基于已发布的 `54e8cd7a`，分支 `codex/gmvupup-simple-console`。发布时合并至 `codex/gmvupup-release`，由既有自建 CI 测试、构建固定版本镜像后部署。

- 按用户提供的「GmvUpUp-开发源码.zip」采用红黑白首页，保留模型品牌视觉，只有一个可点击入口「进入控制台」。去掉演示弹窗、模型点击、语言按钮、文档按钮与未经核实的调用量/可用率数据。
- 复用现有登录检查，登录后默认 `/dashboard/overview` 显示本站实时模型列表；未登录进入原登录页。
- 客户控制台参考所附 Grsai 截图：左侧固定一级导航，顶部余额与账号，深色说明区，模型筛选/搜索，三列价格卡片；手机改为单列卡片与侧栏抽屉。
- 卡片直接显示模型名、用途一句话与价格。图片按张，视频按每个已配置分辨率逐项列出秒价；文字模型显示当前时段的输入/输出/缓存费用。复用本站价格、汇率、分组和表达式显示函数，不写死参考站价格，不修改渠道或账本。
- 公告、个人资料、数据看板、模型列表、使用日志、充值、订单、任务日志、密钥、文档等均为直接入口。订单复用原账单组件的页内展示；保留管理员原有工作区和权限。
- 模型 ID 支持复制；正常卡片无需打开详情页才能看价格。特殊条件价格使用既有详情抽屉，避免猜测价格。
- 公开目录当前读取两个图片模型和 DeepSeek；视频卡片使用配置驱动，开通后自动出现。测试视频价格只用于本地 fixture，不对外宣称已经可调用。
- 界面提供中英文，其他语言回退英文。保留 New API 署名、许可证与项目来源。

源码：`web/src/features/gmvupup/`、`web/src/features/gmvupup-console/`；样式：`gmvupup-landing.css`、`gmvupup-console.css`。验证与截图在工作区 `outputs/gmvupup-console-20261109/`。

## API 文档

入口 `/docs`，适配 GMVUPUP 的 Base URL `https://gmvupup.top/v1`。参考 AGToken 文档的信息结构，重做为本站暖白、深绿、荧光绿的三栏排版；手机端改为抽屉目录。

- 8 篇内容：快速开始、身份验证、模型目录、图像生成、图像编辑、对话生成、视频任务、错误与重试。
- 文档搜索、Enter 打开结果、页内导航、上下篇、Base URL 复制、cURL/Python/JavaScript 切换、示例模型切换、代码复制和下载。
- 公开模型目录复用 `/api/pricing`；令牌自己的可用范围通过 `GET /v1/models` 查询。代码示例使用本站 ID `gpt-image-2.6`、`nanobanana-2.1`、`deepseek-v4.1-flash`，不直接使用上游别名。
- 图像说明采用本站同步接口。视频页明确要求先开通对应模型；示例为已有 Doubao 适配器的 `doubao-seedance-2.5`，不是 CCG 插件协议，不宣称当前已可调用。
- `/gmvupup/docs/openapi.json` 提供 OpenAPI 3.1 文件，覆盖模型列表、图像生成/编辑和对话；`/gmvupup/docs/guide.md` 提供中文接入指南。
- 控制台文档入口指向 `/docs`；文档页导航使用实际控制台和文档路由。

源码：`web/src/features/gmvupup-docs/`、`web/src/routes/docs/`、`web/src/styles/gmvupup-docs.css`。验证与截图：工作区 `outputs/gmvupup-docs/`，报告 `outputs/GMVUPUP-文档验收.md`。

## 本机预览

依赖与构建在验证目录 `/Users/qinxian/.codex/worktrees/gmvupup-validation-20261108/web` 中执行。当前任务在本机 `http://127.0.0.1:4200/` 和 `/dashboard/overview` 提供预览。

独立终端启动工作区 `outputs/gmvupup-console-20261109/preview-api.mjs`，然后在 `web/` 运行：

```sh
VITE_REACT_APP_SERVER_URL=http://127.0.0.1:4201 bun run dev --host 127.0.0.1 --port 4200
```

预览服务仅监听回环地址；价格来自本轮公开目录快照，账户与余额为本机测试数据；仅模拟刷新测试会话，其他写请求全部拒绝，不连接生产或上游。生产由现有 New API 后端提供真实认证与数据。

## 组件复用

客户控制台组合仓库的 Sidebar、NavGroup、ProfileDropdown、LanguageSwitcher，继续使用 useSidebarConfig 的管理员/用户菜单开关。原 AppSidebar 采用进入子工作区后切换目录的方式，不能提供本轮要求的客户固定一级导航；管理员页继续沿用它。原 AppHeader 将系统品牌和版本固定在全宽顶部，新增客户页组合把品牌置于左栏，并保留共享账号控件。

模型卡片组合 Card、CopyButton、原有价格格式化/分组/时间表达式/任务价格展示函数。原 ModelCard 固定展示接口、分组、性能与详情入口，且任务价格为范围摘要；本轮需要常驻的逐分辨率价格格和一句说明，因此新增配置驱动的展示组件，不另写计费计算。BillingHistoryDialog 新增兼容的 inline 展示，订单页复用原查询、复制和确认行为。

## 视觉资产

`web/public/gmvupup/creative-sneaker.webp`（233,616 bytes）由内置 imagegen 生成，再仅作 WebP 编码优化。原图保存在工作区 `outputs/gmvupup-creative-sneaker-original.png`。这是前一版网站概念视觉，新首页不再使用；不是 CCG/Seedance 的实测输出。

生成提示词：高端广告产品摄影；无品牌的淡紫色运动鞋，荧光绿细节和半透明鞋底，斜向悬浮于淡紫色沙丘上；后方浅荧光绿圆盘与飘动的淡紫色绸缎；柔和、有方向的棚拍光，材质清晰；不添加 Logo、文字、水印、边框或 UI。

## 检查

```sh
cd web
bun run typecheck
bunx --no-install oxlint -c .oxlintrc.json src/features/gmvupup src/features/gmvupup-docs src/routes/docs src/features/home/index.tsx scripts/gmvupup-preview-api.mjs
bunx --no-install vitest run src/features/gmvupup/__tests__/homepage.test.tsx src/features/gmvupup-console/__tests__ src/features/gmvupup-docs/__tests__/documentation.test.tsx
bun run build
```

本轮验证结果见工作区 `outputs/gmvupup-console-20261109/REPORT.md`。本次是本地前端开发；服务器版本更新按用户要求走既有自建 CI 测试、构建和固定版本制品部署。本次未提交、推送或部署。

## 验证目录

工作区 Documents 目录的多个原有文件读取曾阻塞，类型检查与构建因此未结束。已在 `/Users/qinxian/.codex/worktrees/gmvupup-validation-20261108` 的同版本隔离副本完成检查和生产前端构建，再用 SHA-256 清单核对并同步官网文件；未将文件系统等待误报为代码通过。该验证目录保留依赖和构建结果，可直接按上面的命令启动本地预览。两份源码的对应关系见工作区 `outputs/gmvupup-validation-manifest.json`；文档任务新增同步清单见 `outputs/gmvupup-docs/sync-manifest.json`。

<details>
<summary>主视觉完整生成提示词（内置 imagegen）</summary>

Use case: ads-marketing. Asset type: hero concept photograph for GMVUPUP, a premium AI creative API website. Create a single striking high-end advertising still life, square 1024 by 1024 composition. A sculptural unbranded lavender and pale lilac running sneaker with vivid acid-lime accents, translucent lime wavy sole and beautifully detailed mesh, floating diagonally from lower left to upper right above undulating pale lavender sand dunes. Behind the shoe is a very large luminous pale lime circular studio disc, mostly cropped, with hazy lavender atmospheric background. A small ripple of lilac satin flows behind the heel, suggesting subtle movement. Make the sneaker the unmistakable main subject occupying 65 percent of the image, beautifully lit editorial product photography with strong directional shadows and tactile materials, sophisticated surreal commercial art direction. Colors limited to lavender, pale orchid, acid lime, and warm white. No logos, no words, no text, no watermark, no interface, no border. It must feel like an expensive sportswear campaign, not a generic 3D illustration.

</details>
