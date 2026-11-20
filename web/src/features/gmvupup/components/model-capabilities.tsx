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
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import type { PricingModel } from '@/features/pricing/types'

import { getCustomerModelName, isNativeSeedanceModel } from '../customer-models'

export function CustomerModelCapabilities(props: {
  model: PricingModel
  fallback?: ReactNode
}) {
  const { t } = useTranslation()
  let details: string[]
  if (
    isNativeSeedanceModel(props.model) &&
    getCustomerModelName(props.model) === 'Seedance 2.5'
  ) {
    details = [
      t(
        'Create videos from text, first or last frames, or reference images, video and audio.'
      ),
      t(
        '480p, 720p or 1080p, up to 30 seconds. Set the aspect ratio and choose whether to generate audio.'
      ),
      t('Submit a Doubao video task and query the result by task ID.'),
    ]
  } else {
    switch (props.model.model_name) {
      case 'MiniMax-H3':
        details = [
          t(
            'Create videos from text, first or last frames, or reference images, video and audio.'
          ),
          t(
            '768P or 2K, 4–15 seconds. Use frame images or reference media in a single task.'
          ),
          t(
            'Up to 9 images, 3 reference videos and 3 reference audio clips. Reference videos total up to 15 seconds.'
          ),
        ]
        break
      case 'gpt-image-2.6':
        details = [
          t('Generate images from text or edit an uploaded reference image.'),
          t(
            'Choose 1K, 2K or 4K and set the aspect ratio. All three sizes have the same price.'
          ),
          t(
            'Supports OpenAI-compatible image generation and image edit endpoints.'
          ),
        ]
        break
      case 'nanobanana-2.1':
        details = [
          t(
            'Generate images from text, edit reference images, or combine multiple references.'
          ),
          t(
            'Choose 1K, 2K or 4K and set the aspect ratio. All three sizes have the same price.'
          ),
          t('Supports common ratios including 1:1, 16:9 and 9:16.'),
        ]
        break
      case 'gemini-3.8-flash':
        details = [
          t(
            'Text generation and multimodal understanding for scripts, storyboards and prompts.'
          ),
          t('Supports the Gemini API and OpenAI Chat Completions.'),
        ]
        break
      case 'deepseek-v4.1-flash':
        details = [
          t(
            'Question answering, writing, coding assistance and image understanding.'
          ),
          t(
            'Supports reasoning mode and tool calls through OpenAI Chat Completions.'
          ),
        ]
        break
      default:
        return props.fallback ?? null
    }
  }

  return (
    <div className='min-w-0 space-y-1.5 text-xs leading-relaxed'>
      <p className='font-semibold'>{t('Capabilities')}</p>
      <ul className='text-muted-foreground list-disc space-y-1 ps-4'>
        {details.map((detail) => (
          <li key={detail}>{detail}</li>
        ))}
      </ul>
    </div>
  )
}
