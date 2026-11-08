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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { beforeEach, afterEach, expect, it, vi } from 'vitest'

import en from '@/i18n/locales/en.json'
import { api } from '@/lib/api'
import { STATUS_QUERY_KEY } from '@/lib/status-query'
import { useAuthStore } from '@/stores/auth-store'

import { GmvupupModelCatalog } from '../index'

let client: QueryClient
const fixture = {
  success: true,
  data: [
    {
      id: 1,
      model_name: 'image-product',
      description: 'Product imagery',
      quota_type: 1,
      model_price: 0.1,
      model_ratio: 0,
      completion_ratio: 0,
      supported_endpoint_types: ['image-generation'],
      enable_groups: ['default'],
    },
    {
      id: 2,
      model_name: 'text-copy',
      description: 'Product copywriting',
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 2,
      supported_endpoint_types: ['openai'],
      enable_groups: ['default'],
    },
  ],
  vendors: [],
  group_ratio: { default: 1 },
  usable_group: { default: 'Default' },
  supported_endpoint: {},
  auto_groups: [],
}
beforeEach(async () => {
  await i18next.changeLanguage('en')
  i18next.addResourceBundle('en', 'translation', en.translation, true, true)
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  client.setQueryData(STATUS_QUERY_KEY, { price: 7.3, usd_exchange_rate: 7.3 })
  useAuthStore
    .getState()
    .auth.setUser({ id: 99, username: 'test', role: 1, group: 'default' })
  vi.spyOn(api, 'get').mockResolvedValue({ data: fixture })
})
afterEach(() => {
  client.clear()
  useAuthStore.getState().auth.reset()
  vi.restoreAllMocks()
})
function renderCatalog() {
  render(
    <QueryClientProvider client={client}>
      <GmvupupModelCatalog />
    </QueryClientProvider>
  )
}
it('filters models by type and search, then restores the list from the empty state', async () => {
  const user = userEvent.setup()
  renderCatalog()
  await screen.findByText('image-product')
  await user.selectOptions(
    screen.getByRole('combobox', { name: 'Model type' }),
    'image'
  )
  expect(screen.queryByText('text-copy')).not.toBeInTheDocument()
  await user.type(
    screen.getByRole('searchbox', { name: 'Search models' }),
    'missing'
  )
  expect(screen.getByText('No matching models')).toBeVisible()
  await user.click(screen.getByRole('button', { name: 'Clear filters' }))
  expect(screen.getByText('text-copy')).toBeVisible()
  expect(screen.getByText('image-product')).toBeVisible()
})
it('reports a failed catalog and retries instead of displaying fake prices', async () => {
  vi.mocked(api.get).mockRejectedValue(new Error('offline'))
  renderCatalog()
  expect(
    await screen.findByText('Could not load the model catalog')
  ).toBeVisible()
  vi.mocked(api.get).mockResolvedValue({ data: fixture })
  await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
  await waitFor(() => expect(screen.getByText('image-product')).toBeVisible())
})
