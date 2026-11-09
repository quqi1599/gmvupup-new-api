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
  ArrowUpRight,
  Bot,
  Image,
  KeyRound,
  LayoutGrid,
  MessageSquare,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  CodeBlock,
  CodeBlockCopyButton,
} from '@/components/ai-elements/code-block'
import { CopyButton } from '@/components/copy-button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { API_BASE_URL, DOC_PAGES, type DocTopic } from '../content'
import { AgentSkills } from './agent-skills'
import { CodeExamples } from './code-examples'
import { ModelCatalog } from './model-catalog'

export function DocArticle(props: { topic: DocTopic }) {
  const { t } = useTranslation()
  const data = DOC_PAGES[props.topic]
  const quickStart = props.topic === 'quick-start'
  const errors = props.topic === 'errors'
  return (
    <>
      <section id='overview' className='gmv-doc-overview'>
        {quickStart && (
          <>
            <div className='gmv-doc-base-url'>
              <div>
                <span>{t('API Base URL')}</span>
                <code>{API_BASE_URL}</code>
                <p>{t('For OpenAI-compatible clients. Includes /v1.')}</p>
              </div>
              <CopyButton
                value={API_BASE_URL}
                aria-label={t('Copy Base URL')}
              />
            </div>
            <div className='gmv-doc-steps'>
              <div>
                <span>01</span>
                <h3>{t('Create an API key')}</h3>
                <p>
                  {t(
                    'Open the console and prepare a key with access to your chosen model.'
                  )}
                </p>
                <Link to='/keys'>
                  {t('Console')}
                  <ArrowUpRight aria-hidden='true' />
                </Link>
              </div>
              <div>
                <span>02</span>
                <h3>{t('Confirm your model')}</h3>
                <p>{t('Copy an exact model ID from the current catalog.')}</p>
                <Link to='/docs/$topic' params={{ topic: 'models' }}>
                  {t('Model directory')}
                  <ArrowUpRight aria-hidden='true' />
                </Link>
              </div>
              <div>
                <span>03</span>
                <h3>{t('Make your first request')}</h3>
                <p>{t('Start by listing the models available to your key.')}</p>
                <a href='#request'>
                  {t('Request example')}
                  <ArrowUpRight aria-hidden='true' />
                </a>
              </div>
            </div>
            <p className='gmv-doc-paragraph'>
              {t(
                'Use full request URLs in cURL. Do not append /v1 twice in an SDK.'
              )}
            </p>
          </>
        )}
        {data.conditional && (
          <div className='gmv-doc-notice'>{t(data.conditional)}</div>
        )}
        {data.body?.map((text) => (
          <p className='gmv-doc-paragraph' key={text}>
            {t(text)}
          </p>
        ))}
        {data.endpoint && (
          <div className='gmv-doc-endpoint'>
            <span className={data.method === 'POST' ? 'is-post' : ''}>
              {data.method}
            </span>
            <code>{data.endpoint}</code>
            <CopyButton
              value={`https://gmvupup.top${data.endpoint}`}
              aria-label={t('Copy Base URL')}
            />
          </div>
        )}
        {props.topic === 'models' && (
          <div className='gmv-doc-live-catalog'>
            <h2>{t('Live model catalog')}</h2>
            <ModelCatalog />
          </div>
        )}
      </section>
      {props.topic === 'agent-skills' && <AgentSkills />}
      {data.examples && (
        <section id='request' className='gmv-doc-section'>
          <div className='gmv-doc-section-title'>
            <h2>{quickStart ? t('First request') : t('Request example')}</h2>
            <span>HTTP / JSON</span>
          </div>
          <CodeExamples
            key={props.topic}
            examples={data.examples}
            models={data.models}
          />
        </section>
      )}
      {data.fields && (
        <section id='parameters' className='gmv-doc-section'>
          <h2>{errors ? t('Common status codes') : t('Request parameters')}</h2>
          <div
            className={
              errors
                ? 'gmv-doc-parameter-table is-errors'
                : 'gmv-doc-parameter-table'
            }
            role='region'
            aria-label={t('Request parameters')}
          >
            <Table tabIndex={0}>
              <TableHeader>
                <TableRow>
                  <TableHead>
                    {errors ? t('HTTP status') : t('Field')}
                  </TableHead>
                  {!errors && <TableHead>{t('Type')}</TableHead>}
                  <TableHead>{t('Description')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.fields.map((field) => (
                  <TableRow key={field.name}>
                    <TableCell>
                      <code>{field.name}</code>
                      {field.required && (
                        <span className='gmv-doc-required'>
                          {t('Required')}
                        </span>
                      )}
                    </TableCell>
                    {!errors && (
                      <TableCell>
                        <span className='gmv-doc-field-type'>{field.type}</span>
                      </TableCell>
                    )}
                    <TableCell>{t(field.description)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </section>
      )}
      {data.response && (
        <section id='response' className='gmv-doc-section'>
          <h2>{t('Response example')}</h2>
          <p className='gmv-doc-paragraph'>
            {t(
              'This response illustrates the structure; it is not a live generation result.'
            )}
          </p>
          <CodeBlock
            className='gmv-doc-response'
            language='json'
            code={data.response}
            showToolbar
            enableCollapse={false}
          >
            <CodeBlockCopyButton />
          </CodeBlock>
        </section>
      )}
      {data.followUp && (
        <section id='polling' className='gmv-doc-section'>
          <h2>{t('Task polling & download')}</h2>
          <CodeBlock
            className='gmv-doc-code'
            language='bash'
            code={data.followUp}
            showToolbar
            enableCollapse={false}
          >
            <CodeBlockCopyButton />
          </CodeBlock>
        </section>
      )}
      <section id='notes' className='gmv-doc-section gmv-doc-notes'>
        <h2>
          <KeyRound aria-hidden='true' />
          {t('Usage notes')}
        </h2>
        <ul>
          {data.notes.map((text) => (
            <li key={text}>{t(text)}</li>
          ))}
        </ul>
      </section>
      {quickStart && (
        <section id='next-steps' className='gmv-doc-section'>
          <h2>{t('Choose your next step')}</h2>
          <div className='gmv-doc-next-cards'>
            <Link to='/docs/$topic' params={{ topic: 'agent-skills' }}>
              <Bot aria-hidden='true' />
              <strong>{t('Agent Skills')}</strong>
              <p>{t('Let your agent create with GMVUPUP.')}</p>
              <ArrowUpRight aria-hidden='true' />
            </Link>
            <Link to='/docs/$topic' params={{ topic: 'images' }}>
              <Image aria-hidden='true' />
              <strong>{t('Image generation API')}</strong>
              <p>{t('Explore image generation and editing.')}</p>
              <ArrowUpRight aria-hidden='true' />
            </Link>
            <Link to='/docs/$topic' params={{ topic: 'chat' }}>
              <MessageSquare aria-hidden='true' />
              <strong>{t('Chat API')}</strong>
              <p>{t('Connect a conversation to your product.')}</p>
              <ArrowUpRight aria-hidden='true' />
            </Link>
            <Link to='/docs/$topic' params={{ topic: 'models' }}>
              <LayoutGrid aria-hidden='true' />
              <strong>{t('Model directory')}</strong>
              <p>{t('Find models and their supported interfaces.')}</p>
              <ArrowUpRight aria-hidden='true' />
            </Link>
            <Link to='/docs/$topic' params={{ topic: 'authentication' }}>
              <KeyRound aria-hidden='true' />
              <strong>{t('Authentication')}</strong>
              <p>{t('Keep credentials in the right place.')}</p>
              <ArrowUpRight aria-hidden='true' />
            </Link>
          </div>
        </section>
      )}
    </>
  )
}
