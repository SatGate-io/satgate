import { defineConfig } from 'vitest/config';

export default defineConfig({
  resolve: {
    alias: {
      'node-fetch': new URL('./vitest.node-fetch.mock.ts', import.meta.url).pathname,
    },
  },
  test: {
    environment: 'node',
    include: ['src/__tests__/**/*.test.ts'],
    globals: true,
    setupFiles: ['./vitest.setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      reportsDirectory: 'coverage',
      include: ['src/**/*.ts'],
      exclude: ['src/**/*.test.ts'],
    },
  },
});
