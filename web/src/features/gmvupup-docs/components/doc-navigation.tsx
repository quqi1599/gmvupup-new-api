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
import { Link, useNavigate } from '@tanstack/react-router'
import {
  BookOpen,
  Bot,
  Film,
  Image,
  KeyRound,
  LayoutGrid,
  MessageSquare,
  ScanLine,
  Search,
  TriangleAlert,
  X,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { DOC_IDS, DOC_PAGES, type DocTopic } from '../content'

const icons = {
  'quick-start': BookOpen,
  'agent-skills': Bot,
  authentication: KeyRound,
  models: LayoutGrid,
  images: Image,
  'image-edits': ScanLine,
  chat: MessageSquare,
  videos: Film,
  errors: TriangleAlert,
}
const groups = [
  { id: 'start', label: 'Getting started' },
  { id: 'reference', label: 'API reference' },
  { id: 'help', label: 'Troubleshooting' },
]

export function DocNavigation(props: {
  topic: DocTopic
  onNavigate?: () => void
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const query = search.trim().toLocaleLowerCase()
  const matches = DOC_IDS.filter((id) =>
    [
      t(DOC_PAGES[id].title),
      DOC_PAGES[id].title,
      t(DOC_PAGES[id].summary),
      DOC_PAGES[id].keywords,
    ]
      .join(' ')
      .toLocaleLowerCase()
      .includes(query)
  )
  return (
    <div className='gmv-doc-navigation'>
      <form
        className='gmv-doc-search'
        onSubmit={(event) => {
          event.preventDefault()
          if (matches[0]) {
            void navigate({ to: '/docs/$topic', params: { topic: matches[0] } })
            setSearch('')
            props.onNavigate?.()
          }
        }}
      >
        <Search aria-hidden='true' />
        <Input
          type='search'
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t('Search documentation')}
          aria-label={t('Search documentation')}
        />
        {search && (
          <Button
            variant='ghost'
            size='icon-sm'
            type='button'
            onClick={() => setSearch('')}
            aria-label={t('Clear search')}
          >
            <X aria-hidden='true' />
          </Button>
        )}
      </form>
      <nav aria-label={t('Documentation navigation')}>
        {groups.map((group) => {
          const ids = matches.filter((id) => DOC_PAGES[id].group === group.id)
          if (!ids.length) return null
          return (
            <div className='gmv-doc-nav-group' key={group.id}>
              <p>{t(group.label)}</p>
              {ids.map((id) => {
                const Icon = icons[id]
                return (
                  <Link
                    key={id}
                    to='/docs/$topic'
                    params={{ topic: id }}
                    aria-current={props.topic === id ? 'page' : undefined}
                    className={
                      props.topic === id
                        ? 'gmv-doc-nav-link is-current'
                        : 'gmv-doc-nav-link'
                    }
                    onClick={() => {
                      setSearch('')
                      props.onNavigate?.()
                    }}
                  >
                    <Icon aria-hidden='true' />
                    <span>{t(DOC_PAGES[id].title)}</span>
                    {DOC_PAGES[id].method && (
                      <small>{DOC_PAGES[id].method}</small>
                    )}
                  </Link>
                )
              })}
            </div>
          )
        })}
        {matches.length === 0 && (
          <div className='gmv-doc-no-results' role='status'>
            <strong>{t('No matching documents')}</strong>
            <p>{t('Try a model name, endpoint or another keyword.')}</p>
          </div>
        )}
      </nav>
      <div className='gmv-doc-sidebar-bottom'>
        <span>GMVUPUP / API v1</span>
        <Link to='/keys'>
          {t('Create an API key')}
          <KeyRound size={14} aria-hidden='true' />
        </Link>
      </div>
    </div>
  )
}
