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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Banner,
  Button,
  Card,
  Input,
  Select,
  Table,
  Tag,
  Tooltip,
} from '@douyinfe/semi-ui';
import {
  Activity,
  ArrowLeft,
  Gauge,
  Hash,
  HeartPulse,
  RefreshCw,
  Search,
  Timer,
  TriangleAlert,
} from 'lucide-react';
import { API } from '../../helpers/api';
import {
  modelPerformanceLogLink,
  summarizeModelPerformance,
} from '../../helpers/model-performance';
import './official-monitor-style.css';

const minuteOptions = [15, 60, 1440];
const noValue = '—';
const count = (n) => n.toLocaleString();
const seconds = (n) =>
  Number.isFinite(n) ? (n / 1000).toFixed(2) + ' s' : noValue;
const speed = (n) => (Number.isFinite(n) ? n.toFixed(1) + ' t/s' : noValue);
const localTime = (n) =>
  new Date(n * 1000).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
  });
const rate = (n) => (Number.isFinite(n) ? (n * 100).toFixed(1) + '%' : noValue);
const p95 = (row) =>
  Number.isFinite(row.first_p95_ms)
    ? (row.first_p95_over ? '> ' : '≤ ') + seconds(row.first_p95_ms)
    : noValue;

const completionTone = (value) =>
  !Number.isFinite(value)
    ? 'muted'
    : value < 0.7
      ? 'low'
      : value < 0.9
        ? 'watch'
        : 'healthy';

function CompletionBadge({ value }) {
  if (!Number.isFinite(value)) return noValue;
  return (
    <span className={'mp-health-value mp-' + completionTone(value)}>
      <span className='mp-health-dot' aria-hidden='true' />
      {rate(value)}
    </span>
  );
}

function MetricIcon({ icon: Icon, tone = 'info' }) {
  return (
    <span className={'mp-metric-icon mp-' + tone} aria-hidden='true'>
      <Icon size={14} strokeWidth={1.8} />
    </span>
  );
}

function HealthOverview({ summary, t }) {
  return (
    <div className='mp-health-strip'>
      <h2 className='mp-health-title'>
        <MetricIcon icon={HeartPulse} tone='healthy' />
        {t('性能健康')}
      </h2>
      {summary.samples === 0 ? (
        <span className='mp-muted'>{t('无样本')}</span>
      ) : (
        <dl className='mp-health-values'>
          <div className='mp-inline-metric'>
            <dt>
              <MetricIcon icon={Hash} />
              {t('最终请求')}
            </dt>
            <dd>{count(summary.samples)}</dd>
          </div>
          <div className='mp-inline-metric'>
            <dt>
              <MetricIcon
                icon={Activity}
                tone={completionTone(summary.completionRate)}
              />
              {t('完成率')}
            </dt>
            <dd>
              <CompletionBadge value={summary.completionRate} />
            </dd>
          </div>
          <div className='mp-inline-metric'>
            <dt>
              <MetricIcon
                icon={TriangleAlert}
                tone={summary.noRoute ? 'watch' : 'muted'}
              />
              {t('无可用渠道')}
            </dt>
            <dd>{count(summary.noRoute)}</dd>
          </div>
        </dl>
      )}
    </div>
  );
}

