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
import {
  getCustomerModelName,
  isNativeSeedanceModel,
} from '@/features/gmvupup/customer-models'
import {
  resolveLocalizedText,
  type LocalizedTextValue,
} from '@/lib/localized-text'

import type {
  BillingUsageFieldSchema,
  BillingUsageSchema,
  PricingModel,
} from '../types'
import {
  splitBillingExprAndRequestRules,
  type ParsedTaskTier,
  type TaskTierCondition,
} from './billing-expr'
import { formatBillingCondition } from './billing-expression/condition-display'
import { compileBillingExpression } from './billing-expression/parser'
import { evaluateBillingExpression } from './billing-expression/runtime'
import { visitExpression } from './billing-expression/types'
import {
  getDynamicPricingSummary,
  getDynamicPriceEntries,
  formatTaskUsageUnitPrice,
  type DynamicPriceOptions,
} from './dynamic-price'
import { withPluginPricing } from './plugin-pricing'
import { getTaskPricingDisplayTiers } from './task-matrix-display'

export type VideoSecondPricing = {
  rows: { resolution: string; price: string; referencePrice?: string }[]
  estimated?: boolean
  extraImagePrice?: string
}

/** Display estimates only; the original expression and settlement units stay intact. */
export function getVideoSecondPricing(
  model: PricingModel,
  options: DynamicPriceOptions
): VideoSecondPricing | null {
  const variants = model.billing_plugin_variants
  const priced =
    variants?.length === 1 ? withPluginPricing(model, variants[0]) : model
  if (isNativeSeedanceModel(model)) {
    const schema = priced.billing_usage_schema
    const expression = splitBillingExprAndRequestRules(
      priced.billing_expr || ''
    )
    if (expression.requestRuleExpr || schema?.tokens?.unit !== 'token') {
      return null
    }
    const tiers = getTaskPricingDisplayTiers(expression.billingExpr, schema)
    const resolutions = schema.resolution?.enum ?? []
    if (
      !tiers ||
      tiers.length !== resolutions.length * 2 ||
      !resolutions.length
    ) {
      return null
    }
    // Official 16:9 output sizes and 24 fps. 480p differs between 2.0 and 2.5.
    // https://docs.volcengine.com/docs/ark/video-generation-tutorial?lang=zh
    // https://docs.volcengine.com/docs/ark/model-pricing?lang=zh
    const pixels: Record<string, number> = {
      '480p':
        getCustomerModelName(model) === 'Seedance 2.0' ? 864 * 496 : 854 * 480,
      '720p': 1280 * 720,
      '1080p': 1920 * 1080,
      '4k': 3840 * 2160,
    }
    const rows: VideoSecondPricing['rows'] = []
    for (const resolution of resolutions) {
      if (!pixels[resolution]) return null
      const prices: string[] = []
      for (const input of ['none', 'video']) {
        const matching = tiers.filter(
          (tier) =>
            tier.conditions.length === 2 &&
            tier.conditions.every(
              (condition) =>
                (condition.field === 'resolution' &&
                  condition.value === resolution) ||
                (condition.field === 'video_input' && condition.value === input)
            )
        )
        const tier = matching[0]
        if (
          matching.length !== 1 ||
          tier.constant !== 0 ||
          tier.conditionText ||
          Object.keys(tier.unitPrices).some((key) => key !== 'tokens')
        ) {
          return null
        }
        const rate = tier.unitPrices.tokens
        if (!Number.isFinite(rate) || rate < 0) return null
        prices.push(
          formatTaskUsageUnitPrice(
            (rate * pixels[resolution] * 24) / 1024 / 1_000_000,
            options
          )
        )
      }
      rows.push({ resolution, price: prices[0], referencePrice: prices[1] })
    }
    return { rows, estimated: true }
  }
  if (
    model.model_name === 'MiniMax-H3' &&
    variants?.length === 1 &&
    variants[0].plugin_key === 'hailuo'
  ) {
    const compiled = compileBillingExpression(priced.billing_expr || '')
    if (
      compiled.status !== 'ready' ||
      [...compiled.functions].some(
        (name) => !['tier', 'u', 'max'].includes(name)
      )
    ) {
      return null
    }
    const rows: VideoSecondPricing['rows'] = []
    let extraImagePrice: string | undefined
    for (const resolution of priced.billing_usage_schema?.resolution?.enum ??
      []) {
      const costs = [
        { seconds: 1, input_video_seconds: 0, input_images: 0 },
        { seconds: 2, input_video_seconds: 0, input_images: 5 },
        { seconds: 0, input_video_seconds: 1, input_images: 0 },
        { seconds: 1, input_video_seconds: 0, input_images: 6 },
      ].map((facts) =>
        evaluateBillingExpression(compiled, { usage: { ...facts, resolution } })
      )
      if (costs.some((cost) => cost.status !== 'success')) return null
      const values = costs.map((cost) =>
        cost.status === 'success' ? cost.cost : Number.NaN
      )
      if (
        Math.abs(values[1] - values[0] * 2) > 1e-10 ||
        Math.abs(values[2] - values[0]) > 1e-10 ||
        values[3] < values[0]
      ) {
        return null
      }
      const imagePrice = formatTaskUsageUnitPrice(
        values[3] - values[0],
        options
      )
      if (extraImagePrice !== undefined && extraImagePrice !== imagePrice) {
        return null
      }
      extraImagePrice = imagePrice
      rows.push({
        resolution,
        price: formatTaskUsageUnitPrice(values[0], options),
      })
    }
    return rows.length ? { rows, extraImagePrice } : null
  }
  const rows = getVideoResolutionPrices(model, options)
  return rows ? { rows } : null
}

