import test from 'node:test';
import assert from 'node:assert/strict';
import { cn } from './utils.ts';

test('cn utility', async (t) => {
  await t.test('merges class names', () => {
    assert.strictEqual(cn('foo', 'bar'), 'foo bar');
  });

  await t.test('handles conditional classes', () => {
    assert.strictEqual(cn('foo', true && 'bar', false && 'baz'), 'foo bar');
    // @ts-ignore
    assert.strictEqual(cn('foo', null, undefined, 'bar'), 'foo bar');
  });

  await t.test('handles object inputs', () => {
    assert.strictEqual(cn({ foo: true, bar: false, baz: true }), 'foo baz');
  });

  await t.test('handles array inputs', () => {
    assert.strictEqual(cn(['foo', 'bar']), 'foo bar');
    // @ts-ignore
    assert.strictEqual(cn(['foo', ['bar', 'baz']]), 'foo bar baz');
  });

  await t.test('merges tailwind classes correctly', () => {
    // Note: In the local test environment with mock tailwind-merge,
    // it handles basic conflict resolution where the last class for a property wins.
    assert.strictEqual(cn('px-2 py-2', 'p-4'), 'p-4');
    assert.strictEqual(cn('text-red-500', 'text-blue-500'), 'text-blue-500');
  });

  await t.test('handles complex nested inputs', () => {
    const result = cn('base', { conditional: true, hidden: false }, ['array1', ['array2']], null, 'end');
    assert.ok(result.includes('base'));
    assert.ok(result.includes('conditional'));
    assert.ok(!result.includes('hidden'));
    assert.ok(result.includes('array1'));
    assert.ok(result.includes('array2'));
    assert.ok(result.includes('end'));
  });
});
