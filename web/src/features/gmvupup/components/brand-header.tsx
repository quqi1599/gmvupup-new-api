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
// GMVUPUP brand customization, 2026-11-08.
import { Link } from '@tanstack/react-router'
import { ArrowUpRight, Menu } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { LanguageSwitcher } from '@/components/language-switcher'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'

export function BrandMark() {
  return (
    <span className='gmv-mark' aria-hidden='true'>
      <ArrowUpRight />
      <ArrowUpRight />
    </span>
  )
}

export function BrandHeader(props: { isAuthenticated: boolean }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const destination = props.isAuthenticated ? '/dashboard' : '/sign-up'

  const navigation = (
    <>
      <Link to='/' hash='possibilities' onClick={() => setOpen(false)}>
        {t('Creative possibilities')}
      </Link>
      <Link to='/pricing' onClick={() => setOpen(false)}>
        {t('Models & pricing')}
      </Link>
      <Link to='/docs' onClick={() => setOpen(false)}>
        {t('API documentation')}
      </Link>
      <Link to='/' hash='questions' onClick={() => setOpen(false)}>
        {t('Common questions')}
      </Link>
    </>
  )

  return (
    <header className='gmv-header'>
      <div className='gmv-container gmv-header-inner'>
        <Link to='/' className='gmv-brand' aria-label='GMVUPUP'>
          <BrandMark />
          <span>
            GMV<span className='gmv-word-up'>UPUP</span>
          </span>
        </Link>
        <nav className='gmv-desktop-nav' aria-label={t('Main navigation')}>
          {navigation}
        </nav>
        <div className='gmv-header-actions'>
          <LanguageSwitcher />
          <Link
            to={props.isAuthenticated ? '/dashboard' : '/sign-in'}
            className='gmv-login'
          >
            {props.isAuthenticated ? t('Dashboard') : t('Sign in')}
          </Link>
          <Button
            className='gmv-button gmv-header-cta'
            render={<Link to={destination} />}
          >
            {props.isAuthenticated ? t('Open console') : t('Get started')}
            <ArrowUpRight aria-hidden='true' />
          </Button>
          <Sheet open={open} onOpenChange={setOpen}>
            <SheetTrigger
              render={
                <Button
                  className='gmv-menu-button'
                  variant='ghost'
                  size='icon'
                  aria-label={t('Open navigation')}
                />
              }
            >
              <Menu aria-hidden='true' />
            </SheetTrigger>
            <SheetContent className='gmv-mobile-sheet'>
              <SheetTitle>GMVUPUP</SheetTitle>
              <nav aria-label={t('Mobile navigation')}>{navigation}</nav>
              <Button
                className='gmv-button'
                render={
                  <Link to={destination} onClick={() => setOpen(false)} />
                }
              >
                {t('Get started')}
                <ArrowUpRight aria-hidden='true' />
              </Button>
              <Link to='/sign-in' onClick={() => setOpen(false)}>
                {t('Sign in')}
              </Link>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  )
}
