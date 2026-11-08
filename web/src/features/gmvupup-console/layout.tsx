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
import { Link, useLocation } from '@tanstack/react-router'
import { HelpCircle } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { LanguageSwitcher } from '@/components/language-switcher'
import { NavGroup } from '@/components/layout/components/nav-group'
import { ProfileDropdown } from '@/components/profile-dropdown'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from '@/components/ui/sidebar'
import { GmvupupWordmark } from '@/features/gmvupup/components/wordmark'
import { useSidebarConfig } from '@/hooks/use-sidebar-config'
import { useSidebarData } from '@/hooks/use-sidebar-data'
import { toIntlLocale } from '@/i18n/languages'
import { formatQuotaWithCurrency } from '@/lib/currency'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { getConsoleNavigation } from './navigation'

import '@/styles/gmvupup-landing.css'
import '@/styles/gmvupup-console.css'

export function GmvupupConsoleLayout(props: { children: ReactNode }) {
  const { t, i18n } = useTranslation()
  const path = useLocation({ select: (location) => location.pathname })
  const user = useAuthStore((state) => state.auth.user)
  const currency = useSystemConfigStore((state) => state.config.currency)
  const groups = useSidebarConfig(getConsoleNavigation(t))
  const originalGroups = useSidebarConfig(useSidebarData().navGroups)
  const adminGroups = originalGroups
    .filter((group) => group.id === 'admin')
    .map((group) => ({
      ...group,
      items: group.items.filter(
        (item) => !item.requiredRole || (user?.role ?? 0) >= item.requiredRole
      ),
    }))
  const active = groups
    .flatMap((group) => group.items)
    .find((item) => item.url === path || item.url === path.replace(/\/$/, ''))
  const title = active?.title ?? t('Console')
  const quota =
    typeof user?.quota === 'number'
      ? formatQuotaWithCurrency(user.quota, {
          locale: toIntlLocale(i18n.language),
          abbreviate: false,
        })
      : '—'
  return (
    <SidebarProvider
      className='gmv-console'
      defaultOpen
      style={
        {
          '--sidebar-width': '15.5rem',
          '--app-header-height': '0px',
        } as React.CSSProperties
      }
    >
      <Sidebar
        collapsible='offcanvas'
        variant='sidebar'
        className='gmvc-sidebar'
      >
        <SidebarHeader className='gmvc-brand'>
          <Link to='/dashboard'>
            <GmvupupWordmark />
          </Link>
          <span>{t('AI creative console')}</span>
        </SidebarHeader>
        <SidebarContent>
          <nav aria-label={t('Console navigation')}>
            {groups.map((group) => (
              <NavGroup key={group.id} {...group} />
            ))}
            {(user?.role ?? 0) >= ROLE.ADMIN &&
              adminGroups.map((group) => (
                <NavGroup key={group.id} {...group} />
              ))}
          </nav>
        </SidebarContent>
        <SidebarFooter className='gmvc-sidebar-footer'>
          <Link to='/docs'>
            <HelpCircle size={18} />
            <span>
              {t('Help & documentation')}
              <small>GMVUPUP API</small>
            </span>
          </Link>
          <small>Powered by New API</small>
        </SidebarFooter>
      </Sidebar>
      <SidebarInset className='gmvc-inset'>
        <header className='gmvc-header'>
          <div>
            <SidebarTrigger />
            <strong>{title}</strong>
          </div>
          <div className='gmvc-account'>
            <div className='gmvc-balance' title={currency?.quotaDisplayType}>
              <span>{t('Balance')}</span>
              <strong>{quota}</strong>
            </div>
            <LanguageSwitcher />
            <span className='gmvc-username'>
              {user?.display_name || user?.username}
            </span>
            <ProfileDropdown />
          </div>
        </header>
        <div id='main' className='gmvc-page'>
          {props.children}
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
