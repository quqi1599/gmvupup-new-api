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
import type { PricingModel } from '@/features/pricing/types'

// These IDs are alternate integrations of the same customer product. Keep the
// selected record intact: its callable ID, prices and protocol belong together.
const seedanceProducts = [
  {
    name: 'Seedance 2.5',
    publicId: 'doubao-seedance-2.5',
    ids: [
      'doubao-seedance-2-5-260628',
      'doubao-seedance-2-5',
      'doubao-seedance-2.5',
    ],
  },
  {
    name: 'Seedance 2.0',
    publicId: 'doubao-seedance-2.0',
    ids: [
      'doubao-seedance-2-0-260128',
      'doubao-seedance-2-0',
      'doubao-seedance-2.0',
    ],
  },
]

export function getCustomerModelName(model: PricingModel): string {
  return (
    seedanceProducts.find((product) => product.ids.includes(model.model_name))
      ?.name ?? model.model_name
  )
}

export function getCustomerModelSearchNames(model: PricingModel): string {
  const product = seedanceProducts.find((entry) =>
    entry.ids.includes(model.model_name)
  )
  return product ? [product.name, ...product.ids].join(' ') : model.model_name
}

export function selectCustomerModels(
  models: PricingModel[],
  group?: string
): PricingModel[] {
  const selected = new Map<string, PricingModel>()
  for (const product of seedanceProducts) {
    const candidates = product.ids.flatMap((id) =>
      models.filter((model) => model.model_name === id)
    )
    const publicNative = candidates.find(
      (model) =>
        model.model_name === product.publicId &&
        isNativeSeedanceModel(model) &&
        (!group ||
          !model.enable_groups?.length ||
          model.enable_groups.includes(group))
    )
    const available =
      publicNative ??
      candidates.find(
        (model) =>
          !group ||
          !model.enable_groups?.length ||
          model.enable_groups.includes(group)
      )
    if (available || candidates[0]) {
      selected.set(product.name, available ?? candidates[0])
    }
  }
  return models.filter((model) => {
    const chosen = selected.get(getCustomerModelName(model))
    return !chosen || chosen === model
  })
}

export function isNativeSeedanceModel(model: PricingModel): boolean {
  return (
    seedanceProducts.some((product) =>
      product.ids.includes(model.model_name)
    ) &&
    model.billing_plugin_variants?.length === 1 &&
    model.billing_plugin_variants[0].plugin_key === 'doubao'
  )
}

export function getCustomerModelDescriptionKey(
  model: PricingModel
): string | undefined {
  if (!isNativeSeedanceModel(model)) return undefined
  if (getCustomerModelName(model) === 'Seedance 2.5') {
    return 'Create videos from text and images, or edit a reference video. Pricing depends on resolution and reference video input.'
  }
  return 'Create videos from text, images and reference videos. Pricing depends on resolution and reference video input.'
}
