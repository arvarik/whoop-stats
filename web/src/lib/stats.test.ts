import test from 'node:test';
import assert from 'node:assert/strict';
import { computeAvg, computeStdDev, percentChange } from './stats.ts';

test('stats utilities', async (t) => {
  await t.test('computeAvg', async (t) => {
    await t.test('computes average of positive numbers', () => {
      assert.strictEqual(computeAvg([1, 2, 3, 4, 5]), 3);
    });

    await t.test('computes average of negative numbers', () => {
      assert.strictEqual(computeAvg([-1, -2, -3, -4, -5]), -3);
    });

    await t.test('computes average of mixed numbers', () => {
      assert.strictEqual(computeAvg([-2, -1, 0, 1, 2]), 0);
    });

    await t.test('returns null for empty array', () => {
      assert.strictEqual(computeAvg([]), null);
    });

    await t.test('handles single-element array', () => {
      assert.strictEqual(computeAvg([42]), 42);
    });
  });

  await t.test('computeStdDev', async (t) => {
    await t.test('computes standard deviation of standard numeric array', () => {
      const result = computeStdDev([1, 2, 3, 4, 5]);
      assert.ok(result !== null);
      // Math.sqrt(2) = 1.4142135623730951
      assert.ok(Math.abs(result - 1.4142135623730951) < 1e-10);
    });

    await t.test('returns null for empty array', () => {
      assert.strictEqual(computeStdDev([]), null);
    });

    await t.test('returns null for array with one element', () => {
      assert.strictEqual(computeStdDev([42]), null);
    });

    await t.test('returns 0 for array with identical elements', () => {
      assert.strictEqual(computeStdDev([5, 5, 5, 5, 5]), 0);
    });
  });

  await t.test('percentChange', async (t) => {
    await t.test('computes positive change correctly', () => {
      assert.strictEqual(percentChange(10, 5), 100);
    });

    await t.test('computes negative change correctly', () => {
      assert.strictEqual(percentChange(5, 10), -50);
    });

    await t.test('computes zero change correctly', () => {
      assert.strictEqual(percentChange(5, 5), 0);
    });

    await t.test('returns null when previous value is 0', () => {
      assert.strictEqual(percentChange(10, 0), null);
    });

    await t.test('computes correctly when current value is 0', () => {
      assert.strictEqual(percentChange(0, 10), -100);
    });
  });
});
