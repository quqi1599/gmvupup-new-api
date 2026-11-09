# GMVUPUP API 接入指南

文档：https://gmvupup.top/docs

SDK Base URL：https://gmvupup.top/v1

鉴权：Authorization: Bearer YOUR_GMVUPUP_API_KEY

先在控制台创建密钥，再用 GET /v1/models 核对可用模型。不要在浏览器前端保存真实密钥。

## 模型目录

找到准确的 model ID，以及对应的调用接口。

GET /v1/models

```bash
curl "https://gmvupup.top/v1/models" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY"
```

- 完整复制 model ID，包括标点和大小写。请求使用本站模型 ID，不使用上游别名。
- 可用范围和价格可能调整，批量调用前请核对实时目录与密钥权限。

## 图像生成

通过 OpenAI 兼容接口，把文字描述变成图像。

POST /v1/images/generations

```bash
curl "https://gmvupup.top/v1/images/generations" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY" \
  -H "Content-Type: application/json" \
  --max-time 240 \
  -d '{
  "model": "gpt-image-2.6",
  "prompt": "A lavender sneaker on pale sand, soft studio lighting.",
  "n": 1,
  "size": "1024x1024"
}'
```

- 按实际响应读取 data[].url 或 data[].b64_json。返回的链接可能过期，请及时下载并保存产物。
- 本页说明同步图片请求。上游的 async 参数或轮询地址不能直接当作本站接口使用。
- 高级质量、输出格式和透明背景参数取决于模型，正式使用前请确认所选模型支持。

## 图像编辑

上传参考图片，说明你希望修改的内容。

POST /v1/images/edits

```bash
curl "https://gmvupup.top/v1/images/edits" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY" \
  --max-time 240 \
  -F "model=gpt-image-2.6" \
  -F "image=@reference.png" \
  -F "prompt=Change the background to pale lavender. Keep the product unchanged." \
  -F "n=1"
```

- 将 reference.png 换成你自己的文件。该上传请求不要手动设置 Content-Type: application/json。
- 图像编辑能力按模型目录展示；发送前请确认所选模型的文件和参数限制。

## 对话生成

发送消息，获得模型的回答。

POST /v1/chat/completions

```bash
curl "https://gmvupup.top/v1/chat/completions" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY" \
  -H "Content-Type: application/json" \
  --max-time 240 \
  -d '{
  "model": "deepseek-v4.1-flash",
  "messages": [
    {
      "role": "user",
      "content": "Write three short headlines for a new running shoe."
    }
  ],
  "stream": false
}'
```

- 非流式回答可从 choices[0].message.content 读取。处理成功结果前先检查 HTTP 状态和错误响应。
- 多模态输入、工具调用和流式输出属于不同能力，模型出现在目录中不等于支持所有可选功能。

## Seedance 视频任务

模型：`doubao-seedance-2.5`、`doubao-seedance-2.0`。使用本站 Key，通过 `POST /v1/videos` 提交 JSON。

### 人民币固定秒价

2026-11-09 配置价格，default/vip/svip 当前分组倍率均为 1。实时价格以 https://gmvupup.top/pricing 为准。

| 模型 | 480p 元/秒 | 720p 元/秒 | 1080p 元/秒 |
| --- | ---: | ---: | ---: |
| doubao-seedance-2.5 | 0.536 | 1.208 | 2.992 |
| doubao-seedance-2.0 | 0.368 | 0.792 | 1.984 |

费用 = 输出时长 × 对应分辨率秒价。比如 5 秒 720p，2.5 为 6.04 元，2.0 为 3.96 元。参考素材输入也使用对应分辨率的固定输出秒价。任务失败后退回本站预扣额度。

定价参考[火山方舟中国区官方价格](https://ark.volcengine.com/region:cn-beijing/docs/model-pricing)中 16:9、无参考视频的每秒示例值 × 0.8；本站按固定秒价计费，不按实际视频 Token 重算。

### 提交

```bash
curl "https://gmvupup.top/v1/videos" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY" \
  -H "Content-Type: application/json" \
  --max-time 240 \
  -d '{
    "model": "doubao-seedance-2.5",
    "userPrompt": "A sneaker floating above lilac dunes, slow camera movement.",
    "duration": 5,
    "quality": "720p",
    "aspectRatio": "16:9"
  }'
```

两模型请求格式一致。必填 `model`、`userPrompt`、`duration`、`quality`、`aspectRatio`：

- `userPrompt`：1–32000 字符，不能全为空白。
- `duration`：4–15 的整数秒。
- `quality`：480p / 720p / 1080p。
- `aspectRatio`：21:9 / 16:9 / 4:3 / 1:1 / 3:4 / 9:16。
- `firstImageUrl`、`lastImageUrl`、`refImageUrls`：首尾帧与参考图片合计最多 9 张。
- `refVideoUrls`、`refAudioUrls`：视频和音频参考各最多 3 个。
- 参考素材使用可访问的绝对 HTTP(S) URL，或有权使用的 `asset://asset_ID`；不支持本地路径、Base64 和 multipart 上传。
- 可选 `generateAudio`、`returnLastFrame` 为布尔值；`seed` 为 -1–4294967295 整数；`bitrateMode` 为 standard / high。
- 不接受 `prompt`、`seconds`、`size` 等 OpenAI 视频字段。

响应示意（不是实际生成结果）：

```json
{"id":"YOUR_TASK_ID","object":"video","status":"queued"}
```

### 查询和下载

保存返回的 `id` 为 `TASK_ID`，约每 10 秒查询一次。`queued`、`in_progress` 表示处理中，`completed` 后可下载，`failed` 时读取错误原因。

```bash
curl "https://gmvupup.top/v1/videos/$TASK_ID" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY"

# 仅在 completed 后下载
curl "https://gmvupup.top/v1/videos/$TASK_ID/content" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY" \
  -o result.mp4

# 仅在请求 returnLastFrame 且返回 lastFrameUrl 后下载
curl "https://gmvupup.top/v1/tasks/$TASK_ID/artifacts/last_frame/content" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY" \
  -o last-frame.jpg
```

查询、下载均使用本站 Key 和本站任务 ID。超时后先查询原任务，不自动重复提交。完整参数和三种语言示例见 https://gmvupup.top/docs/videos 。

## 错误与重试

先读懂响应，再决定是否重试。

- 付费生成请求超时后不要盲目重发，第一次请求可能仍在执行。
- 对适合重试的失败使用有上限的指数退避；参数或权限错误应先修正原因。
