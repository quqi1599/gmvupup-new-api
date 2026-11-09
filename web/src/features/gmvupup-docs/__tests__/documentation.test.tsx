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
import { readFileSync } from 'node:fs'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { assert, beforeEach, describe, expect, it, vi } from 'vitest'

import type { PricingModel } from '@/features/pricing/types'
import en from '@/i18n/locales/en.json'
import { api } from '@/lib/api'
import { STATUS_QUERY_KEY } from '@/lib/status-query'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { CodeExamples } from '../components/code-examples'
import { API_BASE_URL, DOC_PAGES, isDocTopic } from '../content'
import { GmvupupDocs } from '../index'

async function renderDocs(
  topic = 'quick-start',
  cachedCatalog = true,
  models: PricingModel[] = []
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  client.setQueryData(STATUS_QUERY_KEY, {})
  if (cachedCatalog) {
    client.setQueryData(['pricing'], {
      success: true,
      data: models,
      vendors: [],
      group_ratio: { default: 1 },
      usable_group: {},
      supported_endpoint: {},
      auto_groups: [],
    })
  }
  const root = createRootRoute({ component: Outlet })
  const route = createRoute({
    getParentRoute: () => root,
    path: '/docs/$topic',
    component: () => {
      const params = route.useParams()
      return (
        <GmvupupDocs
          topic={isDocTopic(params.topic) ? params.topic : 'quick-start'}
        />
      )
    },
  })
  const router = createRouter({
    routeTree: root.addChildren([route]),
    history: createMemoryHistory({ initialEntries: [`/docs/${topic}`] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByRole('heading', { level: 1 })
  return router
}

beforeEach(async () => {
  localStorage.clear()
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
  i18next.addResourceBundle('en', 'translation', en.translation, true, true)
  await i18next.changeLanguage('en')
  vi.spyOn(window, 'scrollTo').mockImplementation(() => undefined)
})

describe('GMVUPUP documentation', () => {
  it('copies the SDK base URL with one v1 prefix', async () => {
    const user = userEvent.setup()
    await renderDocs()
    await user.click(screen.getByRole('button', { name: 'Copy Base URL' }))
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Copied' })).toBeVisible()
    )
    expect(await navigator.clipboard.readText()).toBe('https://gmvupup.top/v1')
  })
  it('filters documentation and opens the matching topic with Enter', async () => {
    const user = userEvent.setup()
    const router = await renderDocs()
    const search = screen.getByRole('searchbox', {
      name: 'Search documentation',
    })
    await user.type(search, 'multipart')
    const nav = screen.getByRole('navigation', {
      name: 'Documentation navigation',
    })
    expect(
      within(nav).getByRole('link', { name: /Image editing/ })
    ).toBeVisible()
    await user.keyboard('{Enter}')
    await waitFor(() =>
      expect(router.state.location.pathname).toBe('/docs/image-edits')
    )
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
      'Image editing'
    )
  })
  it('shows a clear empty search and restores navigation when cleared', async () => {
    const user = userEvent.setup()
    await renderDocs()
    await user.type(
      screen.getByRole('searchbox', { name: 'Search documentation' }),
      'not-a-real-endpoint'
    )
    expect(screen.getByRole('status')).toHaveTextContent(
      'No matching documents'
    )
    await user.click(screen.getByRole('button', { name: 'Clear search' }))
    expect(screen.queryByText('No matching documents')).not.toBeInTheDocument()
    expect(screen.getByRole('searchbox')).toHaveValue('')
  })
  it('copies the selected image model in the selected code language', async () => {
    const user = userEvent.setup()
    const imageExamples = DOC_PAGES.images.examples
    assert(imageExamples)
    render(
      <CodeExamples examples={imageExamples} models={DOC_PAGES.images.models} />
    )
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Example model' }),
      'nanobanana-2.1'
    )
    await user.click(screen.getByRole('tab', { name: 'Python' }))
    const panel = screen.getByRole('tabpanel', { name: 'Python' })
    await user.click(within(panel).getByRole('button', { name: 'Copy code' }))
    await waitFor(() =>
      expect(
        within(panel).getByRole('button', { name: 'Copied!' })
      ).toBeVisible()
    )
    const copied = await navigator.clipboard.readText()
    expect(copied).toContain('"model": "nanobanana-2.1"')
    expect(copied).toContain('https://gmvupup.top/v1/images/generations')
    expect(copied).not.toContain('gpt-image-2.6')
  })
  it('copies the selected Seedance model using the installed JSON protocol', async () => {
    const user = userEvent.setup()
    const video = DOC_PAGES.videos
    assert(video.examples)
    render(<CodeExamples examples={video.examples} models={video.models} />)
    await user.selectOptions(
      screen.getByRole('combobox', { name: 'Example model' }),
      'doubao-seedance-2.0'
    )
    await user.click(screen.getByRole('tab', { name: 'Python' }))
    const panel = screen.getByRole('tabpanel', { name: 'Python' })
    await user.click(within(panel).getByRole('button', { name: 'Copy code' }))
    await waitFor(() =>
      expect(
        within(panel).getByRole('button', { name: 'Copied!' })
      ).toBeVisible()
    )
    const copied = await navigator.clipboard.readText()
    expect(copied).toContain('"model": "doubao-seedance-2.0"')
    expect(copied).toContain('"userPrompt":')
    expect(copied).toContain('"duration": 5')
    expect(copied).toContain('"quality": "720p"')
    expect(copied).toContain('"aspectRatio": "16:9"')
    expect(copied).not.toMatch(/"(prompt|seconds|size)":/)
  })
  it('uses live provider prices in the video guide and only documents its video endpoint', async () => {
    useSystemConfigStore.getState().setConfig({
      currency: {
        ...DEFAULT_CURRENCY_CONFIG,
        quotaDisplayType: 'CNY',
        usdExchangeRate: 7.3,
      },
    })
    const model: PricingModel = {
      id: 1,
      completion_ratio: 1,
      model_name: 'doubao-seedance-2.5',
      quota_type: 0,
      model_ratio: 37.5,
      enable_groups: ['default'],
      supported_endpoint_types: ['openai', 'anthropic', 'openai-video'],
      billing_plugin_variants: [
        {
          plugin_key: 'ccg-seedance',
          plugin_name: 'CCG Seedance',
          billing_mode: 'tiered_expr',
          billing_expr:
            'u("quality") == "480p" ? tier("480p", u("seconds") * 0.1) : u("quality") == "720p" ? tier("720p", u("seconds") * 0.2) : tier("1080p", u("seconds") * 0.4)',
          billing_usage_schema: {
            seconds: { type: 'number', unit: 'second' },
            quality: { enum: ['480p', '720p', '1080p'] },
          },
        },
      ],
    }
    const router = await renderDocs('videos', true, [model])
    const table = screen.getAllByRole('table')[0]
    for (const price of ['¥0.73', '¥1.46', '¥2.92']) {
      expect(within(table).getByText(price)).toBeVisible()
    }
    await router.navigate({ to: '/docs/$topic', params: { topic: 'models' } })
    expect(await screen.findByText('openai-video')).toBeVisible()
    expect(screen.queryByText('anthropic')).not.toBeInTheDocument()
    expect(
      screen.queryByText('openai', { exact: true })
    ).not.toBeInTheDocument()
  })
  it('renders an empty catalog without inventing available models', async () => {
    await renderDocs('models')
    expect(await screen.findByText('No models are displayed yet')).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'Copy model ID' })
    ).not.toBeInTheDocument()
  })
  it('shows a catalog failure and allows an explicit retry', async () => {
    const user = userEvent.setup()
    const request = vi
      .spyOn(api, 'get')
      .mockRejectedValue(new Error('fixture network failure'))
    await renderDocs('models', false)
    expect(
      await screen.findByText('Could not load the model catalog')
    ).toBeVisible()
    request.mockResolvedValue({
      data: {
        success: true,
        data: [],
        vendors: [],
        group_ratio: {},
        usable_group: {},
        supported_endpoint: {},
        auto_groups: [],
      },
    })
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('No models are displayed yet')).toBeVisible()
  })
  it('keeps downloadable API paths on GMVUPUP and excludes upstream-only async routes', () => {
    const spec = JSON.parse(
      readFileSync('public/gmvupup/docs/openapi.json', 'utf8')
    )
    expect(spec.servers).toEqual([{ url: API_BASE_URL }])
    expect(Object.keys(spec.paths).sort()).toEqual([
      '/chat/completions',
      '/images/edits',
      '/images/generations',
      '/models',
      '/tasks/{task_id}/artifacts/last_frame/content',
      '/videos',
      '/videos/{task_id}',
      '/videos/{task_id}/content',
    ])
    for (const page of Object.values(DOC_PAGES)) {
      for (const source of Object.values(page.examples ?? {})) {
        expect(source).not.toMatch(/agtoken\.(vip|online)/)
        expect(source).not.toContain('/v1/v1/')
      }
    }
    expect(isDocTopic('__proto__')).toBe(false)
    expect(isDocTopic('not-an-article')).toBe(false)
  })
})
