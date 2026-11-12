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
import { isNativeSeedanceModel } from '@/features/gmvupup/customer-models'
import { isTokenBasedModel } from '@/features/pricing/lib/model-helpers'
import type { PricingModel } from '@/features/pricing/types'

export function getModelCategory(model: PricingModel) {
  const endpoints = model.supported_endpoint_types ?? []
  if (
    isNativeSeedanceModel(model) ||
    endpoints.some((type) => /video|sora/.test(type)) ||
    Object.values(model.billing_usage_schema ?? {}).some(
      (field) => field.unit === 'second'
    )
  ) {
    return 'video'
  }
  if (
    endpoints.some(
      (type) => type === 'image-generation' || type === 'image-edits'
    )
  ) {
    return 'image'
  }
  if (isTokenBasedModel(model)) return 'text'
  return 'other'
}
