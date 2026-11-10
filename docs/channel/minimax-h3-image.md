# MiniMax H3 图片生成

使用 MiniMax 渠道（类型 35），Base URL 为 `https://api.minimax.cn`，添加模型：

- `MiniMax-H3-Image`
- `MiniMax-H3-Image-Highspeed`

H3 使用 `/v1/images/generations` 和 `/v1/images/edits`。旧模型 `image-01`、`image-01-live` 仍走原来的 `/v1/image_generation`。

## 请求

文生图使用 JSON；编辑支持 JSON `images[].image_url`（公网 URL、data URI、`mm_file://`），以及 OpenAI SDK 的 multipart `image` / `image[]` 文件上传。

```json
{
  "model": "MiniMax-H3-Image",
  "prompt": "A ceramic cup on a table",
  "area": "1K",
  "aspect_ratio": "1:1",
  "n": 1,
  "response_format": "url",
  "stream": false
}
```

支持 `size`、`area`、`aspect_ratio`、`output_format`、`output_compression`、`disable_prompt_enhanced`、`aigc_watermark`。显式的 `false` 和 `0` 会保留。`n` 为 1–10；编辑参考图为 1–16 张。尺寸和输出面积等限制以官方接口校验为准。

客户端、网关和前置代理的等待预算应不小于 600 秒；客户端 SDK 设置 `max_retries=0`。H3 请求发出后的 HTTP 错误、超时或断流不跨渠道重试，避免重复生成与计费。

## 响应和结算

- 普通 JSON 保留逐张结果、失败项、尺寸、增强提示词、`request_id` 和 `usage`；支持 URL 和 Base64。
- 流式响应原样保留 `image_generation.*` / `image_edit.*` 事件：`partial_succeeded` 是一张完成的图片，`partial_failed` 是单张失败，`completed` 是最终用量汇总。上游异常断流不会补发成功终止事件。
- token 计费分别使用文本输入、图片输入和图片输出用量；输入/输出别名不会重复相加，也不会再乘请求张数。
- 若管理员选择按张收费，按实际完成图片数结算；HTTP 200 中的失败项不收费。异常断流仅按已收到的用量或完整图片处理，不估算缺失的 token。
- 上游返回的图片 URL 可复用现有图片结果缓存。客户端不能把收到流式事件等同于请求已全部成功，必须检查最终事件。

本次不增加固定按张默认价；运营价格由管理员配置。官方按 token 收费，单张示例费用不是固定价格。

## 核验来源

2026-11-11 核对：[生成指南](https://platform.minimax.cn/docs/guides/image-generation)、[流式事件](https://platform.minimax.cn/docs/api-reference/image-generation-h3-streaming)、[美元原价](https://platform.minimax.io/docs/pricing/overview)。

本地自动检查覆盖模型映射、JSON 字段、multipart 顺序与认证、逐张失败、重复事件、断流、HTTP 错误禁止重试和 SQLite 额度账本结算。真实调用另有标准版文生图及高速版 multipart 流式编辑验证；线上服务升级需要独立走正式发布流程。
