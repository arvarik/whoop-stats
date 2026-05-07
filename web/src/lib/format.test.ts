import test from 'node:test';
import assert from 'node:assert/strict';
import {
  formatDuration,
  formatShortDate,
  formatFullDate,
  formatTime,
  getRecoveryColor,
  getRecoveryColorValue,
  getRecoveryLabel,
  formatDistance,
  formatCalories,
  kjToCal
} from './format.ts';

// Set timezone to UTC for deterministic date/time testing
process.env.TZ = 'UTC';

test('format utilities', async (t) => {
  await t.test('formatDuration', () => {
    assert.strictEqual(formatDuration(0), '--');
    assert.strictEqual(formatDuration(-1000), '--');
    // @ts-ignore
    assert.strictEqual(formatDuration(null), '--');
    // @ts-ignore
    assert.strictEqual(formatDuration(undefined), '--');

    // 30m = 30 * 60 * 1000 = 1800000 ms
    assert.strictEqual(formatDuration(1800000), '30m');

    // 1h = 60 * 60 * 1000 = 3600000 ms
    assert.strictEqual(formatDuration(3600000), '1h 0m');

    // 6h 32m = (6 * 3600000) + (32 * 60000) = 21600000 + 1920000 = 23520000 ms
    assert.strictEqual(formatDuration(23520000), '6h 32m');
  });

  await t.test('date formatting', () => {
    const dateStr = '2025-03-10T12:00:00Z';

    // These might be locale dependent, but usually 'en-US' is available in Node.js
    // formatShortDate uses month: "short", day: "numeric"
    assert.strictEqual(formatShortDate(dateStr), 'Mar 10');

    // formatFullDate uses month: "short", day: "numeric", year: "numeric"
    assert.strictEqual(formatFullDate(dateStr), 'Mar 10, 2025');

    // formatTime uses hour: "numeric", minute: "2-digit"
    // In UTC, 12:00:00Z should be 12:00 PM
    assert.strictEqual(formatTime(dateStr), '12:00 PM');
  });

  await t.test('recovery formatting', () => {
    // Green range (>= 66)
    assert.strictEqual(getRecoveryColor(66), 'green');
    assert.strictEqual(getRecoveryColorValue(66), 'var(--color-recovery-green)');
    assert.strictEqual(getRecoveryLabel(66), 'Primed to perform');

    assert.strictEqual(getRecoveryColor(100), 'green');

    // Yellow range (34 - 65)
    assert.strictEqual(getRecoveryColor(65), 'yellow');
    assert.strictEqual(getRecoveryColorValue(65), 'var(--color-recovery-yellow)');
    assert.strictEqual(getRecoveryLabel(65), 'Moderate readiness');

    assert.strictEqual(getRecoveryColor(34), 'yellow');

    // Red range (< 34)
    assert.strictEqual(getRecoveryColor(33), 'red');
    assert.strictEqual(getRecoveryColorValue(33), 'var(--color-recovery-red)');
    assert.strictEqual(getRecoveryLabel(33), 'Take it easy');

    assert.strictEqual(getRecoveryColor(0), 'red');
  });

  await t.test('formatDistance', () => {
    assert.strictEqual(formatDistance(0), '--');
    assert.strictEqual(formatDistance(-5), '--');

    assert.strictEqual(formatDistance(500), '500 m');
    assert.strictEqual(formatDistance(1000), '1.0 km');
    assert.strictEqual(formatDistance(2500), '2.5 km');
    assert.strictEqual(formatDistance(1234), '1.2 km');
  });

  await t.test('calorie formatting', () => {
    assert.strictEqual(formatCalories(0), '--');
    // @ts-ignore
    assert.strictEqual(formatCalories(null), '--');

    // 1000 kJ * 0.239006 = 239.006 -> 239 Cal
    assert.strictEqual(formatCalories(1000), '239 Cal');
    assert.strictEqual(kjToCal(1000), 239);

    // 5000 kJ * 0.239006 = 1195.03 -> 1,195 Cal
    assert.strictEqual(formatCalories(5000), '1,195 Cal');
  });
});
