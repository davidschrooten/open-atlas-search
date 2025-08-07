// Jest setup file

// Mock AbortController if not available in test environment
if (typeof AbortController === 'undefined') {
  global.AbortController = class AbortController {
    signal = {
      aborted: false,
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
    };
    abort = jest.fn();
  } as any;
}
