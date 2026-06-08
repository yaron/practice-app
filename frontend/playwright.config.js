import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  retries: process.env.CI ? 2 : 0,

  use: {
    baseURL: 'http://localhost:5174',
    trace: 'on-first-retry',
  },

  webServer: [
    {
      // Fresh DB on every CI run; reuse if already running in dev
      command:
        'rm -f /tmp/vq-e2e.db && (cd ../backend && DB_PATH=/tmp/vq-e2e.db PORT=18081 GIN_MODE=release go run .)',
      port: 18081,
      timeout: 60_000, // allow time for Go compilation
      reuseExistingServer: !process.env.CI,
    },
    {
      // Port 5174 avoids colliding with the regular dev server on 5173
      command: 'npm run dev -- --mode test --port 5174',
      port: 5174,
      reuseExistingServer: !process.env.CI,
    },
  ],
});
