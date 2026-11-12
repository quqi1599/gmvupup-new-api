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
import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { beforeEach, afterEach, expect, it, vi } from 'vitest'

import en from '@/i18n/locales/en.json'
import { api } from '@/lib/api'
import { STATUS_QUERY_KEY } from '@/lib/status-query'
import { useAuthStore } from '@/stores/auth-store'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

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
  await i18next.init({
    lng: 'en',
    fallbackLng: 'en',
    resources: { en: { translation: en.translation } },
    interpolation: { escapeValue: false },
  })
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  client.setQueryData(STATUS_QUERY_KEY, {
    price: 7.3,
    usd_exchange_rate: 7.3,
    quota_display_type: 'CNY',
  })
  useSystemConfigStore.getState().setConfig({
    currency: {
      ...DEFAULT_CURRENCY_CONFIG,
      quotaDisplayType: 'CNY',
      usdExchangeRate: 7.3,
    },
  })
  useAuthStore
    .getState()
    .auth.setUser({ id: 99, username: 'test', role: 1, group: 'default' })
  vi.spyOn(api, 'get').mockResolvedValue({ data: fixture })
})
afterEach(() => {
  client.clear()
  useAuthStore.getState().auth.reset()
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
  localStorage.removeItem('status')
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

it('states RMB and converts image and text prices with the site rate', async () => {
  renderCatalog()
  await screen.findByText('image-product')
  expect(screen.getByText('Chinese yuan (CNY/RMB)')).toBeVisible()
  expect(
    screen.getByText(/All prices are in Chinese yuan \(CNY\/RMB\)\./)
  ).toBeVisible()
  expect(screen.getByText('¥0.73')).toBeVisible()
  expect(screen.getByText('¥14.6')).toBeVisible()
  expect(screen.getByText('¥29.2')).toBeVisible()
  expect(document.body).not.toHaveTextContent('$')
})

it('waits for current currency settings even when old USD status is cached', async () => {
  client.removeQueries({ queryKey: STATUS_QUERY_KEY })
  localStorage.setItem(
    'status',
    JSON.stringify({
      price: 1,
      usd_exchange_rate: 1,
      quota_display_type: 'USD',
    })
  )
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
  let resolveStatus!: (value: {
    data: {
      success: boolean
      data: {
        price: number
        usd_exchange_rate: number
        quota_display_type: string
      }
    }
  }) => void
  const status = new Promise<Parameters<typeof resolveStatus>[0]>((resolve) => {
    resolveStatus = resolve
  })
  vi.mocked(api.get).mockImplementation(async (url) => {
    if (url === '/api/status') return await status
    return { data: fixture }
  })
  renderCatalog()
  await waitFor(() => expect(client.getQueryData(['pricing'])).toBeDefined())
  expect(screen.queryByText('image-product')).not.toBeInTheDocument()
  expect(document.body).not.toHaveTextContent('USD')
  expect(document.body).not.toHaveTextContent('$')
  await act(async () =>
    resolveStatus({
      data: {
        success: true,
        data: { price: 7.3, usd_exchange_rate: 7.3, quota_display_type: 'CNY' },
      },
    })
  )
  expect(await screen.findByText('¥0.73')).toBeVisible()
  expect(screen.getByText('Chinese yuan (CNY/RMB)')).toBeVisible()
})

it('offers retry when currency loading fails instead of showing USD prices', async () => {
  client.removeQueries({ queryKey: STATUS_QUERY_KEY })
  vi.mocked(api.get).mockImplementation(async (url) => {
    if (url === '/api/status') throw new Error('status offline')
    return { data: fixture }
  })
  renderCatalog()
  expect(
    await screen.findByText('Could not load the model catalog')
  ).toBeVisible()
  expect(screen.queryByText('image-product')).not.toBeInTheDocument()
  vi.mocked(api.get).mockImplementation(async (url) => ({
    data:
      url === '/api/status'
        ? {
            success: true,
            data: {
              price: 7.3,
              usd_exchange_rate: 7.3,
              quota_display_type: 'CNY',
            },
          }
        : fixture,
  }))
  await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
  expect(await screen.findByText('¥0.73')).toBeVisible()
})
