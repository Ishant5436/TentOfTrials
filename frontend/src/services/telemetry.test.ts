import assert from 'node:assert';

const listeners: Record<string, Function[]> = {};

Object.defineProperty(globalThis, 'window', {
  value: {
    addEventListener: (event: string, fn: Function) => {
      if (!listeners[event]) listeners[event] = [];
      listeners[event].push(fn);
    },
    setInterval: () => 123,
    clearInterval: () => {},
    location: { href: 'http://localhost' },
    innerWidth: 1024,
    innerHeight: 768,
  },
  writable: true
});

let beaconsSent = 0;

Object.defineProperty(globalThis, 'navigator', {
  value: {
    sendBeacon: (url: string, data: any) => {
      beaconsSent++;
      return true;
    },
    userAgent: 'test-agent',
    language: 'en-US',
    hardwareConcurrency: 4,
  },
  writable: true
});

Object.defineProperty(globalThis, 'document', {
  value: {
    addEventListener: (event: string, fn: Function) => {
      if (!listeners[event]) listeners[event] = [];
      listeners[event].push(fn);
    },
    title: 'Test',
    referrer: '',
  },
  writable: true
});

Object.defineProperty(globalThis, 'screen', {
  value: {
    width: 1920,
    height: 1080,
  },
  writable: true
});

Object.defineProperty(globalThis, 'sessionStorage', {
  value: {
    getItem: () => null,
    setItem: () => {},
  },
  writable: true
});

// Trigger event helper
function triggerEvent(event: string) {
  if (listeners[event]) {
    for (const fn of listeners[event]) {
      fn();
    }
  }
}

// Now import telemetry after globals are set
import { 
  initTelemetry, 
  track, 
  getTelemetryStats, 
  setTelemetryEnabled,
} from './telemetry.ts';

console.log("Running Telemetry Batch Flush Tests...");

// Initialize
initTelemetry({
  enabled: true,
  endpoint: 'http://localhost/telemetry',
  batchSize: 100,
});

// After initTelemetry, it queues 2 events (session_start and page_view)
let stats = getTelemetryStats();
assert.strictEqual(stats.queued, 2, "initTelemetry should queue 2 initial events");

// Test: partial batches are preserved (no flush if < 100)
for (let i = 0; i < 97; i++) {
  track('custom_event');
}
stats = getTelemetryStats();
assert.strictEqual(stats.queued, 99, "Partial batch (99 events) should be queued");
assert.strictEqual(beaconsSent, 0, "No flush should have occurred yet");

// Test: flush triggers at 100 events
track('custom_event');
stats = getTelemetryStats();
assert.strictEqual(stats.queued, 0, "Queue should be empty after flushing at 100 events");
assert.strictEqual(beaconsSent, 1, "Flush should have occurred exactly once");
assert.strictEqual(stats.sent, 100, "100 events should be marked as sent");

// Test: reset after flush (partial batches preserved again)
track('custom_event');
track('custom_event');
stats = getTelemetryStats();
assert.strictEqual(stats.queued, 2, "Queue should accumulate events again");

// Test: flush triggers on page unload
triggerEvent('beforeunload');
stats = getTelemetryStats();
assert.strictEqual(stats.queued, 0, "Queue should be flushed on beforeunload");
assert.strictEqual(beaconsSent, 2, "Second flush should have occurred on unload");
assert.strictEqual(stats.sent, 102, "102 events sent in total after unload flush");

console.log("All telemetry flush threshold tests passed successfully!");
