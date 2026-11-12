import { Layers, Search, Sparkles } from 'lucide-react'
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
import { lazy, Suspense, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

import { ConsoleModelCard } from './model-card'
import { getModelCategory } from './model-category'

const ModelDetailsDrawer = lazy(() =>
  import('@/features/pricing/components/model-details').then((module) => ({
    default: module.ModelDetailsDrawer,
  }))
)

export function GmvupupModelCatalog() {
  const { t, i18n } = useTranslation()
  const pricing = usePricingData()
  const userGroup = useAuthStore((state) => state.auth.user?.group)
  const [search, setSearch] = useState('')
  const [category, setCategory] = useState('all')
  const [group, setGroup] = useState<string>()
  const [detail, setDetail] = useState<string | null>(null)
  const groups = Object.keys(pricing.usableGroup).filter(
    (key) => key !== 'auto'
  )
  const groupNames = new Map([
    ['default', t('Default group')],
    ['vip', t('VIP group')],
  ])
  const selectedGroup =
    group ?? (userGroup && groups.includes(userGroup) ? userGroup : groups[0])
  const models = useMemo(
    () =>
      pricing.models
        .filter((model) => {
          const haystack = [
            model.model_name,
            model.description,
            model.vendor_name,
          ]
            .join(' ')
            .toLocaleLowerCase()
          return (
            (category === 'all' || getModelCategory(model) === category) &&
            haystack.includes(search.trim().toLocaleLowerCase())
          )
        })
        .sort((a, b) => {
          const order = { image: 0, video: 1, text: 2, other: 3 }
          return (
            order[getModelCategory(a)] - order[getModelCategory(b)] ||
            (a.model_name ?? '').localeCompare(b.model_name ?? '')
          )
        }),
    [pricing.models, category, search]
  )
  const selectedModel =
    pricing.models.find((model) => model.model_name === detail) ?? null
  const locale = toIntlLocale(i18n.language)
  let currencyLabel = '—'
  if (!pricing.isLoading && !pricing.error) {
    currencyLabel =
      pricing.currencyLabel === 'CNY'
        ? t('Chinese yuan (CNY/RMB)')
        : (pricing.currencyLabel ?? '—')
  }
  let content
  if (pricing.isLoading) {
    content = <LoadingState />
  } else if (pricing.error) {
    content = (
      <ErrorState
        title={t('Could not load the model catalog')}
        action={
          <Button onClick={() => void pricing.refetch()}>{t('Retry')}</Button>
        }
      />
    )
  } else if (models.length === 0) {
    content = (
      <EmptyState
        title={t('No matching models')}
        description={t('Try another model type or search term.')}
        action={
          <Button
            variant='outline'
            onClick={() => {
              setSearch('')
              setCategory('all')
            }}
          >
            {t('Clear filters')}
          </Button>
        }
      />
    )
  } else {
    content = (
      <div className='gmvc-model-grid'>
        {models.map((model) => (
          <ConsoleModelCard
            key={model.model_name}
            model={model}
            selectedGroup={selectedGroup}
            priceRate={pricing.priceRate}
            usdExchangeRate={pricing.usdExchangeRate}
            onDetails={setDetail}
          />
        ))}
      </div>
    )
  }

  return (
    <main className='gmvc-catalog'>
      <title>{t('Model list')} | GMVUPUP</title>
      <section className='gmvc-intro'>
        <div>
          <span className='gmvc-eyebrow'>
            <Sparkles size={14} />
            {t('AI model services')}
          </span>
          <h1>{t('Model catalog')}</h1>
          <p>
            {t(
              'See the model, the price and what it does. Everything you need to choose.'
            )}
          </p>
        </div>
        <div className='gmvc-intro-stats'>
          <div>
            <Layers size={19} />
            <strong>
              {pricing.isLoading
                ? '—'
                : formatNumber(pricing.models.length, locale)}
            </strong>
            <span>{t('Available models')}</span>
          </div>
          <div>
            <span>{t('Console currency')}</span>
            <strong>{currencyLabel}</strong>
            <span>{t('Prices from this platform')}</span>
          </div>
        </div>
      </section>
      <section className='gmvc-toolbar' aria-label={t('Filter models')}>
        <div className='gmvc-catalog-brand'>
          <Layers size={19} />
          <strong>GmvUpUp</strong>
        </div>
        <div className='gmvc-filters'>
          {groups.length > 1 && (
            <NativeSelect
              aria-label={t('Pricing group')}
              value={selectedGroup ?? ''}
              onChange={(event) => setGroup(event.target.value)}
            >
              {groups.map((key) => (
                <NativeSelectOption key={key} value={key}>
                  {groupNames.get(key) ?? key}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          )}
          <NativeSelect
            aria-label={t('Model type')}
            value={category}
            onChange={(event) => setCategory(event.target.value)}
          >
            <NativeSelectOption value='all'>
              {t('All models')}
            </NativeSelectOption>
            <NativeSelectOption value='image'>
              {t('Image models')}
            </NativeSelectOption>
            <NativeSelectOption value='video'>
              {t('Video models')}
            </NativeSelectOption>
            <NativeSelectOption value='text'>
              {t('Text models')}
            </NativeSelectOption>
            <NativeSelectOption value='other'>
              {t('Other models')}
            </NativeSelectOption>
          </NativeSelect>
          <div className='gmvc-search'>
            <Search size={16} aria-hidden='true' />
            <Input
              type='search'
              aria-label={t('Search models')}
              placeholder={t('Search model name or description')}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
          </div>
        </div>
      </section>
      <div className='gmvc-results-heading'>
        <div>
          <h2>{t('Model list')}</h2>
          <span>
            {t('{{count}} models displayed', { count: models.length })}
          </span>
        </div>
        <p>
          {!pricing.isLoading &&
            !pricing.error &&
            t('All prices are in {{currency}}.', { currency: currencyLabel })}
        </p>
      </div>
      {content}
      {selectedModel && (
        <Suspense fallback={<LoadingState />}>
          <ModelDetailsDrawer
            model={selectedModel}
            open={Boolean(selectedModel)}
            onOpenChange={(open) => {
              if (!open) setDetail(null)
            }}
            groupRatio={pricing.groupRatio}
            usableGroup={pricing.usableGroup}
            endpointMap={
              pricing.endpointMap as unknown as Record<
                string,
                { path?: string; method?: string }
              >
            }
            autoGroups={pricing.autoGroups}
            priceRate={pricing.priceRate}
            usdExchangeRate={pricing.usdExchangeRate}
            tokenUnit='M'
            showRechargePrice={false}
          />
        </Suspense>
      )}
      <footer className='gmvc-catalog-footer'>
        GMVUPUP · {t('Prices update with the current model configuration.')}
      </footer>
    </main>
  )
}
