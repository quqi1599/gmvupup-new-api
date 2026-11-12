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
import { Image, Video, MessageSquare, Box } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { GmvupupModelLogo } from '@/features/gmvupup/components/model-logo'
import {
  getCustomerModelName,
  getCustomerModelDescriptionKey,
} from '@/features/gmvupup/customer-models'
import { useBillingTime } from '@/features/pricing/hooks/use-billing-time'
import type { ParsedTaskTier } from '@/features/pricing/lib/billing-expr'
import { formatBillingCondition } from '@/features/pricing/lib/billing-expression/condition-display'
import {
  getDynamicPricingSummary,
  getDynamicPriceEntries,
  getDynamicDisplayGroupRatio,
  getDynamicPriceUnitLabelKey,
  isUnconfiguredTaskUsageModel,
} from '@/features/pricing/lib/dynamic-price'
import {
  getFixedPriceUnit,
  isTokenBasedModel,
} from '@/features/pricing/lib/model-helpers'
import { formatPrice, formatRequestPrice } from '@/features/pricing/lib/price'
import {
  taskPriceLabel,
  taskUsageUnitLabel,
  taskPricingConditions,
} from '@/features/pricing/lib/task-price-display'
import type { PricingModel } from '@/features/pricing/types'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { getModelCategory } from './model-category'

