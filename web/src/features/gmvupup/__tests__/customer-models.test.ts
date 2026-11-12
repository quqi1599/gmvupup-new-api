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
import { describe, expect, it } from 'vitest'

import type { PricingModel } from '@/features/pricing/types'

import {
  getCustomerModelName,
  selectCustomerModels,
  isNativeSeedanceModel,
} from '../customer-models'

const legacy: PricingModel = {
  id: 1,
  model_name: 'doubao-seedance-2.5',
  quota_type: 0,
  model_ratio: 0,
  completion_ratio: 0,
  enable_groups: ['default'],
}
const native: PricingModel = {
  ...legacy,
  id: 2,
  model_name: 'doubao-seedance-2-5-260628',
}

describe('customer Seedance product selection', () => {
  it('keeps a public short ID on the native protocol when that ID is bound to Doubao', () => {
    const publicNative: PricingModel = {
      ...legacy,
      billing_plugin_variants: [
        {
          plugin_key: 'doubao',
          plugin_name: 'Doubao',
          billing_expr: 'tier("base", u("tokens") * 0.000001)',
          billing_usage_schema: { tokens: { type: 'number', unit: 'token' } },
        },
      ],
    }
    expect(isNativeSeedanceModel(publicNative)).toBe(true)
    expect(selectCustomerModels([native, publicNative], 'default')).toEqual([
      publicNative,
    ])
  })
  it('prefers the native API record with its own prices, independent of response order', () => {
    const prices = [
      { ...legacy, billing_expr: 'seconds-price' },
      { ...native, billing_expr: 'token-price' },
    ]
    expect(selectCustomerModels(prices, 'default')).toEqual([prices[1]])
    expect(selectCustomerModels([...prices].reverse(), 'default')[0]).toBe(
      prices[1]
    )
    expect(prices).toHaveLength(2)
  })
  it('keeps a callable legacy entry when native is unavailable in the selected group', () => {
    expect(
      selectCustomerModels(
        [legacy, { ...native, enable_groups: ['vip'] }],
        'default'
      )
    ).toEqual([legacy])
    expect(selectCustomerModels([legacy], 'default')).toEqual([legacy])
  })
  it('does not conflate different versions, fast or mini with a base model', () => {
    const v20 = { ...native, model_name: 'doubao-seedance-2-0-260128' }
    const fast = { ...native, model_name: 'doubao-seedance-2-0-fast-260128' }
    const mini = { ...native, model_name: 'doubao-seedance-2-0-mini-260615' }
    expect(selectCustomerModels([legacy, native, v20, fast, mini])).toEqual([
      native,
      v20,
      fast,
      mini,
    ])
    expect([native, v20, fast].map(getCustomerModelName)).toEqual([
      'Seedance 2.5',
      'Seedance 2.0',
      fast.model_name,
    ])
    expect(selectCustomerModels([])).toEqual([])
  })
})
