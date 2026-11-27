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

const isPositiveNumber = (value) => Number.isFinite(value) && value > 0;

export function getUsageLogTiming(log, other) {
  const total = isPositiveNumber(other?.duration_ms)
    ? other.duration_ms / 1000
    : isPositiveNumber(log.use_time)
      ? log.use_time
      : null;
  const first =
    log.is_stream && isPositiveNumber(other?.frt) ? other.frt / 1000 : null;
  const speed =
    log.completion_tokens >= 100 && total != null
      ? log.completion_tokens / total
      : null;
  return {
    total,
    first,
    firstTone:
      first == null
        ? 'muted'
        : first < 5
          ? 'healthy'
          : first < 10
            ? 'watch'
            : 'low',
    totalTone:
      total == null
        ? 'muted'
        : speed == null
          ? total < 10
            ? 'healthy'
            : total < 30
              ? 'watch'
              : 'low'
          : speed >= 30
            ? 'healthy'
            : speed >= 15
              ? 'watch'
              : 'low',
  };
}

export function getUsageLogSpeed(log, other) {
  if (
    log.type !== 2 ||
    !isPositiveNumber(log.completion_tokens) ||
    (other?.is_task === true && other.task_sync !== true)
  ) {
    return null;
  }
  const precise = isPositiveNumber(other?.duration_ms);
  const seconds = precise ? other.duration_ms / 1000 : log.use_time;
  if (!isPositiveNumber(seconds)) return null;
  const value = log.completion_tokens / seconds;
  return Number.isFinite(value) ? { value, approximate: !precise } : null;
}

export function summarizeModelPerformance(rows) {
  const totals = { samples: 0, completed: 0, eligible: 0, noRoute: 0 };
  for (const row of rows) {
    totals.samples += row.samples;
    totals.completed += row.completed;
    totals.eligible += row.eligible;
    totals.noRoute += row.no_route;
  }
  return {
    ...totals,
    completionRate:
      totals.eligible > 0 ? totals.completed / totals.eligible : null,
  };
}

const logWindowSeconds = 15 * 60;
const isModelFilter = (name) =>
  typeof name === 'string' &&
  name.length > 0 &&
  name.length <= 128 &&
  !/[\s%\u0000-\u001f\u007f]/u.test(name);
const isChannelID = (value) =>
  Number.isSafeInteger(value) && value >= 0 && value <= 2147483647;
const isTimestamp = (value) =>
  Number.isSafeInteger(value) &&
  value >= 1_000_000_000 &&
  value <= 4_102_444_800;

export function modelPerformanceLogLink(model, channel, from, to) {
  if (
    !isModelFilter(model) ||
    !isChannelID(channel) ||
    !isTimestamp(from) ||
    !isTimestamp(to) ||
    from > to
  ) {
    return null;
  }
  const query = new URLSearchParams({
    performance: '1',
    model,
    from: String(Math.max(from, to - logWindowSeconds)),
    to: String(to),
  });
  if (channel > 0) query.set('channel', String(channel));
  return '/console/log?' + query.toString();
}

export function readPerformanceLogLink(search, admin) {
  if (!admin) return null;
  const q = new URLSearchParams(search);
  if (q.get('performance') !== '1') return null;
  const model = q.get('model');
  const channel = q.get('channel') ?? '0';
  const from = q.get('from') ?? '';
  const to = q.get('to') ?? '';
  if (
    !isModelFilter(model) ||
    ![channel, from, to].every((v) => /^\d+$/.test(v)) ||
    !isChannelID(Number(channel)) ||
    !isTimestamp(Number(from)) ||
    !isTimestamp(Number(to)) ||
    Number(from) > Number(to) ||
    Number(to) - Number(from) > logWindowSeconds
  ) {
    return null;
  }
  return {
    model,
    channel: Number(channel) === 0 ? '' : channel,
    from: Number(from),
    to: Number(to),
  };
}
