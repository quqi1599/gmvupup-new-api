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
import { render, renderHook, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { STATUS_QUERY_KEY } from '@/lib/status-query'

import { ModelDetailsApi } from '../components/model-details-api'
import { usePricingData } from '../hooks/use-pricing-data'
import type { PricingModel } from '../types'

let client: QueryClient
const model = (
  name: string,
  extra: Partial<PricingModel> = {}
): PricingModel => ({
  id: 1,
  model_name: name,
  quota_type: 0,
  model_ratio: 1,
  completion_ratio: 1,
  enable_groups: ['default'],
  ...extra,
})
const endpointMap = {
  openai: { path: '/v1/chat/completions', method: 'POST' },
  gemini: { path: '/v1beta/models/{model}:generateContent', method: 'POST' },
  'image-generation': { path: '/v1/images/generations', method: 'POST' },
  'image-edits': { path: '/v1/images/edits', method: 'POST' },
  'openai-video': { path: '/v1/videos', method: 'POST' },
}
function Wrapper(props: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={client}>{props.children}</QueryClientProvider>
  )
}
beforeEach(() => {
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  client.setQueryData(STATUS_QUERY_KEY, {
    server_address: 'https://gmvupup.top',
    price: 7.3,
    usd_exchange_rate: 7.3,
    quota_display_type: 'CNY',
  })
})
afterEach(() => {
  client.clear()
  vi.restoreAllMocks()
})

it('lists image and video interfaces instead of the generic chat fallback', async () => {
  const fixtures = [
    model('MiniMax-H3-Image', {
      supported_endpoint_types: ['openai', 'image-generation'],
    }),
    model('MiniMax-H3', {
      supported_endpoint_types: ['openai', 'openai-video'],
    }),
    model('wan3.0-video', {
      supported_endpoint_types: ['openai', 'anthropic'],
    }),
  ]
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: fixtures,
      vendors: [],
      group_ratio: { default: 1 },
      supported_endpoint: endpointMap,
    },
  })
  const hook = renderHook(() => usePricingData(), { wrapper: Wrapper })
  await waitFor(() => expect(hook.result.current.models).toHaveLength(3))
  expect(
    hook.result.current.models.map((m) => m.supported_endpoint_types)
  ).toEqual([['image-generation', 'image-edits'], ['hailuo'], ['alibaba']])
})

it('copies JSON image edits with a reference image and no chat messages', async () => {
  const user = userEvent.setup()
  render(
    <ModelDetailsApi
      model={model('MiniMax-H3-Image', {
        supported_endpoint_types: ['image-generation', 'image-edits'],
      })}
      endpointMap={endpointMap}
    />,
    { wrapper: Wrapper }
  )
  await user.click(screen.getByRole('tab', { name: 'image-edits' }))
  expect(document.body.textContent).toContain('/v1/images/edits')
  expect(document.body.textContent).toContain('image_url')
  expect(document.body.textContent).not.toContain('messages')
  expect(document.body.textContent).not.toContain('disable_prompt_enhanced')
})

it.each(['Python', 'TypeScript', 'JavaScript'])(
  'keeps the Gemini %s example on the platform with bearer authentication',
  async (language) => {
    const user = userEvent.setup()
    render(
      <ModelDetailsApi
        model={model('gemini-3.8-flash', {
          supported_endpoint_types: ['gemini'],
        })}
        endpointMap={endpointMap}
      />,
      { wrapper: Wrapper }
    )
    await user.click(screen.getByRole('tab', { name: language }))
    expect(document.body.textContent).toContain(
      'https://gmvupup.top/v1beta/models/gemini-3.8-flash:generateContent'
    )
    expect(document.body.textContent).toContain('Bearer ')
    expect(document.body.textContent).toContain('NEW_API_KEY')
  }
)

it.each([
  ['MiniMax-H3', '/v2/video_generation', '768P'],
  [
    'wan3.0-video',
    '/ali/api/v1/services/aigc/video-generation/video-synthesis',
    '480P',
  ],
])(
  'shows the %s native request instead of chat completion fields',
  async (name, path, resolution) => {
    render(
      <ModelDetailsApi
        model={model(name, {
          supported_endpoint_types: ['openai', 'openai-video'],
        })}
        endpointMap={endpointMap}
      />,
      { wrapper: Wrapper }
    )
    expect(document.body.textContent).toContain(`https://gmvupup.top${path}`)
    expect(document.body.textContent).toContain(resolution)
    expect(document.body.textContent).not.toContain('messages')
    expect(document.body.textContent).not.toContain('temperature')
  }
)

it('does not publish invented per-group rate limits', () => {
  render(
    <ModelDetailsApi
      model={model('deepseek-v4.1-flash', {
        supported_endpoint_types: ['openai'],
      })}
      endpointMap={endpointMap}
    />,
    { wrapper: Wrapper }
  )
  expect(
    screen.queryByText('Rate limits', { exact: true })
  ).not.toBeInTheDocument()
  expect(screen.queryByText('RPM', { exact: true })).not.toBeInTheDocument()
})
