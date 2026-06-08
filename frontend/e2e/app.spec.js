import { test, expect } from '@playwright/test';

// ── Helpers ────────────────────────────────────────────────────────────────

/** Click the spin button and wait until the animation finishes (button re-enables). */
async function spinAndWait(page) {
  const btn = page.getByRole('button', { name: /Draaien/i });
  await btn.click();
  // The button is disabled={spinning} in Svelte; re-enabling means transitionend fired.
  await expect(btn).toBeEnabled({ timeout: 8_000 });
}

// ── Invalid path ────────────────────────────────────────────────────────────

test('shows error when no child id in path', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByText('Ongeldig adres')).toBeVisible();
});

test('shows error for non-numeric child id', async ({ page }) => {
  await page.goto('/child/abc');
  await expect(page.getByText('Ongeldig adres')).toBeVisible();
});

// ── Child view ───────────────────────────────────────────────────────────────

test.describe('child view at /child/1', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/child/1');
    // Canvas renders once options are fetched — wait for it before each test
    await expect(page.locator('canvas')).toBeVisible();
  });

  test('shows the wheel and spin button', async ({ page }) => {
    await expect(page.getByText('Viool Quest')).toBeVisible();
    await expect(page.getByRole('button', { name: /Draaien/i })).toBeEnabled();
    // Session list is hidden until the first spin
    await expect(page.getByText('Jouw oefening')).not.toBeVisible();
  });

  test('spinning adds a task to the session list', async ({ page }) => {
    await spinAndWait(page);
    await expect(page.getByText('Jouw oefening')).toBeVisible();
    await expect(page.getByRole('button', { name: /Verstuur sessie/i })).toBeVisible();
  });

  test('multiple spins accumulate tasks', async ({ page }) => {
    await spinAndWait(page);
    await spinAndWait(page);
    // Should show "2 taken"
    await expect(page.getByText('2 taken')).toBeVisible();
  });

  test('submitting session shows success panel', async ({ page }) => {
    await spinAndWait(page);
    await page.getByRole('button', { name: /Verstuur sessie/i }).click();
    await expect(page.getByText('Goed gedaan')).toBeVisible();
    await expect(page.getByText(/verstuurd/i)).toBeVisible();
    // Wheel and tracker should be gone
    await expect(page.locator('canvas')).not.toBeVisible();
  });

  test('new session resets back to the wheel', async ({ page }) => {
    await spinAndWait(page);
    await page.getByRole('button', { name: /Verstuur sessie/i }).click();
    await expect(page.getByText('Goed gedaan')).toBeVisible();

    await page.getByRole('button', { name: /Nieuwe sessie/i }).click();

    await expect(page.locator('canvas')).toBeVisible();
    await expect(page.getByText('Jouw oefening')).not.toBeVisible();
    await expect(page.getByRole('button', { name: /Draaien/i })).toBeEnabled();
  });
});
