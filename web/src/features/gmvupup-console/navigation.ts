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
import {
  Bell,
  User,
  LayoutDashboard,
  Layers,
  FileText,
  Wallet,
  ReceiptText,
  ListVideo,
  KeyRound,
  BookOpen,
  FlaskConical,
  ShieldCheck,
} from 'lucide-react'

import type { NavGroup } from '@/components/layout/types'

export function getConsoleNavigation(t: (key: string) => string): NavGroup[] {
  return [
    {
      id: 'console',
      title: t('Workspace'),
      items: [
        { title: t('Announcements'), url: '/announcements', icon: Bell },
        { title: t('Profile'), url: '/profile', icon: User },
        {
          title: t('Dashboard'),
          url: '/dashboard/models',
          icon: LayoutDashboard,
        },
        { title: t('Model list'), url: '/dashboard/overview', icon: Layers },
        { title: t('Usage Logs'), url: '/usage-logs/common', icon: FileText },
        { title: t('Recharge'), url: '/wallet', icon: Wallet },
        {
          title: t('Orders'),
          url: '/orders',
          configUrls: ['/wallet'],
          icon: ReceiptText,
        },
        { title: t('Task Logs'), url: '/usage-logs/task', icon: ListVideo },
      ],
    },
    {
      id: 'api',
      title: 'API',
      items: [
        { title: t('API Keys'), url: '/keys', icon: KeyRound },
        { title: t('API documentation'), url: '/docs', icon: BookOpen },
        { title: t('Playground'), url: '/playground', icon: FlaskConical },
        { title: t('Security & Access'), url: '/security', icon: ShieldCheck },
      ],
    },
  ]
}
