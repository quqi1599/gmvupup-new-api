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

客户模型 ID 为 `doubao-seedance-2.5` 和 `doubao-seedance-2.0`。用本站 API Key 调用 Doubao 原生接口。

### 提交

```bash
curl "https://gmvupup.top/doubao/api/v3/contents/generations/tasks" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY" \
  -H "Content-Type: application/json" \
  --max-time 30 \
  -d '{
    "model": "doubao-seedance-2.5",
    "content": [{"type": "text", "text": "A white cup on a table. The camera slowly pushes in."}],
    "duration": 5,
    "resolution": "480p",
    "ratio": "16:9",
    "generate_audio": false
  }'
```

必填 `model` 和 `content`。上面的文生视频示例也明确指定时长、分辨率和宽高比。参考素材按模型支持的原生媒体类型加入 `content`。

### 查询、下载和扣费

从响应 JSON **正文**复制 `id`（以 `task_` 开头）。`x-oneapi-request-id` 响应头用于排查请求。

```bash
curl "https://gmvupup.top/doubao/api/v3/contents/generations/tasks/$TASK_ID" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY"
```

- `queued` 或 `running` 时，每约 10 秒重复查询。提交超时时先检查是否创建了任务，避免重复消费。
- `succeeded` 后下载 `content.video_url` 对应的临时签名文件。访问该结果链接时省略本站 API Key。
- `failed`、`cancelled` 或 `expired` 时检查错误与账单记录。

[模型广场](https://gmvupup.top/pricing)显示的每秒价格用于估算。最终扣费按返回的计费 Token、分辨率、是否有参考视频及适用分组结算。任务完成后，在使用日志核对预扣、补扣或退款。

## 错误与重试

先读懂响应，再决定是否重试。

- 付费生成请求超时后不要盲目重发，第一次请求可能仍在执行。
- 对适合重试的失败使用有上限的指数退避；参数或权限错误应先修正原因。
