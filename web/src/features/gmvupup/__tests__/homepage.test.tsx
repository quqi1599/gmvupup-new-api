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
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { beforeEach, describe, expect, it } from 'vitest'

import en from '@/i18n/locales/en.json'
import zh from '@/i18n/locales/zh.json'
import { STATUS_QUERY_KEY } from '@/lib/status-query'

import { GmvupupHome } from '../index'

async function renderHomepage(isAuthenticated = false) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  client.setQueryData(STATUS_QUERY_KEY, {
    user_agreement_enabled: false,
    privacy_policy_enabled: false,
  })
  const root = createRootRoute({ component: Outlet })
  const home = createRoute({
    getParentRoute: () => root,
    path: '/',
    component: () => <GmvupupHome isAuthenticated={isAuthenticated} />,
  })
  const destinations = [
    '/sign-up',
    '/sign-in',
    '/pricing',
    '/dashboard',
    '/keys',
    '/docs',
  ].map((path) =>
    createRoute({
      getParentRoute: () => root,
      path,
      component: () => <p>{path}</p>,
    })
  )
  const router = createRouter({
    routeTree: root.addChildren([home, ...destinations]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
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
  i18next.addResourceBundle('en', 'translation', en.translation, true, true)
  i18next.addResourceBundle('zhCN', 'translation', zh.translation, true, true)
  await i18next.changeLanguage('en')
})

describe('GMVUPUP public homepage', () => {
  it.each([false, true])(
    'offers only the console link when authenticated=%s',
    async (authenticated) => {
      const user = userEvent.setup()
      const router = await renderHomepage(authenticated)
      const entry = screen.getByRole('button', { name: 'Enter console' })
      expect(screen.getAllByRole('button')).toHaveLength(1)
      expect(screen.queryByRole('link')).not.toBeInTheDocument()
      expect(entry).toHaveAttribute('href', '/dashboard')
      await user.click(entry)
      await waitFor(() =>
        expect(router.state.location.pathname).toBe('/dashboard')
      )
    }
  )
  it('uses the Chinese console label without adding other home actions', async () => {
    await i18next.changeLanguage('zhCN')
    await renderHomepage()
    expect(screen.getByRole('button', { name: '进入控制台' })).toBeVisible()
    expect(screen.getAllByRole('button')).toHaveLength(1)
    expect(
      screen.getByText(
        'Frontend design and development by New API contributors.'
      )
    ).toBeVisible()
    expect(screen.queryByText(/500W|99\.99%/)).not.toBeInTheDocument()
  })
  it('embeds the playable universe while retaining model logos and console access', async () => {
    await renderHomepage()
    const frame = screen.getByTitle('GMV ecommerce universe')
    expect(frame).toHaveAttribute('src', '/gmv-universe.html?embedded=1')
    expect(frame).toHaveAttribute('sandbox', 'allow-scripts allow-downloads')
    for (const name of [
      'GPT Image',
      'Nano Banana',
      'DeepSeek',
      'Seedance',
      'MiniMax',
      'Gemini',
    ]) {
      expect(screen.getByRole('img', { name })).toBeVisible()
    }
    expect(
      screen.getByRole('button', { name: 'Enter console' })
    ).toHaveAttribute('href', '/dashboard')
  })
})
