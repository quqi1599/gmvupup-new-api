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
import ByteDance from '@lobehub/icons/es/ByteDance/components/Color'
import DeepSeek from '@lobehub/icons/es/DeepSeek/components/Color'
import Gemini from '@lobehub/icons/es/Gemini/components/Color'
import Minimax from '@lobehub/icons/es/Minimax/components/Color'
import NanoBanana from '@lobehub/icons/es/NanoBanana/components/Color'
import OpenAI from '@lobehub/icons/es/OpenAI/components/Mono'
import type { ReactNode } from 'react'

import { getLobeIcon } from '@/lib/lobe-icon'

// Product marks take precedence over a generic provider icon in catalog metadata.
const modelBrands = [
  { pattern: /^gpt[-_ ]?image(?:[-_ ]|$)/i, icon: OpenAI },
  {
    pattern: /^(?:nano[-_ ]?banana|gemini[-_ ]nano[-_ ]banana)(?:[-_ ]|$)/i,
    icon: NanoBanana,
  },
  { pattern: /^deepseek(?:[-_ ]|$)/i, icon: DeepSeek },
  { pattern: /^(?:doubao[-_ ])?seedance(?:[-_ ]|$)/i, icon: ByteDance },
  { pattern: /^minimax(?:[-_ ]|$)/i, icon: Minimax },
  { pattern: /^gemini(?:[-_ ]|$)/i, icon: Gemini },
]

export function GmvupupModelLogo(props: {
  modelName: string
  icon?: string
  size?: number
  fallback?: ReactNode
}) {
  const size = props.size ?? 28
  const brand = modelBrands.find((item) => item.pattern.test(props.modelName))
  let logo: ReactNode = props.fallback ?? (
    <span aria-hidden='true'>
      {props.modelName.charAt(0).toUpperCase() || '?'}
    </span>
  )
  if (brand) {
    logo = <brand.icon size={size} />
  } else if (props.icon) {
    logo = getLobeIcon(props.icon, size)
  }
  return (
    <span
      className='inline-flex shrink-0 items-center justify-center'
      role='img'
      aria-label={props.modelName}
    >
      {logo}
    </span>
  )
}
