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
  ArrowDown,
  ArrowRight,
  ArrowUpRight,
  Braces,
  Check,
  Clapperboard,
  Image,
  Sparkles,
  Type,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

import { BrandHeader, BrandMark } from './components/brand-header'
import { CreativeExplorer } from './components/creative-explorer'
import { IntegrationGuide } from './components/integration-guide'
import { ModelStrip } from './components/model-strip'

import '@/styles/gmvupup.css'

export function GmvupupHome(props: { isAuthenticated: boolean }) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const startDestination = props.isAuthenticated ? '/dashboard' : '/sign-up'
  return (
    <div className='gmv-site'>
      <title>{`${t('GMVUPUP · Creative AI, moving forward')} | New API`}</title>
      <meta
        name='description'
        content={t(
          'GMVUPUP brings video, image and language creation into your workflow. Explore creative possibilities, model pricing and API integration.'
        )}
      />
      <link rel='canonical' href='https://gmvupup.top/' />
      <a className='gmv-skip-link' href='#main-content'>
        {t('Skip to content')}
      </a>
      <BrandHeader isAuthenticated={props.isAuthenticated} />
      <main id='main-content'>
        <section
          className='gmv-hero gmv-container'
          aria-labelledby='hero-heading'
        >
          <div className='gmv-hero-copy'>
            <p className='gmv-hero-eyebrow'>
              <span className='gmv-status-dot' />
              {t('AI CREATIVE API PLATFORM')}
              <span className='gmv-eyebrow-line' />
            </p>
            <h1 id='hero-heading'>
              {t('Let your ideas')}
              <br />
              <span className='gmv-headline-bottom'>
                {t('go')}{' '}
                <span className='gmv-up-word'>
                  UP.
                  <svg viewBox='0 0 196 14' aria-hidden='true'>
                    <path d='M3 10 Q75 -1 190 7' />
                  </svg>
                </span>
                <ArrowUpRight aria-hidden='true' />
              </span>
            </h1>
            <p className='gmv-hero-description'>
              {t('Video. Images. Words. One creative starting point.')}
              <br />
              {t(
                'Connect your next big idea to the tools that bring it to life.'
              )}
            </p>
            <div className='gmv-hero-actions'>
              <Button
                className='gmv-button gmv-button-large'
                render={<Link to={startDestination} />}
              >
                {props.isAuthenticated
                  ? t('Open console')
                  : t('Start creating')}
                <ArrowUpRight aria-hidden='true' />
              </Button>
              <a href='#possibilities' className='gmv-secondary-link'>
                {t('Explore possibilities')}
                <ArrowDown aria-hidden='true' />
              </a>
            </div>
            <div className='gmv-hero-footnotes'>
              <span>
                <Check aria-hidden='true' />
                {t('One workspace')}
              </span>
              <span>
                <Check aria-hidden='true' />
                {t('API access')}
              </span>
              <span>
                <Check aria-hidden='true' />
                {t('Usage visibility')}
              </span>
            </div>
          </div>
          <div className='gmv-hero-art'>
            <div className='gmv-art-topline'>
              <span>
                <Sparkles aria-hidden='true' />
                {t('IMAGINATION, IN FRAME')}
              </span>
              <span>GMV / 001</span>
            </div>
            <figure className='gmv-main-art'>
              <img
                src='/gmvupup/creative-sneaker.webp'
                alt={t('Lavender sneaker floating above lilac dunes')}
                width='1254'
                height='1254'
                fetchPriority='high'
              />
              <figcaption>{t('Concept artwork')}</figcaption>
            </figure>
            <div className='gmv-art-sticker' aria-hidden='true'>
              <ArrowUpRight />
              <span>
                MAKE
                <br />
                IT MOVE.
              </span>
            </div>
            <div className='gmv-prompt-card'>
              <span className='gmv-prompt-icon'>
                <Sparkles aria-hidden='true' />
              </span>
              <div>
                <span className='gmv-prompt-heading'>
                  {t('It starts with an idea')}
                </span>
                <p>{t('A sneaker. Lilac dunes. No limits.')}</p>
              </div>
              <span className='gmv-prompt-arrow' aria-hidden='true'>
                <ArrowRight />
              </span>
            </div>
          </div>
        </section>
        <ModelStrip />
        <div className='gmv-capabilities'>
          <div className='gmv-container gmv-capabilities-inner'>
            <p>{t('A TOOLKIT FOR WHAT’S NEXT')}</p>
            <span>
              <Clapperboard aria-hidden='true' />
              {t('Video creation')}
            </span>
            <span>
              <Image aria-hidden='true' />
              {t('Image creation')}
            </span>
            <span>
              <Type aria-hidden='true' />
              {t('Creative writing')}
            </span>
            <span>
              <Braces aria-hidden='true' />
              {t('API integration')}
            </span>
          </div>
        </div>
        <CreativeExplorer />
        <IntegrationGuide />
        <section
          className='gmv-faq gmv-section gmv-container'
          id='questions'
          aria-labelledby='questions-heading'
        >
          <div>
            <p className='gmv-eyebrow'>03 / GOOD TO KNOW</p>
            <h2 id='questions-heading'>
              {t('Before your')}
              <br />
              {t('next big idea.')}
            </h2>
            <p>{t('A few things to get you started.')}</p>
          </div>
          <Accordion className='gmv-accordion'>
            <AccordionItem value='start'>
              <AccordionTrigger>{t('How do I get started?')}</AccordionTrigger>
              <AccordionContent>
                {t(
                  'Create an account, check the model catalog and pricing, then generate an API key in the console. Confirm your available balance and model access before submitting a request.'
                )}
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value='models'>
              <AccordionTrigger>
                {t('Which models can I use?')}
              </AccordionTrigger>
              <AccordionContent>
                {t(
                  'Availability depends on the models enabled for your account. The live model catalog and console are the source of truth for supported models, parameters and prices.'
                )}{' '}
                <Link to='/pricing'>{t('View catalog')}</Link>
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value='video'>
              <AccordionTrigger>
                {t('How does video generation work?')}
              </AccordionTrigger>
              <AccordionContent>
                {t(
                  'Video generation is asynchronous: submit a request, keep the returned task ID, and poll its status. Download the result after completion. Do not resubmit automatically when a request times out.'
                )}
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value='examples'>
              <AccordionTrigger>
                {t('Are the visuals on this page real customer results?')}
              </AccordionTrigger>
              <AccordionContent>
                {t(
                  'No. These are concept visuals made for this website, illustrating creative directions. They are not customer work or a guarantee of a particular model output.'
                )}
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </section>
        <section className='gmv-cta gmv-container'>
          <div>
            <p className='gmv-eyebrow'>NEXT IDEA. NEXT LEVEL.</p>
            <h2>{t('What will you create next?')}</h2>
          </div>
          <Button
            className='gmv-button gmv-button-large'
            render={<Link to={startDestination} />}
          >
            {t('Start with GMVUPUP')}
            <ArrowUpRight aria-hidden='true' />
          </Button>
          <ArrowUpRight className='gmv-cta-arrow' aria-hidden='true' />
        </section>
      </main>
      <footer className='gmv-footer gmv-container'>
        <div className='gmv-footer-top'>
          <Link to='/' className='gmv-brand'>
            <BrandMark />
            <span>GMVUPUP</span>
          </Link>
          <p>{t('Ideas up. Possibilities up.')}</p>
          <a href='https://gmvupup.top'>
            gmvupup.top <ArrowUpRight size={15} aria-hidden='true' />
          </a>
        </div>
        <div className='gmv-footer-bottom'>
          <span>© {new Date().getFullYear()} GMVUPUP</span>
          <div>
            {status?.user_agreement_enabled && (
              <Link to='/user-agreement'>{t('User Agreement')}</Link>
            )}
            {status?.privacy_policy_enabled && (
              <Link to='/privacy-policy'>{t('Privacy Policy')}</Link>
            )}
            <Link to='/pricing'>{t('Models & pricing')}</Link>
            <Link to='/docs'>{t('API integration')}</Link>
          </div>
        </div>
        <div className='gmv-attribution'>
          Frontend design and development by{' '}
          <a
            href='https://github.com/QuantumNous/new-api'
            target='_blank'
            rel='noopener noreferrer'
          >
            New API contributors.
          </a>{' '}
          <span>{t('GMVUPUP brand customization.')}</span>
        </div>
      </footer>
    </div>
  )
}
