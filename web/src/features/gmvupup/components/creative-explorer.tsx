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
// GMVUPUP brand customization, 2026-11-08.
import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  ArrowUpRight,
  Clapperboard,
  Image,
  ScanLine,
  Sparkles,
  Type,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

export function CreativeExplorer() {
  const { t } = useTranslation()
  return (
    <section
      className='gmv-section gmv-container'
      id='possibilities'
      aria-labelledby='possibilities-heading'
    >
      <div className='gmv-section-heading'>
        <div>
          <p className='gmv-eyebrow'>01 / {t('MADE FOR YOUR IDEAS')}</p>
          <h2 id='possibilities-heading'>
            {t('One idea. More possibilities.')}
            <span className='gmv-period'>↗</span>
          </h2>
        </div>
        <p>
          {t('From a product shot to a moving story.')}
          <br />
          {t('Find the right format for your next idea.')}
        </p>
      </div>
      <Tabs defaultValue='video' className='gmv-explorer'>
        <TabsList
          className='gmv-creative-tabs'
          aria-label={t('Creative formats')}
        >
          <TabsTrigger value='video'>
            <Clapperboard aria-hidden='true' />
            {t('Video creation')}
          </TabsTrigger>
          <TabsTrigger value='image'>
            <Image aria-hidden='true' />
            {t('Image creation')}
          </TabsTrigger>
          <TabsTrigger value='copy'>
            <Type aria-hidden='true' />
            {t('Creative writing')}
          </TabsTrigger>
        </TabsList>
        <TabsContent value='video' className='gmv-creative-panel'>
          <div className='gmv-story-visual gmv-motion-visual'>
            <img
              src='/gmvupup/creative-sneaker.webp'
              alt={t('Lavender sneaker floating above lilac dunes')}
              width='1254'
              height='1254'
              loading='lazy'
            />
            <span className='gmv-visual-label'>STORYBOARD / 001</span>
            <div className='gmv-frame-corners' aria-hidden='true'>
              <ScanLine />
            </div>
            <span className='gmv-concept-label'>{t('Concept artwork')}</span>
          </div>
          <div className='gmv-story-copy'>
            <p className='gmv-eyebrow'>IMAGE → MOTION</p>
            <h3>
              {t('Give your product')}
              <br />
              {t('a new way to move.')}
            </h3>
            <p>
              {t(
                'Turn a product reference and a scene description into a video concept. Explore product reveals, social shorts and brand storytelling.'
              )}
            </p>
            <div className='gmv-tags'>
              <span>{t('Product films')}</span>
              <span>{t('Social content')}</span>
              <span>{t('Brand stories')}</span>
            </div>
            <a href='#developers' className='gmv-text-link'>
              {t('Explore video integration')}
              <ArrowUpRight aria-hidden='true' />
            </a>
          </div>
        </TabsContent>
        <TabsContent value='image' className='gmv-creative-panel'>
          <div className='gmv-story-visual gmv-image-visual'>
            <img
              src='/gmvupup/creative-sneaker.webp'
              alt={t('Lavender sneaker floating above lilac dunes')}
              width='1254'
              height='1254'
              loading='lazy'
            />
            <span className='gmv-visual-label'>CAMPAIGN / 001</span>
            <span className='gmv-concept-label'>{t('Concept artwork')}</span>
          </div>
          <div className='gmv-story-copy'>
            <p className='gmv-eyebrow'>PROMPT → VISUAL</p>
            <h3>
              {t('Make the first impression')}
              <br />
              {t('worth a second look.')}
            </h3>
            <p>
              {t(
                'Explore product compositions, campaign visuals and new art directions. Describe the subject, lighting and materials to make your brief concrete.'
              )}
            </p>
            <div className='gmv-tags'>
              <span>{t('Product visuals')}</span>
              <span>{t('Campaign concepts')}</span>
              <span>{t('Art direction')}</span>
            </div>
            <Button
              className='gmv-text-link'
              variant='link'
              render={<Link to='/pricing' />}
            >
              {t('Browse model catalog')}
              <ArrowUpRight aria-hidden='true' />
            </Button>
          </div>
        </TabsContent>
        <TabsContent value='copy' className='gmv-creative-panel'>
          <div className='gmv-writing-visual'>
            <Sparkles aria-hidden='true' />
            <p>
              LESS SCROLL.
              <br />
              <em>MORE SOUL.</em>
            </p>
            <div>{t('A light step. A bold new direction.')}</div>
            <span className='gmv-concept-label'>{t('Illustrative copy')}</span>
          </div>
          <div className='gmv-story-copy'>
            <p className='gmv-eyebrow'>BRIEF → STORY</p>
            <h3>
              {t('Good content starts')}
              <br />
              {t('with the right words.')}
            </h3>
            <p>
              {t(
                'Develop product angles, opening hooks and shot-by-shot scripts. Keep your brand voice consistent as you explore different audiences and formats.'
              )}
            </p>
            <div className='gmv-tags'>
              <span>{t('Video scripts')}</span>
              <span>{t('Product copy')}</span>
              <span>{t('Creative briefs')}</span>
            </div>
            <Button
              className='gmv-text-link'
              variant='link'
              render={<Link to='/pricing' />}
            >
              {t('Browse model catalog')}
              <ArrowUpRight aria-hidden='true' />
            </Button>
          </div>
        </TabsContent>
      </Tabs>
      <div className='gmv-catalog-note'>
        <span>
          {t(
            'Creative examples only. Available models and prices are listed in the console.'
          )}
        </span>
        <Link to='/pricing'>
          {t('View catalog')}
          <ArrowRight aria-hidden='true' />
        </Link>
      </div>
    </section>
  )
}
