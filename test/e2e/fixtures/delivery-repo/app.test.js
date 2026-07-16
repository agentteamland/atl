// `node --test` unit test for the fixture app. The tester role runs this to
// produce the green evidence the tech-lead reads back before merge.
const { test } = require('node:test');
const assert = require('node:assert');
const { add } = require('./app');

test('add sums two numbers', () => {
  assert.strictEqual(add(2, 3), 5);
});
