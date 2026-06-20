import { test } from 'node:test';
import assert from 'node:assert';
import { parseDiagnosticMetadata } from './DiagnosticParser.ts';

test('parseDiagnosticMetadata - valid JSON metadata', () => {
  const validJson = `{
    "generated_at": "2026-06-16T15:23:47.496569+00:00",
    "commit": "00000000",
    "diagnostic_logd": "diagnostic/build-00000000.logd",
    "diagnostic_logd_error": null,
    "chunked": false,
    "chunk_size_bytes": null,
    "total_modules": 1,
    "passed": 0,
    "failed": 1,
    "modules": [
      {
        "name": "frailbox",
        "status": "FAIL",
        "elapsed_seconds": 0,
        "artifact": null,
        "output": "Command not found"
      }
    ]
  }`;

  const result = parseDiagnosticMetadata(validJson);
  assert.strictEqual(result.total_modules, 1);
  assert.strictEqual(result.failed, 1);
  assert.strictEqual(result.modules.length, 1);
  assert.strictEqual(result.modules[0].name, 'frailbox');
  assert.strictEqual(result.modules[0].status, 'FAIL');
});

test('parseDiagnosticMetadata - invalid JSON string', () => {
  assert.throws(() => {
    parseDiagnosticMetadata('{ invalid json }');
  }, /Invalid JSON/);
});

test('parseDiagnosticMetadata - missing modules array', () => {
  const invalidFormatJson = `{
    "generated_at": "2026-06-16T15:23:47.496569+00:00",
    "total_modules": 0
  }`;
  
  assert.throws(() => {
    parseDiagnosticMetadata(invalidFormatJson);
  }, /Invalid format: 'modules' array is missing/);
});

test('parseDiagnosticMetadata - invalid root format', () => {
  assert.throws(() => {
    parseDiagnosticMetadata('"just a string"');
  }, /Invalid format: root must be a JSON object/);
});
