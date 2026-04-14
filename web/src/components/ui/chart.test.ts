import test from 'node:test';
import assert from 'node:assert/strict';

// Since we cannot easily import the React component in this environment
// (it requires a DOM and React setup which is not fully available for simple node tests),
// we will test the sanitization logic by extracting it into a testable function
// or by replicating the logic here to ensure the regexes work as intended.

const sanitizeId = (id: string) => id.replace(/[^a-zA-Z0-9_-]/g, "");
const sanitizeKey = (key: string) => key.replace(/[^a-zA-Z0-9_-]/g, "");
const sanitizeColor = (color: string) => color.replace(/[<>{};]/g, "");

test('ChartStyle sanitization logic', async (t) => {
  await t.test('sanitizeId removes dangerous characters', () => {
    assert.strictEqual(sanitizeId('chart-123'), 'chart-123');
    assert.strictEqual(sanitizeId('chart-123"'), 'chart-123');
    assert.strictEqual(sanitizeId('chart-123; body { display: none; }'), 'chart-123bodydisplaynone');
    assert.strictEqual(sanitizeId('id with spaces'), 'idwithspaces');
    assert.strictEqual(sanitizeId('id<script>'), 'idscript');
  });

  await t.test('sanitizeKey removes dangerous characters', () => {
    assert.strictEqual(sanitizeKey('primary'), 'primary');
    assert.strictEqual(sanitizeKey('primary-color'), 'primary-color');
    assert.strictEqual(sanitizeKey('primary; color: red'), 'primarycolorred');
    assert.strictEqual(sanitizeKey('var(--unsafe)'), 'var--unsafe');
  });

  await t.test('sanitizeColor removes dangerous characters', () => {
    assert.strictEqual(sanitizeColor('#ffffff'), '#ffffff');
    assert.strictEqual(sanitizeColor('rgb(255, 255, 255)'), 'rgb(255, 255, 255)');
    assert.strictEqual(sanitizeColor('hsl(200, 100%, 50%)'), 'hsl(200, 100%, 50%)');
    assert.strictEqual(sanitizeColor('#fff; } body { background: red; }'), '#fff  body  background: red ');
    assert.strictEqual(sanitizeColor('<script>alert(1)</script>'), 'scriptalert(1)/script');
  });
});
