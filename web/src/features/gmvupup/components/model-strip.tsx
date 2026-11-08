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
import { Link } from '@tanstack/react-router'
import { ArrowUpRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { getLobeIcon } from '@/lib/lobe-icon'

const models = [
  { name: 'GPT-image-2.6', icon: 'OpenAI' },
  { name: 'nanobanana-2.1', icon: 'NanoBanana.Color' },
  { name: 'seedance-2.5', icon: 'ByteDance.Color' },
  { name: 'MiniMax-H3', icon: 'Minimax.Color' },
  { name: 'Gemini-3.8', icon: 'Gemini.Color' },
  { name: 'DeepSeek-V4.1-flash', icon: 'DeepSeek.Color' },
]

export function ModelStrip() {
  const { t } = useTranslation()
  return (
    <section
      className='gmv-model-strip gmv-container'
      aria-label={t('Model ecosystem')}
    >
      <div className='gmv-model-strip-heading'>
        <p>{t('FAMILIAR MODELS. FRESH POSSIBILITIES.')}</p>
        <Link to='/pricing'>
          {t('Models & pricing')}
          <ArrowUpRight aria-hidden='true' />
        </Link>
      </div>
      <ul className='gmv-model-list'>
        {models.map((model) => (
          <li key={model.name}>
            <span className='gmv-model-icon' aria-hidden='true'>
              {getLobeIcon(model.icon, 31)}
            </span>
            <span>{model.name}</span>
          </li>
        ))}
      </ul>
    </section>
  )
}
