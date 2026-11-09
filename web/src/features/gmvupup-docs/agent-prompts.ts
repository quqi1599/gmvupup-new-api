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
import type { TFunction } from 'i18next'

export type SkillMode = 'all' | 'image' | 'video'
export const IMAGE_MODELS = ['gpt-image-2.6', 'nanobanana-2.1']
export const VIDEO_MODELS = [
  'doubao-seedance-2.5',
  'doubao-seedance-2.0',
  'wan3.0-video',
]

export function buildAgentPrompt(
  t: TFunction,
  mode: SkillMode,
  model: string,
  brief: string
) {
  const parts = [
    '# GMVUPUP Agent Skill',
    t(
      'Use the following GMVUPUP skill to complete my creative brief. If no brief is provided, ask what I want to create before making a paid request.'
    ),
    `## ${t('Execution rules')}`,
    t(
      'You need an HTTP or terminal tool to execute requests. If unavailable, say so and provide runnable code; never claim a generation was performed.'
    ),
    t(
      'Read GMVUPUP_API_KEY from environment variables or the agent secret store. If absent, guide me to configure it there. Never ask me to paste a key into chat, print it, or include it in source code.'
    ),
    t(
      'For GMVUPUP API requests use Authorization: Bearer <GMVUPUP_API_KEY>. Use Content-Type: application/json for JSON bodies. Origin: https://gmvupup.top; OpenAI SDK base URL: https://gmvupup.top/v1. Do not duplicate /v1.'
    ),
    t(
      'Before generation, GET https://gmvupup.top/v1/models with this key and check data[].id for the exact requested model. If absent, stop and report that the key does not list it; ask me to check token model restrictions, group and channel availability. Never silently switch models or providers.'
    ),
    t(
      'Use only the requested model and quantity. Confirm missing details that affect cost, including video duration and resolution; use https://gmvupup.top/pricing for current pricing. Do not run extra paid test generations.'
    ),
    t(
      'Keep prompt text and reference URLs as JSON data, not shell commands. Use a JSON serializer or request body file. Do not invent asset URLs or upload local references to another service without permission.'
    ),
    t(
      'On authentication, balance or validation errors, report the HTTP status, safe error message and request ID. After a submission timeout, do not automatically repeat a paid POST. Preserve task IDs and resume polling instead of resubmitting.'
    ),
  ]
  if (mode === 'all' || mode === 'image') {
    parts.push(
      `## ${t('Images')}`,
      mode === 'all' ? IMAGE_MODELS.join(', ') : model,
      t(
        'POST https://gmvupup.top/v1/images/generations. This is a synchronous image request; allow up to 240 seconds. Replace prompt with the creative brief and use the selected exact model ID.'
      ),
      t(
        'Read data[].url or data[].b64_json. Download the actual URL or decode base64 into a file, inspect the file type, and return a real file or preview. Do not invent asynchronous image polling endpoints.'
      ),
      t(
        'For editing with a local reference image, POST multipart/form-data to https://gmvupup.top/v1/images/edits with model, prompt and image file; let the client set the multipart boundary. See https://gmvupup.top/docs/image-edits for model-specific limits.'
      ),
      JSON.stringify(
        {
          model: mode === 'image' ? model : IMAGE_MODELS[0],
          prompt: '<CREATIVE_BRIEF>',
          n: 1,
          size: '1024x1024',
        },
        null,
        2
      )
    )
  }
  if (mode === 'all' || (mode === 'video' && model !== 'wan3.0-video')) {
    parts.push(
      `## ${t('Seedance video')}`,
      mode === 'all' ? VIDEO_MODELS.slice(0, 2).join(', ') : model,
      t(
        'POST https://gmvupup.top/v1/videos with JSON. Use userPrompt, aspectRatio, duration and quality, not prompt/seconds/size or multipart. duration is an integer from 4 to 15; quality is 480p, 720p or 1080p; aspectRatio is 21:9, 16:9, 4:3, 1:1, 3:4 or 9:16. The body below is an example, not permission to use its cost settings.'
      ),
      t(
        'Optional reference fields: refImageUrls (array, at most 9 images including first/last frames), refVideoUrls (array, at most 3), refAudioUrls (array, at most 3), firstImageUrl and lastImageUrl (strings). Use reachable HTTP(S) URLs or existing asset://asset_ID references. A local path or chat attachment is not a public URL. Optional booleans: generateAudio, returnLastFrame. Reference fields must stay in this JSON body.'
      ),
      t(
        'Save response.id. Poll GET https://gmvupup.top/v1/videos/{id} every 5 seconds for status queued/in_progress/completed/failed, for up to 10 minutes per session. On failed, stop and report error. If still running, return the ID and resume instructions; do not resubmit. On completed, GET https://gmvupup.top/v1/videos/{id}/content with the same Bearer key to save the video. Resolve relative artifact paths against https://gmvupup.top.'
      ),
      JSON.stringify(
        {
          model: mode === 'video' ? model : VIDEO_MODELS[0],
          userPrompt: '<CREATIVE_BRIEF>',
          aspectRatio: '16:9',
          duration: 4,
          quality: '480p',
        },
        null,
        2
      )
    )
  }
  if (mode === 'all' || (mode === 'video' && model === 'wan3.0-video')) {
    parts.push(
      `## ${t('Wan video')}`,
      t(
        'POST https://gmvupup.top/ali/api/v1/services/aigc/video-generation/video-synthesis with JSON. This native endpoint is separate from Seedance and is not under /v1. Use input.prompt and parameters; resolution uses uppercase P. The example below requests 2 seconds at 480P; use the user-approved settings.'
      ),
      t(
        'Save output.task_id. Poll GET https://gmvupup.top/ali/api/v1/tasks/{task_id} every 5 seconds, for up to 10 minutes per session. Read output.task_status: PENDING/RUNNING continue, SUCCEEDED downloads output.video_url or output.results.video_url, FAILED/CANCELED/UNKNOWN stop and report. If still running, preserve the ID for resumption. For image-to-video use /ali/api/v1/services/aigc/image2video/video-synthesis with input.img_url and input.prompt; confirm model support before submission.'
      ),
      JSON.stringify(
        {
          model: 'wan3.0-video',
          input: { prompt: '<CREATIVE_BRIEF>' },
          parameters: {
            resolution: '480P',
            ratio: '16:9',
            duration: 2,
            audio: false,
            prompt_extend: false,
          },
        },
        null,
        2
      )
    )
  }
  parts.push(
    t(
      'Return the generated asset, model ID, task ID when present, and actual status. A queued task is not a finished video. Send the GMVUPUP Bearer key only to https://gmvupup.top, never to third-party asset URLs or cross-origin redirects. Download external signed asset URLs without that key.'
    ),
    `## ${t('Creative brief')}`,
    brief.trim() || t('Ask me what to create.')
  )
  return parts.join('\n\n')
}
