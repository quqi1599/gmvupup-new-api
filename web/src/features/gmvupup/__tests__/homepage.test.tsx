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
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { beforeEach, describe, expect, it } from 'vitest'

import en from '@/i18n/locales/en.json'
import zh from '@/i18n/locales/zh.json'
import { STATUS_QUERY_KEY } from '@/lib/status-query'

import { API_EXAMPLES } from '../api-examples'
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
  it('switches creative formats with keyboard and exposes the selected panel', async () => {
    const user = userEvent.setup()
    await renderHomepage()
    const tabs = screen.getByRole('tablist', { name: 'Creative formats' })
    const video = within(tabs).getByRole('tab', { name: 'Video creation' })
    video.focus()
    await user.keyboard('{ArrowRight}{Enter}')
    const image = within(tabs).getByRole('tab', { name: 'Image creation' })
    await waitFor(() => expect(image).toHaveAttribute('aria-selected', 'true'))
    expect(
      screen.getByRole('heading', { name: /Make the first impression/ })
    ).toBeVisible()
    expect(
      screen.queryByRole('heading', { name: /Give your product/ })
    ).not.toBeInTheDocument()
  })

  it('copies the currently selected code example without making an API request', async () => {
    const user = userEvent.setup()
    await renderHomepage()
    await user.click(screen.getByRole('tab', { name: 'Python' }))
    await user.click(screen.getByRole('button', { name: 'Copy code example' }))
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Copied' })).toBeVisible()
    )
    expect(await navigator.clipboard.readText()).toBe(API_EXAMPLES.python)
    await user.click(screen.getByRole('tab', { name: 'cURL' }))
    expect(screen.getByLabelText('Video API example')).toHaveTextContent(
      'curl https://gmvupup.top/v1/videos'
    )
    expect(API_EXAMPLES.curl).not.toContain('\n+')
  })

  it('opens mobile navigation and closes it when documentation is selected', async () => {
    const user = userEvent.setup()
    await renderHomepage()
    await user.click(screen.getByRole('button', { name: 'Open navigation' }))
    const dialog = screen.getByRole('dialog')
    expect(
      within(dialog).getByRole('navigation', { name: 'Mobile navigation' })
    ).toBeVisible()
    await user.click(
      within(dialog).getByRole('link', { name: 'API documentation' })
    )
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
  })

  it('sends a new visitor to the real registration route', async () => {
    const user = userEvent.setup()
    const router = await renderHomepage()
    await user.click(screen.getByRole('button', { name: 'Start creating' }))
    await waitFor(() => expect(router.state.location.pathname).toBe('/sign-up'))
  })

  it('sends an authenticated visitor to the dashboard and preserves project attribution', async () => {
    await renderHomepage(true)
    for (const link of screen.getAllByRole('button', {
      name: 'Open console',
    })) {
      expect(link).toHaveAttribute('href', '/dashboard')
    }
    expect(
      screen.getByRole('link', { name: 'New API contributors.' })
    ).toHaveAttribute('href', 'https://github.com/QuantumNous/new-api')
    expect(screen.getByText(/Frontend design and development by/)).toBeVisible()
  })

  it('updates the homepage when Chinese is selected and exposes the example disclaimer', async () => {
    const user = userEvent.setup()
    await renderHomepage()
    await user.click(screen.getByRole('button', { name: 'Change language' }))
    await user.click(screen.getByRole('menuitem', { name: '简体中文' }))
    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
        '让创意，'
      )
    )
    await user.click(
      screen.getByRole('button', { name: '页面上的画面是真实客户案例吗？' })
    )
    expect(
      screen.getByText(
        '这些是为官网制作的概念视觉，用于表达创作方向，不是客户案例或特定模型的效果承诺。'
      )
    ).toBeVisible()
  })
})
