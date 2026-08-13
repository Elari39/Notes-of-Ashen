import assert from 'node:assert/strict';
import test from 'node:test';

import {
  isValidRAGHistoryRetentionDays,
  RAG_HISTORY_RETENTION_OPTIONS,
} from '../src/types/api.ts';

test('RAG 会话保留期仅允许计划中的固定选项', () => {
  assert.deepEqual(RAG_HISTORY_RETENTION_OPTIONS, [0, 7, 30, 60, 90, 180, 365]);
  for (const value of RAG_HISTORY_RETENTION_OPTIONS) {
    assert.equal(isValidRAGHistoryRetentionDays(value), true);
  }
  for (const value of [-1, 1, 6, 8, 91, 366, 3_650, Number.NaN]) {
    assert.equal(isValidRAGHistoryRetentionDays(value), false);
  }
});
