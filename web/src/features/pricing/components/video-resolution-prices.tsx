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
import { useTranslation } from 'react-i18next'

export function VideoResolutionPrices(props: {
  rows: { resolution: string; price: string }[]
}) {
  const { t } = useTranslation()
  return (
    <dl className='col-span-full grid min-w-0 gap-2 text-sm'>
      {props.rows.map((row) => (
        <div
          key={row.resolution}
          className='flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1'
        >
          <dt className='text-muted-foreground'>{row.resolution}</dt>
          <dd className='font-mono tabular-nums'>
            <strong>{row.price}</strong>
            <span className='text-muted-foreground'> / {t('s')}</span>
          </dd>
        </div>
      ))}
    </dl>
  )
}