export function taskPriceLabel(
  description: LocalizedTextValue | undefined,
  field: string,
  language: string
): string {
  const localized =
    typeof description === 'object' && description
      ? { ...description, en: description.en?.trim() || field }
      : description
  return resolveLocalizedText(localized, language) || field
}

export function taskEnumLabel(
  definition: BillingUsageFieldSchema | undefined,
  value: string,
  language: string
): string {
  return taskPriceLabel(definition?.enumLabels?.[value], value, language)
}

export function taskUsageUnitLabel(
  definition: { unit?: string; unitLabel?: LocalizedTextValue } | undefined,
  language: string,
  fallback: string
): string {
  if (definition?.unit !== 'count') return fallback
  return resolveLocalizedText(definition.unitLabel, language) || fallback
}

export function taskPricingConditions(
  conditions: TaskTierCondition[],
  schema: BillingUsageSchema | undefined,
  language: string,
  t: (key: string) => string
): string {
  return conditions
    .map(({ field, value }) => {
      const definition = schema?.[field]
      const label = taskPriceLabel(definition?.description, field, language)
      if (definition?.type === 'boolean') {
        return `${label}: ${value === 'true' ? t('Yes') : t('No')}`
      }
      const optionLabel = taskEnumLabel(definition, value, language)
      return optionLabel !== value ? optionLabel : `${label}: ${optionLabel}`
    })
    .join(' · ')
}

export function hasSimpleTaskPricing(model: PricingModel): boolean {
  if (
    !model.billing_usage_schema ||
    model.billing_mode !== 'tiered_expr' ||
    !model.billing_expr
  ) {
    return false
  }
  const split = splitBillingExprAndRequestRules(model.billing_expr)
  if (split.requestRuleExpr?.trim()) return false
  const tiers = getTaskPricingDisplayTiers(
    split.billingExpr,
    model.billing_usage_schema
  )
  return tiers.length === 1 && !tiers[0].conditionText
}

export function taskTierConditions(
  tier: ParsedTaskTier,
  schema: BillingUsageSchema | undefined,
  language: string,
  t: (key: string) => string
): string {
  const usage = taskPricingConditions(tier.conditions, schema, language, t)
  const time = tier.conditionText
    ? (formatBillingCondition(tier.conditionText, t, language) ??
      tier.conditionText)
    : ''
  return [usage, time].filter(Boolean).join(' · ')
}

export function pricingDisplayFallbackKey(
  expression: string,
  schema: BillingUsageSchema | null | undefined
): string {
  const compiled = compileBillingExpression(expression)
  let missingUsageMetadata = false
  if (compiled.status === 'ready') {
    visitExpression(compiled.ast, (node) => {
      if (node.kind !== 'call' || node.name !== 'u') return
      const key = node.args[0]
      if (
        !schema ||
        key.kind !== 'literal' ||
        typeof key.value !== 'string' ||
        !Object.hasOwn(schema, key.value)
      ) {
        missingUsageMetadata = true
      }
    })
  }
  return missingUsageMetadata
    ? 'Task usage metadata is unavailable. Pricing details cannot be displayed.'
    : 'This expression cannot be expanded into a price table. View the original expression below.'
}

/** Expand only a bounded, single-provider resolution matrix, never mixed rules. */
export function getVideoResolutionPrices(
  model: PricingModel,
  options: DynamicPriceOptions
): { resolution: string; price: string }[] | null {
  const summary = getDynamicPricingSummary(model, options)
  if (
    !summary?.isTaskUsage ||
    summary.providerCount ||
    summary.hasRequestRules ||
    summary.isSpecialExpression ||
    summary.tiers.length < 2 ||
    summary.tiers.length > 6
  ) {
    return null
  }
  const rows = []
  for (const tier of summary.tiers) {
    if (
      !('unitPrices' in tier) ||
      tier.constant !== 0 ||
      tier.conditionText ||
      tier.conditions.length !== 1
    ) {
      return null
    }
    const condition = tier.conditions[0]
    if (
      !('field' in condition) ||
      typeof condition.value !== 'string' ||
      !['quality', 'resolution'].includes(condition.field) ||
      !/^(480p|720p|1080p|4k)$/i.test(condition.value)
    ) {
      return null
    }
    const entries = getDynamicPriceEntries(tier, {
      ...options,
      usageSchema:
        model.billing_plugin_variants?.[0]?.billing_usage_schema ??
        model.billing_usage_schema,
    })
    if (entries.length !== 1 || entries[0].unit !== 'second') return null
    rows.push({ resolution: condition.value, price: entries[0].formatted })
  }
  if (new Set(rows.map((row) => row.resolution)).size !== rows.length) {
    return null
  }
  return rows
}
