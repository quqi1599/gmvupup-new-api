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

import { CopyButton } from '@/components/copy-button'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'

import {
  buildAgentPrompt,
  IMAGE_MODELS,
  VIDEO_MODELS,
  type SkillMode,
} from '../agent-prompts'

export function AgentSkills() {
  const { t } = useTranslation()
  const [mode, setMode] = useState<SkillMode>('all')
  const [imageModel, setImageModel] = useState(IMAGE_MODELS[0])
  const [videoModel, setVideoModel] = useState(VIDEO_MODELS[0])
  const [brief, setBrief] = useState('')
  const model = mode === 'image' ? imageModel : videoModel
  const prompt = buildAgentPrompt(t, mode, model, brief)
  return (
    <section id='skill-copy' className='gmv-doc-section gmv-agent-skills'>
      <h2>{t('Skill copy area')}</h2>
      <p className='gmv-doc-paragraph'>
        {t(
          'Choose a template, describe what you want, then copy it into your agent.'
        )}
      </p>
      <div className='gmv-skill-editor'>
        <Tabs
          value={mode}
          onValueChange={(value) => setMode(value as SkillMode)}
        >
          <TabsList aria-label={t('Skill template')} className='gmv-skill-tabs'>
            <TabsTrigger value='all'>{t('Universal skill')}</TabsTrigger>
            <TabsTrigger value='image'>{t('Generate an image')}</TabsTrigger>
            <TabsTrigger value='video'>{t('Generate a video')}</TabsTrigger>
          </TabsList>
          <TabsContent value={mode} className='gmv-skill-fields'>
            {mode !== 'all' && (
              <div>
                <Label htmlFor='skill-model'>{t('Generation model')}</Label>
                <NativeSelect
                  id='skill-model'
                  value={model}
                  onChange={(event) =>
                    mode === 'image'
                      ? setImageModel(event.target.value)
                      : setVideoModel(event.target.value)
                  }
                >
                  {(mode === 'image' ? IMAGE_MODELS : VIDEO_MODELS).map(
                    (id) => (
                      <NativeSelectOption key={id} value={id}>
                        {id}
                      </NativeSelectOption>
                    )
                  )}
                </NativeSelect>
              </div>
            )}
            <div>
              <Label htmlFor='skill-brief'>
                {t('Your creative brief (optional)')}
              </Label>
              <Textarea
                id='skill-brief'
                value={brief}
                onChange={(event) => setBrief(event.target.value)}
                placeholder={t(
                  'Describe your subject, style, aspect ratio and reference materials.'
                )}
                rows={3}
              />
            </div>
            <div className='gmv-skill-copy-row'>
              <CopyButton
                value={prompt}
                variant='default'
                size='default'
                aria-label={t('Copy prompt')}
              >
                {t('Copy prompt')}
              </CopyButton>
              <p>
                {t(
                  'Copying only prepares the prompt. Your agent makes the API request when you ask it to generate.'
                )}
              </p>
            </div>
            <details className='gmv-skill-preview'>
              <summary>{t('Preview the full prompt')}</summary>
              <p>
                {t(
                  'If copying is unavailable, select and copy the full prompt below.'
                )}
              </p>
              <Textarea
                readOnly
                value={prompt}
                aria-label={t('Full agent prompt')}
                rows={16}
              />
            </details>
          </TabsContent>
        </Tabs>
      </div>
    </section>
  )
}
