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
import { Link } from '@tanstack/react-router'
import { ArrowUpRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { GmvupupWordmark } from './components/wordmark'

import '@/styles/gmvupup-landing.css'
import '@/styles/gmvupup-universe.css'

const models = [
  { name: 'GPT Image', icon: OpenAI, category: 'Product images' },
  { name: 'Nano Banana', icon: NanoBanana, category: 'Image editing' },
  {
    name: 'DeepSeek',
    icon: DeepSeek,
    category: 'Copywriting & reasoning',
  },
  { name: 'Seedance', icon: ByteDance, category: 'Creative videos' },
  { name: 'MiniMax', icon: Minimax, category: 'Creative videos' },
  { name: 'Gemini', icon: Gemini, category: 'Multimodal AI' },
]

export function GmvupupHome(_props: { isAuthenticated: boolean }) {
  const { t } = useTranslation()
  return (
    <div className='gmv-universe-home'>
      <title>{t('GMVUPUP · AI for ecommerce | New API')}</title>
      <meta
        name='description'
        content={t(
          'Product images, creative videos and copywriting. Find models and clear pricing in one console.'
        )}
      />
      <header className='gmv-universe-header'>
        <GmvupupWordmark />
        <div
          className='gmv-universe-models'
          aria-label={t('AI model families')}
        >
          {models.map((model) => (
            <span
              key={model.name}
              role='img'
              aria-label={model.name}
              title={model.name}
            >
              <model.icon size={22} aria-hidden='true' />
            </span>
          ))}
        </div>
        <Button
          className='gmv-universe-console'
          render={<Link to='/dashboard' />}
        >
          {t('Enter console')}
          <ArrowUpRight aria-hidden='true' />
        </Button>
      </header>
      <main className='gmv-universe-stage'>
        <h1 className='sr-only'>GmvUpUp — {t('Built for ecommerce')}</h1>
        <iframe
          title={t('GMV ecommerce universe')}
          src='/gmv-universe.html?embedded=1'
          sandbox='allow-scripts allow-downloads'
          className='gmv-universe-frame'
        />
      </main>
      <footer className='gmv-universe-credit'>
        <span>GmvUpUp · {t('Built for ecommerce')}</span>
        <span>Frontend design and development by New API contributors.</span>
      </footer>
    </div>
  )
}
