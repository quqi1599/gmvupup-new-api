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
import { ArrowUpRight, Image, Video, MessageSquare } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { GmvupupWordmark } from './components/wordmark'

import '@/styles/gmvupup-landing.css'

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
    <div className='gmv-landing'>
      <title>{t('GMVUPUP · AI for ecommerce | New API')}</title>
      <meta
        name='description'
        content={t(
          'Product images, creative videos and copywriting. Find models and clear pricing in one console.'
        )}
      />
      <header className='gmv-landing-header'>
        <GmvupupWordmark />
        <span>API</span>
        <p>{t('Built for ecommerce')}</p>
      </header>
      <main className='gmv-landing-main'>
        <section className='gmv-landing-copy'>
          <p className='gmv-landing-eyebrow'>
            <span />
            {t('AI for your next idea')}
          </p>
          <h1>
            Gmv<span>UpUp</span> <small>API</small>
          </h1>
          <h2>
            {t('Built for ecommerce.')}
            <br />
            {t('AI models, all in one place.')}
          </h2>
          <p className='gmv-landing-description'>
            {t(
              'From product images to campaign ideas. Find the right model and see the price before you start.'
            )}
          </p>
          <Button className='gmv-landing-cta' render={<Link to='/dashboard' />}>
            {t('Enter console')}
            <ArrowUpRight aria-hidden='true' />
          </Button>
          <p className='gmv-landing-hint'>
            {t('Models, prices and API keys in one console.')}
          </p>
          <div className='gmv-landing-capabilities'>
            <span>
              <Image />
              {t('Product images')}
            </span>
            <span>
              <Video />
              {t('Creative videos')}
            </span>
            <span>
              <MessageSquare />
              {t('Copywriting & reasoning')}
            </span>
          </div>
        </section>
        <section
          className='gmv-landing-visual'
          aria-label={t('AI model families')}
        >
          <p>ONE API. MORE POSSIBILITIES.</p>
          <div className='gmv-landing-models'>
            <div className='gmv-landing-orbit' aria-hidden='true' />
            <div className='gmv-landing-center' aria-hidden='true'>
              Up<span>Up</span>
              <small>CREATE · GROW · REPEAT</small>
            </div>
            {models.map((model, index) => (
              <div
                className={`gmv-landing-model gmv-landing-model-${index}`}
                key={model.name}
              >
                <span>
                  <model.icon size={30} />
                </span>
                <div>
                  <strong>{model.name}</strong>
                  <small>{t(model.category)}</small>
                </div>
              </div>
            ))}
          </div>
          <small>
            {t('Available models and prices are listed in the console.')}
          </small>
        </section>
      </main>
      <footer className='gmv-landing-footer'>
        <span>GmvUpUp · {t('Built for ecommerce')}</span>
        <span>Frontend design and development by New API contributors.</span>
      </footer>
    </div>
  )
}
