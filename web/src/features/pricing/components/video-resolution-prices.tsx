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

import type { VideoSecondPricing } from '../lib/task-price-display'

export function VideoResolutionPrices(
  props: VideoSecondPricing & { showReference?: boolean }
) {
  const { t } = useTranslation()
  return (
    <div className='col-span-full min-w-0 space-y-2'>
      <dl className='grid min-w-0 gap-2 text-sm'>
        {props.rows.map((row) => (
          <div
            key={row.resolution}
            className='flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1'
          >
            <dt className='text-muted-foreground'>{row.resolution}</dt>
            <dd className='font-mono tabular-nums'>
              {props.showReference && row.referencePrice && (
                <span className='text-muted-foreground mr-2 font-sans text-xs'>
                  {t('No reference video')}
                </span>
              )}
              <strong>{row.price}</strong>
              <span className='text-muted-foreground'> / {t('s')}</span>
              {props.showReference && row.referencePrice && (
                <div className='mt-1'>
                  <span className='text-muted-foreground mr-2 font-sans text-xs'>
                    {t('With reference video')}
                  </span>
                  <strong>{row.referencePrice}</strong>
                  <span className='text-muted-foreground'> / {t('s')}</span>
                </div>
              )}
            </dd>
          </div>
        ))}
      </dl>
      {props.estimated && (
        <p className='text-muted-foreground text-xs'>
          {t(
            props.showReference
              ? 'Estimated at 16:9, 24 fps. Reference video pricing applies to input plus output duration, with a possible minimum charge. Final cost depends on actual video specifications.'
              : 'Estimated at 16:9, 24 fps, without reference video. See details for reference video pricing.'
          )}
        </p>
      )}
      {props.extraImagePrice !== undefined && (
        <p className='text-muted-foreground text-xs'>
          {t(
            'Input video is billed at the same rate. The first 5 reference images are free; each additional image costs {{price}}.',
            { price: props.extraImagePrice }
          )}
        </p>
      )}
    </div>
  )
}
