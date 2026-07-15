import assert from 'node:assert/strict';
import test from 'node:test';
import { getReadingStats } from '../src/utils/readingStats.ts';

test('reading statistics match backend fixtures', () => {
  assert.deepEqual(getReadingStats(''), { wordCount: 0, readingTimeMinutes: 0 });
  assert.deepEqual(getReadingStats('这是中文。'), { wordCount: 4, readingTimeMinutes: 1 });
  assert.deepEqual(getReadingStats('hello world 2026'), { wordCount: 3, readingTimeMinutes: 1 });
  assert.deepEqual(getReadingStats('# 标题\n[OpenAI](https://openai.com) 与 `Go`'), { wordCount: 5, readingTimeMinutes: 1 });
  assert.deepEqual(getReadingStats('……！'), { wordCount: 0, readingTimeMinutes: 1 });
});
