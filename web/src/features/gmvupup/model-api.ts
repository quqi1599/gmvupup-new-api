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
import type { PricingModel } from '@/features/pricing/types'

import { isCcgSeedanceModel } from './api-examples'
import { isNativeSeedanceModel } from './customer-models'

export const MODEL_API_ENDPOINTS: Record<
  string,
  { path: string; method: string }
> = {
  doubao: { path: '/doubao/api/v3/contents/generations/tasks', method: 'POST' },
  hailuo: { path: '/v2/video_generation', method: 'POST' },
  alibaba: {
    path: '/ali/api/v1/services/aigc/video-generation/video-synthesis',
    method: 'POST',
  },
  'image-generation': { path: '/v1/images/generations', method: 'POST' },
  'image-edits': { path: '/v1/images/edits', method: 'POST' },
}

export function getModelEndpointTypes(model: PricingModel): string[] {
  if (isNativeSeedanceModel(model)) return ['doubao']
  if (isCcgSeedanceModel(model)) return ['openai-video']
  if (model.model_name === 'MiniMax-H3') return ['hailuo']
  if (model.model_name === 'wan3.0-video') return ['alibaba']
  if (
    /^MiniMax-H3-Image(?:-Highspeed)?$/.test(model.model_name) ||
    ['gpt-image-2.6', 'nanobanana-2.1'].includes(model.model_name)
  ) {
    return ['image-generation', 'image-edits']
  }
  return model.supported_endpoint_types ?? []
}

export function buildModelApiExample(options: {
  model: PricingModel
  endpointType: string
  path: string
  language: 'curl' | 'python' | 'typescript' | 'javascript'
  baseUrl: string
  apiKeyEnv: string
}): string | undefined {
  const { model, endpointType, language, apiKeyEnv } = options
  const prompt =
    'A blue ceramic cup on a pale yellow background, soft studio light.'
  let body: Record<string, unknown>
  if (endpointType === 'gemini') {
    body = {
      contents: [
        {
          role: 'user',
          parts: [{ text: 'Describe a blue ceramic cup in one sentence.' }],
        },
      ],
    }
  } else if (endpointType === 'doubao' || endpointType === 'hailuo') {
    body = {
      model: model.model_name,
      content: [{ type: 'text', text: prompt }],
      duration: 4,
      resolution: endpointType === 'hailuo' ? '768P' : '720p',
      ratio: '16:9',
    }
  } else if (endpointType === 'alibaba') {
    body = {
      model: model.model_name,
      input: { prompt },
      parameters: { resolution: '480P', duration: 2, ratio: '16:9' },
    }
  } else if (
    endpointType === 'image-generation' ||
    endpointType === 'image-edits'
  ) {
    const minimax = model.model_name.startsWith('MiniMax-H3-Image')
    body = {
      model: model.model_name,
      prompt,
      n: 1,
      size: minimax ? '1024x1024' : '1k',
      response_format: 'url',
    }
    if (endpointType === 'image-edits') {
      body.images = [{ image_url: 'https://example.com/reference.png' }]
    }
    if (minimax && endpointType === 'image-generation') {
      body.disable_prompt_enhanced = true
    }
  } else {
    return undefined
  }
  const url =
    options.baseUrl.replace(/\/$/, '') +
    options.path.replace('{model}', model.model_name)
  const json = JSON.stringify(body, null, 2)
  if (language === 'curl') {
    return [
      `curl "${url}" \\`,
      `  -H "Authorization: Bearer $${apiKeyEnv}" \\`,
      '  -H "Content-Type: application/json" \\',
      '  --max-time 600 \\',
      `  -d '${json}'`,
    ].join('\n')
  }
  if (language === 'python') {
    return [
      'import json',
      'import os',
      'import requests',
      '',
      `response = requests.post("${url}",`,
      `    headers={"Authorization": "Bearer " + os.environ["${apiKeyEnv}"]},`,
      `    json=json.loads('''${json}'''),`,
      '    timeout=600,',
      ')',
      'response.raise_for_status()',
      'print(response.json())',
    ].join('\n')
  }
  return [
    `const response = await fetch("${url}", {`,
    '  method: "POST",',
    '  headers: {',
    `    Authorization: "Bearer " + process.env.${apiKeyEnv},`,
    '    "Content-Type": "application/json",',
    '  },',
    `  body: JSON.stringify(${json}),`,
    '  signal: AbortSignal.timeout(600000),',
    '});',
    'if (!response.ok) throw new Error("HTTP " + response.status);',
    'console.log(await response.json());',
  ].join('\n')
}
