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
import { useFormContext } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

export function ZpayPreset() {
  const { t } = useTranslation()
  const form = useFormContext<{ PayAddress: string }>()
  return (
    <div className='space-y-3 rounded-lg border p-4'>
      <p className='text-sm'>
        {t(
          'ZPay supports Alipay and WeChat Pay through the Epay protocol. Enter your ZPay merchant ID and secret below, and add alipay or wxpay in General payment methods.'
        )}
      </p>
      <div className='flex flex-wrap items-center gap-3'>
        <Button
          type='button'
          variant='outline'
          onClick={() =>
            form.setValue('PayAddress', 'https://zpayz.cn', {
              shouldDirty: true,
              shouldValidate: true,
            })
          }
        >
          {t('Use ZPay gateway')}
        </Button>
        <a
          href='https://7-pay.cn/doc.html'
          target='_blank'
          rel='noopener noreferrer'
          className='text-primary text-sm underline underline-offset-4'
        >
          {t('ZPay documentation')}
        </a>
      </div>
      <p className='text-muted-foreground text-sm'>
        {t(
          'ZPay charges in CNY. Check the top-up conversion price before saving. The gateway button only fills this form; save to apply.'
        )}
      </p>
    </div>
  )
}
