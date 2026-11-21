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

import { cn } from '@/lib/utils'

import type { VideoSecondPricing } from '../lib/task-price-display'

export function VideoResolutionPrices(
  props: VideoSecondPricing & { tiles?: boolean }
) {
  const { t } = useTranslation()
  return (
    <div className='col-span-full min-w-0 space-y-2'>
      <dl
        className={cn(
          'grid min-w-0 gap-2 text-sm',
          props.tiles &&
            'grid-cols-[repeat(auto-fit,minmax(min(100%,5.5rem),1fr))]'
        )}
      >
        {props.rows.map((row) => (
          <div
            key={row.resolution}
            className={cn(
              'min-w-0',
              props.tiles
                ? 'border-border/60 bg-card/65 space-y-1.5 rounded-md border px-2.5 py-2'
                : 'flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1'
            )}
          >
            <dt className='text-muted-foreground'>{row.resolution}</dt>
            <dd className='min-w-0 space-y-1 font-mono tabular-nums'>
              <div>
                {row.referencePrice && (
                  <span className='text-muted-foreground block font-sans text-xs'>
                    {t('No reference video')}
                  </span>
                )}
                <strong>{row.price}</strong>
                <span className='text-muted-foreground whitespace-nowrap'>
                  {' '}
                  / {t('s')}
                </span>
              </div>
              {row.referencePrice && (
                <div>
                  <span className='text-muted-foreground block font-sans text-xs'>
                    {t('With reference video')}
                  </span>
                  <strong>{row.referencePrice}</strong>
                  <span className='text-muted-foreground whitespace-nowrap'>
                    {' '}
                    / {t('s')}
                  </span>
                </div>
              )}
            </dd>
          </div>
        ))}
      </dl>
      {props.estimated && (
        <p className='text-muted-foreground text-xs'>
          {t(
            'Estimated at 16:9, 24 fps. With reference video, count input and output duration subject to a minimum. Final charges use actual video tokens.'
          )}
        </p>
      )}
      {props.extraImagePrice !== undefined && (
        <p className='text-muted-foreground text-xs'>
          {t(
            'Add the input and output video seconds, then multiply by the output resolution rate. The first 5 input images are free, and each additional image costs {{price}}. Audio has no separate charge.',
            { price: props.extraImagePrice }
          )}
        </p>
      )}
    </div>
  )
}
