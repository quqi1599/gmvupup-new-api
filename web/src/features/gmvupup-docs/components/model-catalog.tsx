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
import { ArrowUpRight, Box } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'

export function ModelCatalog() {
  const { t } = useTranslation()
  const { models, isLoading, error, refetch } = usePricingData()
  if (isLoading) return <LoadingState />
  if (error) {
    return (
      <ErrorState
        title={t('Could not load the model catalog')}
        description={t(
          'Retry or open the pricing page to check available models.'
        )}
        onRetry={() => {
          void refetch()
        }}
      />
    )
  }
  if (!models.length) {
    return (
      <EmptyState
        title={t('No models are displayed yet')}
        description={t(
          'Check the console or query /v1/models with your own API key.'
        )}
      />
    )
  }
  return (
    <div className='gmv-doc-model-catalog'>
      {models.map((model) => (
        <div className='gmv-doc-model-row' key={model.model_name}>
          <Box size={20} aria-hidden='true' />
          <div>
            <code>{model.model_name}</code>
            <div className='gmv-doc-endpoint-tags'>
              {(model.supported_endpoint_types ?? []).map((endpoint) => (
                <span key={endpoint}>{endpoint}</span>
              ))}
            </div>
          </div>
          <div className='gmv-doc-model-actions'>
            <CopyButton
              value={model.model_name}
              aria-label={t('Copy model ID')}
            />
            <Link
              to='/pricing'
              search={{ search: model.model_name }}
              aria-label={`${t('View live pricing')}: ${model.model_name}`}
            >
              <ArrowUpRight size={17} aria-hidden='true' />
            </Link>
          </div>
        </div>
      ))}
    </div>
  )
}
