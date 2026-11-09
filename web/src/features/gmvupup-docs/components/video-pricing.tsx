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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { isCcgSeedanceModel } from '@/features/gmvupup/api-examples'
import { ModelPriceCell } from '@/features/pricing/components/model-price-cell'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'

export function VideoPricing() {
  const { t } = useTranslation()
  const pricing = usePricingData()
  const models = pricing.models.filter(isCcgSeedanceModel)
  return (
    <section id='pricing' className='gmv-doc-section'>
      <h2>{t('Current video pricing')}</h2>
      <p className='gmv-doc-paragraph'>
        {t(
          'Rates below come from the live catalog for the default group. Charge = output duration × resolution rate. Check your group in the console.'
        )}
      </p>
      <p className='gmv-doc-paragraph'>
        {t(
          'The fixed per-second rates use 80% of the China-region official 16:9 examples without reference video, rather than billing by video tokens.'
        )}
      </p>
      {pricing.isLoading && <LoadingState />}
      {pricing.error && (
        <ErrorState
          title={t('Could not load the model catalog')}
          onRetry={() => void pricing.refetch()}
        />
      )}
      {!pricing.isLoading &&
        !pricing.error &&
        (models.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Model')}</TableHead>
                <TableHead>{t('Video pricing')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {models.map((model) => (
                <TableRow key={model.model_name}>
                  <TableCell className='font-mono text-xs break-all'>
                    {model.model_name}
                  </TableCell>
                  <TableCell>
                    <ModelPriceCell
                      model={model}
                      options={{
                        selectedGroup: 'default',
                        priceRate: pricing.priceRate,
                        usdExchangeRate: pricing.usdExchangeRate,
                      }}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <p>{t('Price pending configuration')}</p>
        ))}
      <Link to='/pricing' search={{ search: 'doubao-seedance' }}>
        {t('View live pricing')}
      </Link>
    </section>
  )
}
