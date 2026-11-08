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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  CodeBlock,
  CodeBlockCopyButton,
} from '@/components/ai-elements/code-block'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import type { CodeExamples as Examples } from '../content'

export function CodeExamples(props: { examples: Examples; models?: string[] }) {
  const { t } = useTranslation()
  const [model, setModel] = useState(props.models?.[0] ?? '')
  return (
    <div className='gmv-doc-examples'>
      {props.models && (
        <div className='gmv-doc-model-select'>
          <span>{t('Example model')}</span>
          <NativeSelect
            value={model}
            aria-label={t('Example model')}
            onChange={(event) => setModel(event.target.value)}
          >
            {props.models.map((name) => (
              <NativeSelectOption key={name} value={name}>
                {name}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        </div>
      )}
      <Tabs defaultValue='curl'>
        <TabsList className='gmv-doc-code-tabs' aria-label={t('Code language')}>
          <TabsTrigger value='curl'>cURL</TabsTrigger>
          <TabsTrigger value='python'>Python</TabsTrigger>
          <TabsTrigger value='javascript'>JavaScript</TabsTrigger>
        </TabsList>
        {Object.entries(props.examples).map(([language, source]) => (
          <TabsContent key={language} value={language}>
            <CodeBlock
              className='gmv-doc-code'
              code={
                props.models
                  ? source.replaceAll('gpt-image-2.6', model)
                  : source
              }
              language={language === 'curl' ? 'bash' : language}
              title={language === 'curl' ? 'cURL' : language}
              showToolbar
              enableCollapse={false}
            >
              <CodeBlockCopyButton />
            </CodeBlock>
          </TabsContent>
        ))}
      </Tabs>
      <p className='gmv-doc-code-caption'>
        {t('Run Python and JavaScript examples on your server.')}
      </p>
    </div>
  )
}
