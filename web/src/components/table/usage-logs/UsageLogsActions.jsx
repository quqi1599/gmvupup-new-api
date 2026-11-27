/*
Copyright (C) 2025 QuantumNous

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

import React from 'react';
import { Button, Space, Skeleton } from '@douyinfe/semi-ui';
import { Activity } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { renderQuota } from '../../../helpers';
import CompactModeToggle from '../../common/ui/CompactModeToggle';
import { useMinimumLoadingTime } from '../../../hooks/common/useMinimumLoadingTime';

function StatPill({ label, value, mark }) {
  return (
    <span className='usage-stat-pill' style={{ '--pill-mark': mark }}>
      {label} <strong>{value}</strong>
    </span>
  );
}

const LogsActions = ({
  stat,
  loadingStat,
  showStat,
  compactMode,
  setCompactMode,
  isAdminUser,
  t,
}) => {
  const navigate = useNavigate();
  const showSkeleton = useMinimumLoadingTime(loadingStat);
  const needSkeleton = !showStat || showSkeleton;

  const placeholder = (
    <Space>
      <Skeleton.Title style={{ width: 108, height: 21, borderRadius: 6 }} />
      <Skeleton.Title style={{ width: 65, height: 21, borderRadius: 6 }} />
      <Skeleton.Title style={{ width: 64, height: 21, borderRadius: 6 }} />
    </Space>
  );

  return (
    <div className='flex flex-col md:flex-row justify-between items-start md:items-center gap-2 w-full'>
      <Skeleton loading={needSkeleton} active placeholder={placeholder}>
        <Space wrap>
          <StatPill
            label={t('消耗额度')}
            value={renderQuota(stat.quota)}
            mark='var(--semi-color-primary)'
          />
          <StatPill
            label='RPM'
            value={stat.rpm}
            mark='var(--semi-color-danger)'
          />
          <StatPill label='TPM' value={stat.tpm} />
        </Space>
      </Skeleton>

      <Space wrap>
        {isAdminUser && (
          <Button
            size='small'
            className='!rounded-full'
            icon={<Activity size={16} />}
            onClick={() => navigate('/console/model-performance')}
          >
            {t('模型监控')}
          </Button>
        )}
        <CompactModeToggle
          compactMode={compactMode}
          setCompactMode={setCompactMode}
          t={t}
        />
      </Space>
    </div>
  );
};

export default LogsActions;