export function ConsoleModelCard(props: {
  model: PricingModel
  selectedGroup?: string
  priceRate: number
  usdExchangeRate: number
  onDetails: (name: string) => void
}) {
  const { t, i18n } = useTranslation()
  useSystemConfigStore((state) => state.config.currency)
  const now = useBillingTime(props.model.billing_expr)
  const options = {
    tokenUnit: 'M' as const,
    priceRate: props.priceRate,
    usdExchangeRate: props.usdExchangeRate,
    groupRatioMultiplier: getDynamicDisplayGroupRatio(
      props.model,
      props.selectedGroup
    ),
    usageSchema: props.model.billing_usage_schema,
    now: now === undefined ? undefined : new Date(now),
  }
  const summary = getDynamicPricingSummary(props.model, options)
  const category = getModelCategory(props.model)
  const Icon = { image: Image, video: Video, text: MessageSquare, other: Box }[
    category
  ]
  const typeLabel = {
    image: t('Image model'),
    video: t('Video model'),
    text: t('Text model'),
    other: t('AI model'),
  }[category]
  const available =
    !props.selectedGroup ||
    !props.model.enable_groups?.length ||
    props.model.enable_groups.includes(props.selectedGroup)
  const unpriced =
    isUnconfiguredTaskUsageModel(props.model) ||
    (!summary &&
      !isTokenBasedModel(props.model) &&
      typeof props.model.model_price !== 'number')
  const fixedUnit = getFixedPriceUnit(props.model)
  const unit = fixedUnit === 'image' ? t('image') : t('request')
  let rows: { key: string; label: string; price: string; unit: string }[] = []
  if (available && !unpriced && summary && !summary.isSpecialExpression) {
    if (
      summary.isTaskUsage &&
      !summary.providerCount &&
      summary.tiers.length > 0
    ) {
      rows = summary.tiers.flatMap((tier, index) => {
        const usageCondition =
          'unitPrices' in tier
            ? taskPricingConditions(
                (tier as ParsedTaskTier).conditions,
                props.model.billing_usage_schema,
                i18n.language,
                t
              )
            : ''
        const timeCondition = tier.conditionText
          ? (formatBillingCondition(tier.conditionText, t, i18n.language) ??
            t('Conditional rate'))
          : ''
        const condition = [usageCondition, timeCondition]
          .filter(Boolean)
          .join(' · ')
        return getDynamicPriceEntries(tier, options).map((entry) => ({
          key: `${index}-${entry.key}`,
          label:
            condition ||
            taskPriceLabel(entry.description, entry.shortLabel, i18n.language),
          price: entry.formatted,
          unit: taskUsageUnitLabel(
            entry,
            i18n.language,
            t(getDynamicPriceUnitLabelKey(entry) ?? 'million tokens')
          ),
        }))
      })
    } else {
      rows = summary.entries.map((entry) => ({
        key: entry.key,
        label:
          entry.labelKind === 'schema'
            ? taskPriceLabel(entry.description, entry.shortLabel, i18n.language)
            : t(entry.shortLabel),
        price: entry.formattedRange ?? entry.formatted,
        unit: taskUsageUnitLabel(
          entry,
          i18n.language,
          t(getDynamicPriceUnitLabelKey(entry) ?? 'million tokens')
        ),
      }))
    }
  } else if (available && !unpriced && !summary) {
    if (isTokenBasedModel(props.model)) {
      const types = [
        'input',
        'output',
        ...(props.model.cache_ratio != null ? (['cache'] as const) : []),
      ] as const
      rows = types.map((type) => ({
        key: type,
        label: t({ input: 'Input', output: 'Output', cache: 'Cached' }[type]),
        price: formatPrice(
          props.model,
          type,
          'M',
          false,
          props.priceRate,
          props.usdExchangeRate,
          props.selectedGroup
        ),
        unit: t('million tokens'),
      }))
    } else {
      rows = [
        {
          key: 'fixed',
          label:
            fixedUnit === 'image'
              ? t('Price per image')
              : t('Price per request'),
          price: formatRequestPrice(
            props.model,
            false,
            props.priceRate,
            props.usdExchangeRate,
            props.selectedGroup
          ),
          unit,
        },
      ]
    }
  }
  const descriptionKey = getCustomerModelDescriptionKey(props.model)
  const lines = (props.model.description ?? '')
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
  const description =
    lines.length > 1 && !/[。！？!?]|\.\s/.test(lines[0]) ? lines[1] : lines[0]
  let unavailableLabel = t('Price pending configuration')
  if (!available) {
    unavailableLabel = t('Not available in this group')
  } else if (summary?.isSpecialExpression) {
    unavailableLabel = t('Request-dependent pricing')
  }
  return (
    <Card className={`gmvc-model gmvc-model-${category}`}>
      <header>
        <span className='gmvc-model-icon'>
          <GmvupupModelLogo
            modelName={props.model.model_name}
            icon={props.model.icon || props.model.vendor_icon}
            size={28}
            fallback={<Icon size={25} aria-hidden='true' />}
          />
        </span>
        <div>
          <h2>{getCustomerModelName(props.model)}</h2>
          <div className='gmvc-model-tags'>
            <span>{typeLabel}</span>
            <span>
              {category === 'image' && fixedUnit === 'image'
                ? t('Billed per image')
                : t('Metered billing')}
            </span>
          </div>
        </div>
        <CopyButton
          value={props.model.model_name ?? ''}
          tooltip={t('Copy model name')}
        />
      </header>
      <div className='gmvc-model-prices' role='group' aria-label={t('Pricing')}>
        <p>
          {t(
            {
              video: 'Video pricing',
              image: 'Image pricing',
              text: 'Model pricing',
              other: 'Model pricing',
            }[category]
          )}
          <Icon size={15} aria-hidden='true' />
        </p>
        {rows.length > 0 ? (
          <dl
            className={
              rows.length === 1 ? 'gmvc-prices-single' : 'gmvc-prices-grid'
            }
          >
            {rows.map((row) => (
              <div key={row.key}>
                <dt>{row.label}</dt>
                <dd>
                  <strong>{row.price}</strong>
                  <span> / {row.unit}</span>
                </dd>
              </div>
            ))}
          </dl>
        ) : (
          <p className='gmvc-price-unavailable'>{unavailableLabel}</p>
        )}
        {summary?.isTimePricing && <small>{t('Current period price')}</small>}
        {summary?.hasRequestRules && (
          <small>{t('Additional request conditions apply')}</small>
        )}
        {summary?.isSpecialExpression && available && (
          <Button
            variant='link'
            onClick={() => props.onDetails(props.model.model_name ?? '')}
          >
            {t('View pricing details')}
          </Button>
        )}
      </div>
      <p className='gmvc-model-description'>
        {descriptionKey
          ? t(descriptionKey)
          : description || t('No description available.')}
      </p>
    </Card>
  )
}
