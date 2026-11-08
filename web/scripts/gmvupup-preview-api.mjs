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
// Local read-only fixtures for GMVUPUP visual QA. Never contacts a live service.
import { createServer } from 'node:http'

const port = Number(process.env.GMVUPUP_PREVIEW_API_PORT ?? 4181)

const fixtures = {
  '/api/setup': {
    success: true,
    data: { status: true, root_initialized: true },
  },
  '/api/status': {
    success: true,
    data: {
      system_name: 'GMVUPUP',
      logo: '/gmvupup/mark.svg',
      version: 'local-preview',
      demo_site_enabled: false,
      password_login: true,
      password_register: true,
      email_verification: false,
      turnstile_check: false,
      user_agreement_enabled: false,
      privacy_policy_enabled: false,
    },
  },
  '/api/home_page_content': { success: true, data: '' },
  '/api/notice': { success: true, data: '' },
  '/api/pricing': {
    success: true,
    data: [],
    vendors: [],
    group_ratio: {},
    usable_group: {},
    supported_endpoint: {},
  },
}
createServer((request, response) => {
  const pathname = new URL(request.url, 'http://127.0.0.1').pathname
  response.setHeader('Content-Type', 'application/json; charset=utf-8')
  response.setHeader('Cache-Control', 'no-store')
  if (request.method !== 'GET') {
    response.writeHead(405)
    response.end(
      JSON.stringify({
        success: false,
        message:
          'Local preview: account and inference operations are disabled.',
      })
    )
    return
  }
  const fixture = fixtures[pathname]
  response.writeHead(fixture ? 200 : 401)
  response.end(
    JSON.stringify(
      fixture ?? {
        success: false,
        message: 'Local preview: sign in to a real instance to continue.',
      }
    )
  )
}).listen(port, '127.0.0.1', () =>
  console.log(`Read-only preview fixtures: http://127.0.0.1:${port}`)
)
