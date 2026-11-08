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

## 视频任务

创建任务、查询进度，并获取完成后的视频。

需要开通对应视频模型。

POST /v1/videos

```bash
curl "https://gmvupup.top/v1/videos" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY" \
  -H "Content-Type: application/json" \
  --max-time 240 \
  -d '{
  "model": "doubao-seedance-2.5",
  "prompt": "A sneaker floating above lilac dunes, slow camera movement.",
  "seconds": 5,
  "size": "1280x720"
}'
```

```bash
curl "https://gmvupup.top/v1/videos/$TASK_ID" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY"

# After status is completed:
curl "https://gmvupup.top/v1/videos/$TASK_ID/content" \
  -H "Authorization: Bearer $GMVUPUP_API_KEY" \
  -o result.mp4
```

- 保存返回的 id，建议约每 10 秒查询一次。queued 和 in_progress 表示尚未完成；completed 后再下载，failed 时读取失败原因。
- 超时不等于已有任务取消。考虑重新提交前，请先查询已保存的任务 ID。

## 错误与重试

先读懂响应，再决定是否重试。

- 付费生成请求超时后不要盲目重发，第一次请求可能仍在执行。
- 对适合重试的失败使用有上限的指数退避；参数或权限错误应先修正原因。
