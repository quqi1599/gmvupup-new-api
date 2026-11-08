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

import { EmptyState } from '@/components/empty-state'
import { LoadingState } from '@/components/loading-state'
import { RichContent } from '@/components/rich-content'
import { Card } from '@/components/ui/card'
import { useNotifications } from '@/hooks/use-notifications'

export function GmvupupAnnouncements() {
  const { t } = useTranslation()
  const data = useNotifications()
  return (
    <main className='gmvc-catalog'>
      <h1 className='gmvc-page-title'>{t('Announcements')}</h1>
      {data.loading ? (
        <LoadingState />
      ) : (
        <>
          {data.notice && (
            <Card className='gmvc-notice'>
              <RichContent content={data.notice} />
            </Card>
          )}
          {data.announcements.map((item, index) => (
            <Card
              className='gmvc-notice'
              key={
                typeof item.id === 'string' || typeof item.id === 'number'
                  ? item.id
                  : index
              }
            >
              <RichContent
                content={typeof item.content === 'string' ? item.content : ''}
              />
            </Card>
          ))}
          {!data.notice && !data.announcements.length && (
            <EmptyState title={t('No announcements yet')} />
          )}
        </>
      )}
    </main>
  )
}
