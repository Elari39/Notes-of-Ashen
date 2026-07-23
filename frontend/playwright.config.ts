import { dirname, join, resolve } from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import { defineConfig, devices } from '@playwright/test';

const frontendRoot = dirname(fileURLToPath(import.meta.url));
const artifactDirectory = process.env.E2E_ARTIFACT_DIR?.trim();
const artifactRoot = artifactDirectory
  ? resolve(artifactDirectory)
  : frontendRoot;

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.spec.ts',
  fullyParallel: false,
  workers: 1,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  timeout: 90_000,
  expect: {
    timeout: 15_000,
  },
  outputDir: join(artifactRoot, 'test-results'),
  reporter: [
    ['list'],
    ['html', {
      outputFolder: join(artifactRoot, 'playwright-report'),
      open: 'never',
    }],
  ],
  use: {
    baseURL: process.env.E2E_WEB_BASE_URL?.trim() || 'http://127.0.0.1:1271',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'mobile-chromium',
      use: { ...devices['Pixel 7'] },
    },
    {
      name: 'mobile-webkit',
      use: { ...devices['iPhone 13'], browserName: 'webkit' },
    },
  ],
});
