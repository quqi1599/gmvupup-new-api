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

import { expect, test } from 'bun:test';
import {
  getUsageLogSpeed,
  getUsageLogTiming,
  modelPerformanceLogLink,
  readPerformanceLogLink,
  summarizeModelPerformance,
} from './model-performance';

test('uses precise duration first and flags legacy seconds as approximate', () => {
  const row = { type: 2, completion_tokens: 120, use_time: 3 };
  expect(getUsageLogSpeed(row, { duration_ms: 2400 })).toEqual({
    value: 50,
    approximate: false,
  });
  expect(getUsageLogSpeed(row, {})).toEqual({ value: 40, approximate: true });
  expect(getUsageLogSpeed({ ...row, use_time: 0 }, {})).toBeNull();
  expect(getUsageLogSpeed({ ...row, type: 5 }, {})).toBeNull();
  expect(getUsageLogSpeed(row, { is_task: true })).toBeNull();
  expect(getUsageLogSpeed({ ...row, completion_tokens: 0 }, {})).toBeNull();
});

test('keeps timing precision and shows missing first events without implying zero', () => {
  const row = { is_stream: true, use_time: 15, completion_tokens: 150 };
  expect(getUsageLogTiming(row, { frt: 5000, duration_ms: 6000 })).toEqual({
    total: 6,
    first: 5,
    firstTone: 'watch',
    totalTone: 'watch',
  });
  expect(getUsageLogTiming(row, {})).toEqual({
    total: 15,
    first: null,
    firstTone: 'muted',
    totalTone: 'low',
  });
  expect(
    getUsageLogTiming({ ...row, is_stream: false, use_time: 0 }, { frt: 120 }),
  ).toEqual({
    total: null,
    first: null,
    firstTone: 'muted',
    totalTone: 'muted',
  });
});

test('combines eligible requests rather than averaging per-model rates', () => {
  const summary = summarizeModelPerformance([
    { samples: 100, completed: 80, eligible: 90, no_route: 2 },
    { samples: 15, completed: 5, eligible: 10, no_route: 4 },
    { samples: 0, completed: 0, eligible: 0, no_route: 0 },
  ]);
  expect(summary).toEqual({
    samples: 115,
    completed: 85,
    eligible: 100,
    noRoute: 6,
    completionRate: 0.85,
  });
  expect(summarizeModelPerformance([]).completionRate).toBeNull();
  expect(
    summarizeModelPerformance([
      { samples: 2, completed: 0, eligible: 0, no_route: 0 },
    ]).completionRate,
  ).toBeNull();
});

test('drill-down keeps encoded model/channel, caps duration and is admin-only', () => {
  const link = modelPerformanceLogLink(
    'example/model+v1',
    168,
    1795000000,
    1795020000,
  );
  const search = new URL(link, 'http://localhost').search;
  expect(readPerformanceLogLink(search, true)).toEqual({
    model: 'example/model+v1',
    channel: '168',
    from: 1795019100,
    to: 1795020000,
  });
  expect(readPerformanceLogLink(search, false)).toBeNull();
  expect(
    readPerformanceLogLink(search.replace('1795019100', '1795000000'), true),
  ).toBeNull();
  expect(
    modelPerformanceLogLink('%all', 168, 1795019100, 1795020000),
  ).toBeNull();
  expect(
    readPerformanceLogLink(
      search.replace('1795020000', '9007199254740991'),
      true,
    ),
  ).toBeNull();
});
