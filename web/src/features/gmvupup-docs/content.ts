/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { API_EXAMPLES, SEEDANCE_MODELS } from '@/features/gmvupup/api-examples'

export type CodeExamples = Record<'curl' | 'python' | 'javascript', string>
export type DocField = {
  name: string
  type: string
  required: boolean
  description: string
}
export type DocPage = {
  title: string
  summary: string
  group: 'start' | 'reference' | 'help'
  keywords: string
  body?: string[]
  method?: string
  endpoint?: string
  models?: string[]
  conditional?: string
  fields?: DocField[]
  examples?: CodeExamples
  response?: string
  followUp?: string
  notes: string[]
}
export const DOC_IDS = [
  'quick-start',
  'agent-skills',
  'authentication',
  'models',
  'images',
  'image-edits',
  'chat',
  'videos',
  'errors',
] as const
export type DocTopic = (typeof DOC_IDS)[number]
export const API_ORIGIN = 'https://gmvupup.top'
export const API_BASE_URL = `${API_ORIGIN}/v1`
export const DOC_PAGES: Record<DocTopic, DocPage> = {
  'quick-start': {
    title: 'Quick start',
    summary: 'A clear path from your first API key to your first response.',
    group: 'start',
    keywords: 'start 开始 入门 base url 地址 接口',
    examples: {
      curl: 'curl "https://gmvupup.top/v1/models" \\\n  -H "Authorization: Bearer $GMVUPUP_API_KEY"',
      python:
        'import os\nimport requests\n\nresponse = requests.get(\n    "https://gmvupup.top/v1/models",\n    headers={"Authorization": "Bearer " + os.environ["GMVUPUP_API_KEY"]},\n    timeout=30,\n)\nresponse.raise_for_status()\nprint(response.json())',
      javascript:
        'const response = await fetch("https://gmvupup.top/v1/models", {\n  headers: {\n    Authorization: "Bearer " + process.env.GMVUPUP_API_KEY,\n  },\n  signal: AbortSignal.timeout(30000),\n});\nif (!response.ok) throw new Error("HTTP " + response.status);\nconsole.log(await response.json());',
    },
    notes: [
      'Keep API keys on your server or in environment variables. Never include a live key in frontend code or public repositories.',
    ],
  },
  'agent-skills': {
    title: 'Agent Skills',
    summary:
      'Copy a skill into your agent to generate images and videos with GMVUPUP.',
    group: 'start',
    keywords:
      'agent skill prompt 复制 提示词 生图 视频 Seedance Wan GPT Image Nano Banana',
    body: [
      'Use an agent with HTTP or terminal tools. Configure GMVUPUP_API_KEY in its environment or secret settings first.',
    ],
    notes: [
      'Only the models returned by GET /v1/models are available to your key. Check live pricing before generation.',
    ],
  },
  authentication: {
    title: 'Authentication',
    summary: 'Use one GMVUPUP API key to authenticate your requests.',
    group: 'start',
    keywords: 'key token 密钥 认证 bearer authorization',
    body: [
      'Create a key in the console and include it in the Authorization header. The key must be allowed to use the selected model and group.',
    ],
    examples: {
      curl: 'export GMVUPUP_API_KEY="YOUR_GMVUPUP_API_KEY"\n\ncurl "https://gmvupup.top/v1/models" \\\n  -H "Authorization: Bearer $GMVUPUP_API_KEY"',
      python:
        'import os\n\nheaders = {\n    "Authorization": "Bearer " + os.environ["GMVUPUP_API_KEY"],\n    "Content-Type": "application/json",\n}',
      javascript:
        'const headers = {\n  Authorization: "Bearer " + process.env.GMVUPUP_API_KEY,\n  "Content-Type": "application/json",\n};',
    },
    fields: [
      {
        name: 'Authorization',
        type: 'header',
        required: true,
        description: 'Bearer followed by one space and your GMVUPUP API key.',
      },
      {
        name: 'Content-Type',
        type: 'header',
        required: false,
        description:
          'Use application/json for JSON requests. Let your client set the boundary for multipart uploads.',
      },
    ],
    notes: [
      'Your console password and your API key serve different purposes. Requests use an API key, not your sign-in password.',
      'If a key is exposed, revoke it in the console and create a replacement. Update the server configuration before resuming requests.',
    ],
  },
  models: {
    title: 'Model directory',
    summary: 'Find the exact model ID and the interfaces available for it.',
    group: 'start',
    keywords: 'model models 模型 目录 GPT nano deepseek',
    method: 'GET',
    endpoint: '/v1/models',
    body: [
      'The directory below reads the public model catalog. Your API key may have a narrower model scope; GET /v1/models returns the list visible to that key.',
    ],
    examples: {
      curl: 'curl "https://gmvupup.top/v1/models" \\\n  -H "Authorization: Bearer $GMVUPUP_API_KEY"',
      python:
        'import os\nimport requests\n\nresponse = requests.get(\n    "https://gmvupup.top/v1/models",\n    headers={"Authorization": "Bearer " + os.environ["GMVUPUP_API_KEY"]},\n    timeout=30,\n)\nresponse.raise_for_status()\nprint(response.json())',
      javascript:
        'const response = await fetch("https://gmvupup.top/v1/models", {\n  headers: {\n    Authorization: "Bearer " + process.env.GMVUPUP_API_KEY,\n  },\n  signal: AbortSignal.timeout(30000),\n});\nif (!response.ok) throw new Error("HTTP " + response.status);\nconsole.log(await response.json());',
    },
    response:
      '{\n  "object": "list",\n  "data": [\n    {\n      "id": "gpt-image-2.6",\n      "object": "model"\n    }\n  ]\n}',
    notes: [
      'Copy the model ID exactly, including punctuation and capitalization. Use the GMVUPUP model ID rather than a provider alias.',
      'Availability and pricing can change. Check the live catalog and your key permissions before starting a batch.',
    ],
  },
  images: {
    title: 'Image generation',
    summary:
      'Turn a text prompt into an image with an OpenAI-compatible request.',
    group: 'reference',
    keywords: 'images 图片 文生图 gpt-image-2.6 nanobanana-2.1',
    method: 'POST',
    endpoint: '/v1/images/generations',
    body: [
      'Send a JSON request with a model ID and a prompt. The example starts with a single image and a 1024x1024 request size.',
    ],
    models: ['gpt-image-2.6', 'nanobanana-2.1'],
    examples: {
      curl: 'curl "https://gmvupup.top/v1/images/generations" \\\n  -H "Authorization: Bearer $GMVUPUP_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  --max-time 240 \\\n  -d \'{\n  "model": "gpt-image-2.6",\n  "prompt": "A lavender sneaker on pale sand, soft studio lighting.",\n  "n": 1,\n  "size": "1024x1024"\n}\'',
      python:
        'import os\nimport requests\n\nresponse = requests.post(\n    "https://gmvupup.top/v1/images/generations",\n    headers={"Authorization": "Bearer " + os.environ["GMVUPUP_API_KEY"]},\n    json={\n      "model": "gpt-image-2.6",\n      "prompt": "A lavender sneaker on pale sand, soft studio lighting.",\n      "n": 1,\n      "size": "1024x1024"\n    },\n    timeout=240,\n)\nresponse.raise_for_status()\nprint(response.json())',
      javascript:
        'const response = await fetch("https://gmvupup.top/v1/images/generations", {\n  method: "POST",\n  headers: {\n    Authorization: "Bearer " + process.env.GMVUPUP_API_KEY,\n    "Content-Type": "application/json",\n  },\n  body: JSON.stringify({\n    "model": "gpt-image-2.6",\n    "prompt": "A lavender sneaker on pale sand, soft studio lighting.",\n    "n": 1,\n    "size": "1024x1024"\n  }),\n  signal: AbortSignal.timeout(240000),\n});\nif (!response.ok) throw new Error("HTTP " + response.status);\nconsole.log(await response.json());',
    },
    fields: [
      {
        name: 'model',
        type: 'string',
        required: true,
        description: 'Exact model ID from the GMVUPUP catalog.',
      },
      {
        name: 'prompt',
        type: 'string',
        required: true,
        description:
          'Describe the subject, composition, materials and lighting.',
      },
      {
        name: 'n',
        type: 'integer',
        required: false,
        description:
          'The number of images; start with 1. Limits depend on the model.',
      },
      {
        name: 'size',
        type: 'string',
        required: false,
        description:
          'Requested size, for example 1024x1024. Read the actual dimensions of the returned image.',
      },
      {
        name: 'response_format',
        type: 'string',
        required: false,
        description:
          'Use url or b64_json when supported by the selected model.',
      },
    ],
    response:
      '{\n  "created": 1794096000,\n  "data": [\n    {\n      "b64_json": "BASE64_IMAGE_DATA"\n    }\n  ]\n}',
    notes: [
      'Read data[].url or data[].b64_json according to the actual response. A returned URL may expire; download and store the asset promptly.',
      'This page documents synchronous image requests. Do not assume an upstream async flag or polling URL is available on GMVUPUP.',
      'Advanced quality, format and transparency options depend on the model. Confirm support before using them in production.',
    ],
  },
  'image-edits': {
    title: 'Image editing',
    summary: 'Upload a reference image and describe the change you want.',
    group: 'reference',
    keywords:
      'edit edits 图生图 图片编辑 上传 multipart gpt-image-2.6 nanobanana-2.1',
    method: 'POST',
    endpoint: '/v1/images/edits',
    body: [
      'Use multipart/form-data for a local image upload. Preserve the parts you want to keep in your prompt, then describe the requested edit.',
    ],
    models: ['gpt-image-2.6', 'nanobanana-2.1'],
    examples: {
      curl: 'curl "https://gmvupup.top/v1/images/edits" \\\n  -H "Authorization: Bearer $GMVUPUP_API_KEY" \\\n  --max-time 240 \\\n  -F "model=gpt-image-2.6" \\\n  -F "image=@reference.png" \\\n  -F "prompt=Change the background to pale lavender. Keep the product unchanged." \\\n  -F "n=1"',
      python:
        'import os\nimport requests\n\nwith open("reference.png", "rb") as image:\n    response = requests.post(\n        "https://gmvupup.top/v1/images/edits",\n        headers={"Authorization": "Bearer " + os.environ["GMVUPUP_API_KEY"]},\n        files={"image": ("reference.png", image, "image/png")},\n        data={\n            "model": "gpt-image-2.6",\n            "prompt": "Change the background to pale lavender.",\n            "n": "1",\n        },\n        timeout=240,\n    )\nresponse.raise_for_status()\nprint(response.json())',
      javascript:
        'import { readFile } from "node:fs/promises";\n\nconst form = new FormData();\nform.set("model", "gpt-image-2.6");\nform.set("image", new Blob([await readFile("reference.png")], {\n  type: "image/png",\n}), "reference.png");\nform.set("prompt", "Change the background to pale lavender.");\nform.set("n", "1");\nconst response = await fetch("https://gmvupup.top/v1/images/edits", {\n  method: "POST",\n  headers: { Authorization: "Bearer " + process.env.GMVUPUP_API_KEY },\n  body: form,\n  signal: AbortSignal.timeout(240000),\n});\nif (!response.ok) throw new Error("HTTP " + response.status);\nconsole.log(await response.json());',
    },
    fields: [
      {
        name: 'model',
        type: 'string',
        required: true,
        description: 'A model with image editing support in the catalog.',
      },
      {
        name: 'image',
        type: 'file',
        required: true,
        description: 'A local PNG, JPEG or WebP reference image.',
      },
      {
        name: 'prompt',
        type: 'string',
        required: true,
        description:
          'Describe the edit and the details that should stay unchanged.',
      },
    ],
    notes: [
      'Replace reference.png with your own file. Do not manually set Content-Type: application/json for this upload.',
      'Image editing support is declared in the model catalog. Check file and parameter limits for the selected model before sending.',
    ],
  },
  chat: {
    title: 'Chat completions',
    summary: 'Send messages and receive an assistant response.',
    group: 'reference',
    keywords: 'chat messages 文本 对话 deepseek-v4.1-flash',
    method: 'POST',
    endpoint: '/v1/chat/completions',
    body: [
      'The example uses deepseek-v4.1-flash with a non-streaming response. Store your key server-side and send messages in order.',
    ],
    examples: {
      curl: 'curl "https://gmvupup.top/v1/chat/completions" \\\n  -H "Authorization: Bearer $GMVUPUP_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  --max-time 240 \\\n  -d \'{\n  "model": "deepseek-v4.1-flash",\n  "messages": [\n    {\n      "role": "user",\n      "content": "Write three short headlines for a new running shoe."\n    }\n  ],\n  "stream": false\n}\'',
      python:
        'import os\nimport requests\n\nresponse = requests.post(\n    "https://gmvupup.top/v1/chat/completions",\n    headers={"Authorization": "Bearer " + os.environ["GMVUPUP_API_KEY"]},\n    json={\n      "model": "deepseek-v4.1-flash",\n      "messages": [\n        {\n          "role": "user",\n          "content": "Write three short headlines for a new running shoe."\n        }\n      ],\n      "stream": False\n    },\n    timeout=240,\n)\nresponse.raise_for_status()\nprint(response.json())',
      javascript:
        'const response = await fetch("https://gmvupup.top/v1/chat/completions", {\n  method: "POST",\n  headers: {\n    Authorization: "Bearer " + process.env.GMVUPUP_API_KEY,\n    "Content-Type": "application/json",\n  },\n  body: JSON.stringify({\n    "model": "deepseek-v4.1-flash",\n    "messages": [\n      {\n        "role": "user",\n        "content": "Write three short headlines for a new running shoe."\n      }\n    ],\n    "stream": false\n  }),\n  signal: AbortSignal.timeout(240000),\n});\nif (!response.ok) throw new Error("HTTP " + response.status);\nconsole.log(await response.json());',
    },
    fields: [
      {
        name: 'model',
        type: 'string',
        required: true,
        description: 'A chat model available to your API key.',
      },
      {
        name: 'messages',
        type: 'array',
        required: true,
        description:
          'An ordered array of messages containing role and content.',
      },
      {
        name: 'stream',
        type: 'boolean',
        required: false,
        description:
          'The example uses false. Stream support depends on the model and interface.',
      },
    ],
    response:
      '{\n  "choices": [\n    {\n      "index": 0,\n      "message": {\n        "role": "assistant",\n        "content": "Example assistant response."\n      },\n      "finish_reason": "stop"\n    }\n  ],\n  "usage": {\n    "prompt_tokens": 18,\n    "completion_tokens": 8,\n    "total_tokens": 26\n  }\n}',
    notes: [
      'For a non-streaming response, read choices[0].message.content. Check the error body before treating the request as successful.',
      'Multimodal input, tools and streaming are separate capabilities. A model appearing in the catalog does not imply support for every optional feature.',
    ],
  },
  videos: {
    title: 'Video tasks',
    summary:
      'Generate videos with Seedance 2.5 or 2.0, then poll and download the result.',
    group: 'reference',
    keywords: 'video seedance 2.5 2.0 视频 异步 任务 价格 轮询',
    method: 'POST',
    endpoint: '/v1/videos',
    models: SEEDANCE_MODELS,
    body: [
      'Use the JSON fields below for both Seedance models. Send requests with your GMVUPUP API key; the provider key is managed by the server.',
      'Seedance uses userPrompt, duration, quality and aspectRatio. The prompt, seconds and size fields and multipart uploads are not accepted.',
    ],
    examples: API_EXAMPLES,
    fields: [
      {
        name: 'model',
        type: 'string',
        required: true,
        description:
          'Use doubao-seedance-2.5 or doubao-seedance-2.0 with your GMVUPUP API key.',
      },
      {
        name: 'userPrompt',
        type: 'string',
        required: true,
        description:
          'Describe the scene, action and camera movement; 1–32000 characters, not blank.',
      },
      {
        name: 'duration',
        type: 'integer',
        required: true,
        description: 'Output duration: an integer from 4 to 15 seconds.',
      },
      {
        name: 'quality',
        type: 'string',
        required: true,
        description: 'Output resolution: 480p, 720p or 1080p.',
      },
      {
        name: 'aspectRatio',
        type: 'string',
        required: true,
        description: 'Aspect ratio: 21:9, 16:9, 4:3, 1:1, 3:4 or 9:16.',
      },
      {
        name: 'firstImageUrl / lastImageUrl',
        type: 'string',
        required: false,
        description:
          'First and last frame URLs. At most 9 images in total, including refImageUrls.',
      },
      {
        name: 'refImageUrls',
        type: 'string[]',
        required: false,
        description:
          'Reference image URLs; at most 9 images including first and last frames.',
      },
      {
        name: 'refVideoUrls / refAudioUrls',
        type: 'string[]',
        required: false,
        description: 'Reference video or audio URLs; at most 3 of each.',
      },
      {
        name: 'generateAudio',
        type: 'boolean',
        required: false,
        description:
          'Whether to generate audio. Omit to use the upstream default.',
      },
      {
        name: 'returnLastFrame',
        type: 'boolean',
        required: false,
        description:
          'Request a last-frame image. Download it only when lastFrameUrl is returned.',
      },
      {
        name: 'seed',
        type: 'integer',
        required: false,
        description: 'Random seed from -1 to 4294967295.',
      },
      {
        name: 'bitrateMode',
        type: 'string',
        required: false,
        description: 'Video bitrate preset: standard or high.',
      },
    ],
    response:
      '{\n  "id": "YOUR_TASK_ID",\n  "object": "video",\n  "status": "queued"\n}',
    followUp:
      'curl "https://gmvupup.top/v1/videos/$TASK_ID" \\\n  -H "Authorization: Bearer $GMVUPUP_API_KEY"\n\n# After status is completed:\ncurl "https://gmvupup.top/v1/videos/$TASK_ID/content" \\\n  -H "Authorization: Bearer $GMVUPUP_API_KEY" \\\n  -o result.mp4',
    notes: [
      'Save the returned id. Poll about every 10 seconds; queued and in_progress are not completed results. Download after completed, and inspect the error for failed tasks.',
      'A timeout does not cancel an existing task. Check the saved task ID before considering another submission.',
      'Use absolute HTTP(S) URLs for reference media. Authorized asset://asset_ID references are also supported; local file paths and Base64 are not accepted.',
      'When returnLastFrame is enabled and the completed response contains lastFrameUrl, download that relative path from https://gmvupup.top using the same API key.',
      'The rate is per output second at the selected resolution, including requests with reference media. A failed task returns its reserved GMVUPUP quota.',
    ],
  },
  errors: {
    title: 'Errors & retries',
    summary: 'Understand the response before deciding what to retry.',
    group: 'help',
    keywords: 'error errors 400 401 403 429 500 错误 重试 超时',
    body: [
      'Check the HTTP status together with error.message and error.code when present. Save the request time, model and any returned request ID for troubleshooting.',
    ],
    response:
      '{\n  "error": {\n    "message": "Example error message",\n    "type": "invalid_request_error",\n    "code": "example_code"\n  }\n}',
    fields: [
      {
        name: '400',
        type: 'HTTP',
        required: false,
        description:
          'Check JSON syntax, required fields and model parameters before retrying.',
      },
      {
        name: '401 / 403',
        type: 'HTTP',
        required: false,
        description:
          'Check the API key, model access, group and available quota.',
      },
      {
        name: '429',
        type: 'HTTP',
        required: false,
        description: 'Reduce concurrency and follow Retry-After when provided.',
      },
      {
        name: '500 / 502 / 503',
        type: 'HTTP',
        required: false,
        description:
          'Inspect the error message and request record; use bounded retries when appropriate.',
      },
    ],
    notes: [
      'Do not retry a paid generation blindly after a timeout. The first request may still be running.',
      'Use exponential backoff with an upper limit for retryable failures. Stop retrying until you fix invalid parameters or permissions.',
    ],
  },
}

export function isDocTopic(value: string): value is DocTopic {
  return Object.hasOwn(DOC_PAGES, value)
}
