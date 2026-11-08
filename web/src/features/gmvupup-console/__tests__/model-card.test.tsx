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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { PricingModel } from '@/features/pricing/types'
import en from '@/i18n/locales/en.json'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { ConsoleModelCard } from '../model-card'

const image: PricingModel = {
  id: 1,
  model_name: 'gpt-image-2.6',
  quota_type: 1,
  model_ratio: 0,
  completion_ratio: 0,
  model_price: 0.02 / 7.3,
  enable_groups: ['default', 'vip', 'free'],
  group_ratio: { default: 1, vip: 2, free: 0 },
  supported_endpoint_types: ['image-generation'],
  description:
    'GPT Image 2.6\nCreate product photos.\nPOST /v1/images/generations',
}
const baseProps = { priceRate: 7.3, usdExchangeRate: 7.3, onDetails: vi.fn() }
beforeEach(async () => {
  i18next.addResourceBundle('en', 'translation', en.translation, true, true)
  await i18next.changeLanguage('en')
  useSystemConfigStore.getState().setConfig({
    currency: {
      ...DEFAULT_CURRENCY_CONFIG,
      quotaDisplayType: 'CNY',
      usdExchangeRate: 7.3,
    },
  })
})
afterEach(() => {
  vi.useRealTimers()
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
})
describe('simple model cards', () => {
  it.each([
    ['default', '¥0.02'],
    ['vip', '¥0.04'],
    ['free', '¥0'],
  ])('shows the real per-image charge for group %s', (group, price) => {
    render(
      <ConsoleModelCard {...baseProps} model={image} selectedGroup={group} />
    )
    const pricing = screen.getByRole('group', { name: 'Pricing' })
    expect(within(pricing).getByText(price)).toBeVisible()
    expect(pricing).toHaveTextContent('/ image')
    expect(screen.getByText('Create product photos.')).toBeVisible()
    expect(screen.queryByText(/POST \/v1/)).not.toBeInTheDocument()
  })
  it('does not present the cheapest other group when the selected group is unavailable', () => {
    render(
      <ConsoleModelCard
        {...baseProps}
        model={image}
        selectedGroup='unavailable'
      />
    )
    expect(screen.getByText('Not available in this group')).toBeVisible()
    expect(screen.queryByText('¥0.02')).not.toBeInTheDocument()
  })
  it('shows each configured video resolution price on the card', () => {
    const video: PricingModel = {
      ...image,
      model_name: 'video-example',
      quota_type: 0,
      supported_endpoint_types: ['openai_video'],
      billing_mode: 'tiered_expr',
      billing_expr:
        'u("resolution") == "480p" ? tier("480p", u("seconds") * 0.1) : u("resolution") == "720p" ? tier("720p", u("seconds") * 0.2) : tier("1080p", u("seconds") * 0.4)',
      billing_usage_schema: {
        seconds: {
          type: 'number',
          unit: 'second',
          description: { en: 'Video generation unit price' },
        },
        resolution: {
          enum: ['480p', '720p', '1080p'],
          description: { en: 'Resolution' },
        },
      },
    }
    render(
      <ConsoleModelCard {...baseProps} model={video} selectedGroup='default' />
    )
    for (const [resolution, price] of [
      ['480p', '¥0.73'],
      ['720p', '¥1.46'],
      ['1080p', '¥2.92'],
    ]) {
      const label = screen.getByText(`Resolution: ${resolution}`)
      expect(label.parentElement).toHaveTextContent(price)
      expect(label.parentElement).toHaveTextContent('/ s')
    }
  })
  it('shows current time pricing and cache price without exposing the formula', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-11-09T10:00:00+08:00'))
    const text: PricingModel = {
      ...image,
      quota_type: 0,
      supported_endpoint_types: ['openai'],
      billing_mode: 'tiered_expr',
      billing_expr:
        'hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12 ? tier("peak", p * 1 + c * 2 + cr * 0.1) : tier("off", p * 0.5 + c * 1 + cr * 0.05)',
    }
    render(
      <ConsoleModelCard {...baseProps} model={text} selectedGroup='default' />
    )
    expect(screen.getByText('¥7.3')).toBeVisible()
    expect(screen.getByText('¥14.6')).toBeVisible()
    expect(screen.getByText('¥0.73')).toBeVisible()
    expect(screen.getByText('Current period price')).toBeVisible()
    expect(screen.queryByText(/tier\(|hour\(/)).not.toBeInTheDocument()
  })
  it('copies the exact model ID instead of sending users to a separate detail page', async () => {
    const user = userEvent.setup()
    render(
      <ConsoleModelCard {...baseProps} model={image} selectedGroup='default' />
    )
    await user.click(screen.getByRole('button', { name: 'Copy model name' }))
    expect(await navigator.clipboard.readText()).toBe('gpt-image-2.6')
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
  it('shows an unconfigured price explicitly instead of a zero charge', () => {
    render(
      <ConsoleModelCard
        {...baseProps}
        model={{ ...image, model_price: undefined }}
      />
    )
    expect(screen.getByText('Price pending configuration')).toBeVisible()
    expect(screen.queryByText('¥0')).not.toBeInTheDocument()
  })
})