function ModelKeyMetrics({ row, t }) {
  const values = [
    [
      Activity,
      '完成率',
      <CompletionBadge value={row.completion_rate} />,
      completionTone(row.completion_rate),
    ],
    [Timer, '首包 P95', p95(row), 'info'],
    [Gauge, '流式生成速度', speed(row.generation_tps), 'info'],
  ];
  return (
    <dl className='mp-kpi-grid'>
      {values.map(([icon, label, value, tone]) => (
        <div key={label} className='mp-kpi'>
          <dt>
            <MetricIcon icon={icon} tone={tone} />
            {t(label)}
          </dt>
          <dd>{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function usePerformance(minutes, model, revision, enabled = true) {
  const scope = JSON.stringify([minutes, model]);
  const [result, setResult] = useState(null);
  useEffect(() => {
    if (!enabled) return undefined;
    const ctrl = new AbortController();
    setResult((prior) => ({
      scope,
      pending: true,
      data: prior?.scope === scope ? prior.data : null,
    }));
    API.get('/api/model-performance', {
      params: model ? { minutes, model } : { minutes },
      signal: ctrl.signal,
      disableDuplicate: true,
      skipErrorHandler: true,
    })
      .then(({ data }) => {
        if (ctrl.signal.aborted) return;
        setResult({
          scope,
          data: data.success ? data.data : null,
          error: !data.success,
        });
      })
      .catch(() => {
        if (!ctrl.signal.aborted) setResult({ scope, error: true });
      });
    return () => ctrl.abort();
  }, [minutes, model, revision, enabled, scope]);
  return enabled && result?.scope === scope ? result : { pending: enabled };
}

function Trend({ title, snapshot, metric, text }) {
  const t = useTranslation().t;
  const start = snapshot.from;
  const width = 420,
    height = 114,
    left = 42,
    right = 9,
    top = 12,
    bottom = 25;
  const step = snapshot.bucket_seconds;
  const slots = Math.max(1, Math.floor((snapshot.to - start) / step));
  const points = snapshot.series.flatMap((row) => {
    const value = row[metric];
    return Number.isFinite(value) &&
      !(metric === 'first_p95_ms' && row.first_p95_over)
      ? [
          {
            slot: (row.bucket_start - start) / step,
            value,
            bucket: row.bucket_start,
          },
        ]
      : [];
  });
  const maxValue = Math.max(1, ...points.map((p) => p.value));
  const x = (index) => left + (index * (width - left - right)) / slots;
  const y = (value) =>
    height - bottom - (value / maxValue) * (height - top - bottom);
  const segments = [];
  for (const p of points) {
    const last = segments.at(-1);
    if (last && last.at(-1).slot + 1 === p.slot) last.push(p);
    else segments.push([p]);
  }
  return (
    <div className='min-w-0'>
      <h3 className='mb-2 text-sm font-medium'>{title}</h3>
      {points.length === 0 ? (
        <div className='flex min-h-24 items-center text-sm text-[var(--semi-color-text-2)]'>
          {t('无样本')}
        </div>
      ) : (
        <svg
          role='img'
          aria-label={title}
          viewBox={'0 0 ' + width + ' ' + height}
          className='w-full max-h-36 text-[var(--semi-color-primary)]'
        >
          <path
            d={'M ' + left + ' ' + (height - bottom) + ' H ' + (width - right)}
            stroke='var(--semi-color-border)'
          />
          <path
            d={'M ' + left + ' ' + top + ' H ' + (width - right)}
            stroke='var(--semi-color-border)'
            strokeDasharray='3 3'
          />
          <g fill='var(--semi-color-text-2)' fontSize='10'>
            <text x='0' y={top + 3}>
              {text(maxValue)}
            </text>
            <text x='0' y={height - bottom + 3}>
              0
            </text>
            <text x={left} y={height - 5}>
              {localTime(start)}
            </text>
            <text x={width - right} y={height - 5} textAnchor='end'>
              {localTime(snapshot.to)}
            </text>
          </g>
          {segments
            .filter((s) => s.length > 1)
            .map((s) => (
              <polyline
                key={s[0].slot}
                fill='none'
                stroke='currentColor'
                strokeWidth='1.6'
                points={s.map((p) => x(p.slot) + ',' + y(p.value)).join(' ')}
              />
            ))}
          {points.map((p) => (
            <circle
              key={p.slot}
              cx={x(p.slot)}
              cy={y(p.value)}
              r='2.1'
              fill='currentColor'
            >
              <title>
                {localTime(p.bucket)} · {text(p.value)}
              </title>
            </circle>
          ))}
        </svg>
      )}
    </div>
  );
}

function OutcomeTags({ row, t }) {
  const kinds = [
    ['no_route', '无可用渠道', 'amber'],
    ['upstream_429', '上游 429', 'orange'],
    ['upstream_5xx', '上游 5xx', 'red'],
    ['stream_error', '流异常', 'red'],
    ['gateway_limit', '本地限流', 'amber'],
    ['other_error', '其他错误', 'red'],
    ['limited', '正常限长', 'grey'],
    ['canceled', '客户端取消', 'grey'],
    ['rejected', '请求拒绝', 'grey'],
  ];
  const observed = kinds.filter(([key]) => row[key] > 0);
  if (observed.length === 0) return noValue;
  return (
    <div className='flex flex-wrap gap-1'>
      {observed.map(([key, label, color]) => (
        <Tag key={key} color={color} size='small' shape='circle'>
          {t(label)} {count(row[key])}
        </Tag>
      ))}
    </div>
  );
}

const numeric = (field) => (a, b) => (a[field] ?? -1) - (b[field] ?? -1);

function MobileMetrics({
  rows,
  t,
  channels = false,
  openModel,
  logButton,
  loading,
}) {
  const [limit, setLimit] = useState(8);
  if (!rows.length) {
    return (
      <p className='p-2 text-sm text-[var(--semi-color-text-2)]'>
        {t(loading ? '加载中' : channels ? '无渠道尝试' : '无样本')}
      </p>
    );
  }
  return (
    <div className='divide-y divide-[var(--semi-color-border)]'>
      {rows.slice(0, limit).map((row) => (
        <article
          key={channels ? row.channel_id : row.model}
          className='py-3 tabular-nums'
        >
          <div className='mb-2 flex flex-wrap items-center justify-between gap-x-1 gap-y-2'>
            {channels ? (
              <span className='break-all text-sm font-medium'>
                #{row.channel_id} {row.channel_name}
              </span>
            ) : (
              <Button
                size='small'
                theme='borderless'
                className='mp-model-pill'
                onClick={() => openModel(row.model)}
              >
                <span className='max-w-52 truncate'>{row.model}</span>
              </Button>
            )}
            {channels && logButton(row.channel_id)}
          </div>
          {row.samples > 0 ? (
            <>
              <dl className='grid grid-cols-2 gap-2 text-sm'>
                {[
                  [channels ? '尝试次数' : '最终请求', count(row.samples)],
                  ['完成率', <CompletionBadge value={row.completion_rate} />],
                  ['首包 P95', p95(row)],
                  ['流式生成速度', speed(row.generation_tps)],
                  ['平均总用时', seconds(row.mean_duration_ms)],
                ].map(([label, value]) => (
                  <div key={label}>
                    <dt className='text-xs text-[var(--semi-color-text-2)]'>
                      {t(label)}
                    </dt>
                    <dd className='font-medium'>{value}</dd>
                  </div>
                ))}
              </dl>
              <div className='mt-2'>
                <OutcomeTags row={row} t={t} />
              </div>
            </>
          ) : (
            <span className='text-xs text-[var(--semi-color-text-2)]'>
              {t('无样本')}
            </span>
          )}
        </article>
      ))}
      {rows.length > limit && (
        <Button
          size='small'
          theme='borderless'
          onClick={() => setLimit((n) => n + 8)}
        >
          {t('查看更多')}
        </Button>
      )}
    </div>
  );
}

export default function ModelPerformance() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const [needle, setNeedle] = useState('');
  const [revision, setRevision] = useState(0);
  const detailRef = useRef(null);
  const paramMinutes = Number(params.get('minutes'));
  const minutes = minuteOptions.includes(paramMinutes) ? paramMinutes : 60;
  const selected = params.get('model') || '';
  const all = usePerformance(minutes, '', revision);
  const detail = usePerformance(minutes, selected, revision, Boolean(selected));
  const report = detail.data;
  const active = report?.models?.[0];

  useEffect(() => {
    const timer = setInterval(() => {
      if (document.visibilityState === 'visible') setRevision((n) => n + 1);
    }, 30000);
    return () => clearInterval(timer);
  }, []);

  function updateParam(name, value) {
    const next = new URLSearchParams(params);
    if (value) next.set(name, value);
    else next.delete(name);
    setParams(next, { replace: true });
  }
  function openModel(model) {
    updateParam('model', model);
    requestAnimationFrame(() =>
      detailRef.current?.scrollIntoView({ block: 'start' }),
    );
  }
  const matching = useMemo(
    () =>
      (all.data?.models ?? []).filter((row) =>
        row.model.toLowerCase().includes(needle.trim().toLowerCase()),
      ),
    [all.data, needle],
  );
  const health = useMemo(() => summarizeModelPerformance(matching), [matching]);
  const parts = (row) => <OutcomeTags row={row} t={t} />;
  const sharedColumns = (samplesTitle) => [
    {
      title: samplesTitle,
      dataIndex: 'samples',
      width: 106,
      align: 'right',
      sorter: numeric('samples'),
      render: (n) => (n > 0 ? count(n) : t('无样本')),
    },
    {
      title: (
        <Tooltip
          content={t(
            '完成率 = 完成 ÷（样本 − 客户端取消 − 请求拒绝）；正常限长单列。',
          )}
        >
          <span>{t('完成率')}</span>
        </Tooltip>
      ),
      dataIndex: 'completion_rate',
      width: 110,
      align: 'right',
      sorter: numeric('completion_rate'),
      render: (n) => <CompletionBadge value={n} />,
    },
    {
      title: (
        <Tooltip
          content={t('首包 P95 为完成或正常限长流中首个有效事件的分桶上界。')}
        >
          <span>{t('首包 P95')}</span>
        </Tooltip>
      ),
      dataIndex: 'first_p95_ms',
      width: 110,
      align: 'right',
      sorter: numeric('first_p95_ms'),
      render: (_, row) => (
        <span title={t('首包样本') + ': ' + row.first_samples}>{p95(row)}</span>
      ),
    },
    {
      title: t('平均总用时'),
      dataIndex: 'mean_duration_ms',
      width: 114,
      align: 'right',
      sorter: numeric('mean_duration_ms'),
      render: seconds,
    },
    {
      title: (
        <Tooltip
          content={t('流式生成速度 = 输出 token 总数 ÷ 首包至结束的总秒数。')}
        >
          <span>{t('流式生成速度')}</span>
        </Tooltip>
      ),
      dataIndex: 'generation_tps',
      width: 135,
      align: 'right',
      sorter: numeric('generation_tps'),
      render: (n, row) => (
        <span title={t('速度样本') + ': ' + row.speed_samples}>{speed(n)}</span>
      ),
    },
    {
      title: t('异常与终态'),
      key: 'outcomes',
      width: 248,
      render: (_, row) => parts(row),
    },
  ];
  const drill = (channel = 0) =>
    modelPerformanceLogLink(selected, channel, report?.from, report?.to);
  const logButton = (channel = 0) => {
    const link = drill(channel);
    return link ? (
      <Button
        size='small'
        theme='borderless'
        className='mp-link'
        onClick={() => navigate(link)}
      >
        {t('近 15 分钟日志')}
      </Button>
    ) : null;
  };
  const tables = {
    size: 'small',
    className: 'mp-table tabular-nums',
    scroll: { x: 1120 },
    pagination: { pageSize: 12, hideOnSinglePage: true },
  };

  return (
    <main className='mp-console mt-[60px] px-2 min-w-0 flex flex-col gap-4'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div>
          <Button
            theme='borderless'
            size='small'
            className='mp-link'
            icon={<ArrowLeft size={15} />}
            onClick={() => navigate('/console/log')}
          >
            {t('使用日志')}
          </Button>
          <h1 className='text-xl font-semibold mt-1'>{t('模型监控')}</h1>
        </div>
        <div className='mp-controls flex flex-wrap items-center gap-2'>
          <Select
            aria-label={t('观察窗口')}
            value={minutes}
            onChange={(value) => updateParam('minutes', value)}
            optionList={[
              { value: 15, label: t('最近 15 分钟') },
              { value: 60, label: t('最近 1 小时') },
              { value: 1440, label: t('最近 24 小时') },
            ]}
            style={{ width: 158 }}
          />
          <Button
            icon={<RefreshCw size={15} />}
            loading={Boolean(all.pending || detail.pending)}
            onClick={() => setRevision((n) => n + 1)}
          >
            {t('刷新')}
          </Button>
        </div>
      </div>

      {all.error && (
        <Banner
          fullMode={false}
          type='danger'
          description={t('监控数据暂不可读')}
        />
      )}
      {all.data && !all.data.ready && (
        <Banner
          fullMode={false}
          type='info'
          description={t('监控尚未就绪或已关闭')}
        />
      )}
      {(all.data?.write_delayed || all.data?.dropped_on_this_node > 0) && (
        <Banner
          fullMode={false}
          type='warning'
          description={t('当前节点统计有延迟或缺样，结果可能不完整。')}
        />
      )}
      {all.data?.ready && (
        <div className='text-xs text-[var(--semi-color-text-2)]'>
          {new Date(all.data.from * 1000).toLocaleString()} –{' '}
          {new Date(all.data.to * 1000).toLocaleString()}
          {' · '}
          {t('按 5 分钟汇总；仅统计已识别模型的同步文本请求。')}
        </div>
      )}
      {all.data?.ready && <HealthOverview summary={health} t={t} />}

      <Card title={t('模型表现')} className='mp-panel min-w-0'>
        <Input
          className='mp-search mb-3 max-w-xs'
          value={needle}
          onChange={setNeedle}
          showClear
          aria-label={t('模型名称')}
          prefix={<Search size={15} />}
          placeholder={t('模型名称')}
        />
        <div className='md:hidden'>
          <MobileMetrics
            key={minutes + ':' + needle}
            rows={matching}
            t={t}
            openModel={openModel}
            loading={all.pending}
          />
        </div>
        <div className='hidden md:block min-w-0 overflow-x-auto'>
          <Table
            {...tables}
            rowKey='model'
            loading={all.pending}
            dataSource={matching}
            empty={t('无样本')}
            columns={[
              {
                title: t('模型'),
                dataIndex: 'model',
                width: 220,
                render: (name) => (
                  <Button
                    theme='borderless'
                    size='small'
                    className='mp-model-pill'
                    onClick={() => openModel(name)}
                    aria-pressed={selected === name}
                  >
                    <span className='max-w-52 truncate' title={name}>
                      {name}
                    </span>
                  </Button>
                ),
              },
              ...sharedColumns(t('最终请求')),
            ]}
          />
        </div>
      </Card>

      {selected && (
        <section
          ref={detailRef}
          className='min-w-0 scroll-mt-20 flex flex-col gap-3'
        >
          <div className='flex flex-wrap items-center justify-between gap-2'>
            <h2 className='min-w-0 break-all text-base font-semibold'>
              {selected}
            </h2>
            <div className='flex flex-wrap items-center gap-1'>
              {active?.samples > 0 && logButton()}
              <Button
                theme='borderless'
                onClick={() => updateParam('model', '')}
              >
                {t('关闭详情')}
              </Button>
            </div>
          </div>
          {detail.error && (
            <Banner
              fullMode={false}
              type='danger'
              description={t('监控数据暂不可读')}
            />
          )}
          {detail.pending && !report && <div role='status'>{t('加载中')}</div>}
          {report?.ready && (!active || active.samples === 0) && (
            <Banner
              fullMode={false}
              type='info'
              description={t('该模型在此时段无样本')}
            />
          )}
          {active?.samples > 0 && (
            <>
              <ModelKeyMetrics row={active} t={t} />
              <Card title={t('模型趋势')} className='mp-panel'>
                <div className='grid grid-cols-1 lg:grid-cols-2 gap-4'>
                  <Trend
                    title={t('首包 P95')}
                    snapshot={report}
                    metric='first_p95_ms'
                    text={seconds}
                  />
                  <Trend
                    title={t('流式生成速度')}
                    snapshot={report}
                    metric='generation_tps'
                    text={speed}
                  />
                </div>
              </Card>
              <Card title={t('渠道尝试')} className='mp-panel min-w-0'>
                <p className='mb-3 text-xs text-[var(--semi-color-text-2)]'>
                  {t(
                    '同一请求重试可产生多次渠道尝试；它们不计入模型最终请求数。',
                  )}
                </p>
                <div className='md:hidden'>
                  <MobileMetrics
                    key={selected + ':' + minutes}
                    rows={report.channels}
                    t={t}
                    channels
                    logButton={logButton}
                  />
                </div>
                <div className='hidden md:block min-w-0 overflow-x-auto'>
                  <Table
                    {...tables}
                    rowKey='channel_id'
                    dataSource={report.channels}
                    empty={t('无渠道尝试')}
                    columns={[
                      {
                        title: t('渠道'),
                        dataIndex: 'channel_id',
                        width: 190,
                        render: (id, row) => (
                          <span className='break-words'>
                            #{id} {row.channel_name}
                          </span>
                        ),
                      },
                      ...sharedColumns(t('尝试次数')),
                      {
                        title: t('日志'),
                        key: 'logs',
                        render: (_, row) => logButton(row.channel_id),
                      },
                    ]}
                  />
                </div>
              </Card>
            </>
          )}
        </section>
      )}
    </main>
  );
}
