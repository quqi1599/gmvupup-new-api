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
import { ArrowUpRight, Braces, KeyRound, Layers3, Workflow } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { API_EXAMPLES } from '../api-examples'

export function IntegrationGuide() {
  const { t } = useTranslation()
  return (
    <section
      className='gmv-developers'
      id='developers'
      aria-labelledby='developers-heading'
    >
      <div className='gmv-container gmv-developer-grid'>
        <div className='gmv-developer-copy'>
          <p className='gmv-eyebrow'>02 / BUILT FOR BUILDERS</p>
          <h2 id='developers-heading'>
            {t('Your ideas,')}
            <br />
            <span>{t('meet your workflow.')}</span>
          </h2>
          <p>
            {t(
              'Bring creative AI into your own product. Manage keys, inspect usage and follow asynchronous tasks from one console.'
            )}
          </p>
          <ul className='gmv-developer-benefits'>
            <li>
              <KeyRound aria-hidden='true' />
              <div>
                <strong>{t('Your key. Your workspace.')}</strong>
                <p>{t('Create and manage API keys in your console.')}</p>
              </div>
            </li>
            <li>
              <Workflow aria-hidden='true' />
              <div>
                <strong>{t('Follow every task.')}</strong>
                <p>
                  {t('Submit once, save the task ID, then check its progress.')}
                </p>
              </div>
            </li>
            <li>
              <Layers3 aria-hidden='true' />
              <div>
                <strong>{t('Keep usage in view.')}</strong>
                <p>
                  {t('Review the model, request and usage details together.')}
                </p>
              </div>
            </li>
          </ul>
          <Link to='/keys' className='gmv-text-link'>
            {t('Manage API keys')}
            <ArrowUpRight aria-hidden='true' />
          </Link>
        </div>
        <div className='gmv-code-window'>
          <div className='gmv-code-title'>
            <div className='gmv-window-dots' aria-hidden='true'>
              <i />
              <i />
              <i />
            </div>
            <span>your-next-idea</span>
            <Braces size={17} aria-hidden='true' />
          </div>
          <Tabs defaultValue='curl'>
            <TabsList className='gmv-code-tabs' aria-label={t('Code language')}>
              <TabsTrigger value='curl'>cURL</TabsTrigger>
              <TabsTrigger value='python'>Python</TabsTrigger>
              <TabsTrigger value='javascript'>JavaScript</TabsTrigger>
            </TabsList>
            {Object.entries(API_EXAMPLES).map(([language, code]) => (
              <TabsContent key={language} value={language}>
                <div className='gmv-code-toolbar'>
                  <span>
                    POST <b>/v1/videos</b>
                  </span>
                  <CopyButton
                    value={code}
                    aria-label={t('Copy code example')}
                    className='gmv-code-copy'
                  />
                </div>
                <pre tabIndex={0} aria-label={t('Video API example')}>
                  <code>{code}</code>
                </pre>
              </TabsContent>
            ))}
          </Tabs>
          <div className='gmv-code-note'>
            <span className='gmv-status-dot' />
            {t('Server-side example · keep your API key private')}
          </div>
        </div>
      </div>
      <p className='gmv-container gmv-integration-note'>
        {t(
          'Video requests require an enabled model and a configured task channel. Check the catalog before calling; a submitted task is not yet a completed video.'
        )}
      </p>
    </section>
  )
}
