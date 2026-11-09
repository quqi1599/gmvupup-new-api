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
import {
  ArrowDownToLine,
  ArrowLeft,
  ArrowRight,
  BookOpen,
  ChevronRight,
  FileJson,
  List,
  LockKeyhole,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { BrandHeader } from '@/features/gmvupup/components/brand-header'
import { useAuthStore } from '@/stores/auth-store'

import '@/styles/gmvupup.css'
import '@/styles/gmvupup-docs.css'

import { DocArticle } from './components/doc-article'
import { DocNavigation } from './components/doc-navigation'
import { DOC_IDS, DOC_PAGES, type DocTopic } from './content'

export function GmvupupDocs(props: { topic: DocTopic }) {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const [menuOpen, setMenuOpen] = useState(false)
  const data = DOC_PAGES[props.topic]
  const index = DOC_IDS.indexOf(props.topic)
  const previous = DOC_IDS[index - 1]
  const next = DOC_IDS[index + 1]
  const toc = [
    {
      id: 'overview',
      label: props.topic === 'quick-start' ? 'Connection details' : 'Overview',
    },
  ]
  if (props.topic === 'videos') {
    toc.push({ id: 'pricing', label: 'Current video pricing' })
  }
  if (data.examples) toc.push({ id: 'request', label: 'Request example' })
  if (data.fields) {
    toc.push({
      id: 'parameters',
      label:
        props.topic === 'errors' ? 'Common status codes' : 'Request parameters',
    })
  }
  if (data.response) toc.push({ id: 'response', label: 'Response example' })
  if (data.followUp) {
    toc.push({ id: 'polling', label: 'Task polling & download' })
  }
  toc.push({ id: 'notes', label: 'Usage notes' })
  if (props.topic === 'quick-start') {
    toc.push({ id: 'next-steps', label: 'Choose your next step' })
  }

  return (
    <div className='gmv-site gmv-docs'>
      <title>
        {t('{{topic}} | GMVUPUP Documentation', { topic: t(data.title) })}
      </title>
      <meta name='description' content={t(data.summary)} />
      <link
        rel='canonical'
        href={`https://gmvupup.top/docs${
          props.topic === 'quick-start' ? '' : `/${props.topic}`
        }`}
      />
      <a className='gmv-skip-link' href='#docs-main'>
        {t('Skip to content')}
      </a>
      <BrandHeader isAuthenticated={Boolean(user)} />
      <div className='gmv-doc-mobile-toolbar'>
        <Sheet open={menuOpen} onOpenChange={setMenuOpen}>
          <SheetTrigger
            render={
              <Button
                variant='ghost'
                aria-label={t('Open documentation menu')}
              />
            }
          >
            <List aria-hidden='true' />
            {t('API documentation')}
          </SheetTrigger>
          <SheetContent side='left' className='gmv-doc-sheet gmv-site'>
            <SheetTitle>{t('API documentation')}</SheetTitle>
            <DocNavigation
              topic={props.topic}
              onNavigate={() => setMenuOpen(false)}
            />
          </SheetContent>
        </Sheet>
        <span>{t(data.title)}</span>
      </div>
      <div className='gmv-doc-layout'>
        <aside className='gmv-doc-sidebar'>
          <div className='gmv-doc-sidebar-title'>
            <BookOpen size={17} aria-hidden='true' />
            {t('Developer guide')}
            <span>v1</span>
          </div>
          <DocNavigation topic={props.topic} />
        </aside>
        <main id='docs-main' className='gmv-doc-main'>
          <nav aria-label={t('Breadcrumb')} className='gmv-doc-breadcrumb'>
            <Link to='/docs'>{t('API documentation')}</Link>
            <ChevronRight size={13} aria-hidden='true' />
            <span>{t(data.title)}</span>
          </nav>
          <header className='gmv-doc-title'>
            <p>
              {t('Developer guide')}
              <span className='gmv-doc-version'>API v1</span>
            </p>
            <h1>{t(data.title)}</h1>
            <div>{t(data.summary)}</div>
            {props.topic === 'quick-start' && (
              <div className='gmv-doc-protocols'>
                <span>REST API</span>
                <span>
                  <LockKeyhole size={12} aria-hidden='true' />
                  {t('Bearer authentication')}
                </span>
                <span>OpenAI Compatible</span>
              </div>
            )}
          </header>
          <DocArticle key={props.topic} topic={props.topic} />
          <nav className='gmv-doc-page-nav' aria-label={t('Page navigation')}>
            {previous ? (
              <Link to='/docs/$topic' params={{ topic: previous }}>
                <ArrowLeft aria-hidden='true' />
                <span>
                  <small>{t('Previous page')}</small>
                  <strong>{t(DOC_PAGES[previous].title)}</strong>
                </span>
              </Link>
            ) : (
              <span />
            )}
            {next && (
              <Link to='/docs/$topic' params={{ topic: next }}>
                <span>
                  <small>{t('Next page')}</small>
                  <strong>{t(DOC_PAGES[next].title)}</strong>
                </span>
                <ArrowRight aria-hidden='true' />
              </Link>
            )}
          </nav>
          <footer className='gmv-doc-footer'>
            <span>GMVUPUP / gmvupup.top</span>
            <p>
              Frontend design and development by{' '}
              <a
                href='https://github.com/QuantumNous/new-api'
                target='_blank'
                rel='noopener noreferrer'
              >
                New API contributors.
              </a>
            </p>
          </footer>
        </main>
        <aside className='gmv-doc-toc'>
          <p>{t('On this page')}</p>
          <nav aria-label={t('On this page')}>
            {toc.map((item) => (
              <a href={`#${item.id}`} key={item.id}>
                {t(item.label)}
              </a>
            ))}
          </nav>
          <div className='gmv-doc-downloads'>
            <FileJson aria-hidden='true' />
            <strong>{t('Build something worth creating.')}</strong>
            <a href='/gmvupup/docs/openapi.json' download>
              <FileJson size={15} aria-hidden='true' />
              {t('Download OpenAPI')}
            </a>
            <a href='/gmvupup/docs/guide.md' download>
              <ArrowDownToLine size={15} aria-hidden='true' />
              {t('Download guide')}
            </a>
          </div>
          <Link to='/pricing' className='gmv-doc-pricing-link'>
            {t('View live pricing')}
            <ArrowRight size={14} aria-hidden='true' />
          </Link>
        </aside>
      </div>
      <div className='gmv-doc-mobile-downloads'>
        <a href='/gmvupup/docs/openapi.json' download>
          {t('Download OpenAPI')}
        </a>
        <a href='/gmvupup/docs/guide.md' download>
          {t('Download guide')}
        </a>
      </div>
    </div>
  )
}
