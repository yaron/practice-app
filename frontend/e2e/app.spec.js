import { test, expect } from '@playwright/test';

// ── Helpers ────────────────────────────────────────────────────────────────

/** Click the spin button and wait until the animation finishes (button re-enables). */
async function spinAndWait(page) {
  const btn = page.getByRole('button', { name: /Draaien/i });
  await btn.click();
  // The button is disabled={spinning} in Svelte; re-enabling means transitionend fired.
  await expect(btn).toBeEnabled({ timeout: 8_000 });
}

/** Submit one session as child 1 and wait for the success panel. */
async function submitChildSession(page) {
  await page.goto('/child/1');
  await expect(page.locator('canvas')).toBeVisible();
  await spinAndWait(page);
  await page.getByRole('button', { name: /Verstuur sessie/i }).click();
  await expect(page.getByText('Goed gedaan')).toBeVisible();
}

/** Log in to the admin panel and wait for the dashboard. */
async function adminLogin(page, username = 'admin', password = 'changeme') {
  await page.goto('/admin');
  await page.getByPlaceholder('Gebruikersnaam').fill(username);
  await page.getByPlaceholder('Wachtwoord').fill(password);
  await page.getByRole('button', { name: /Inloggen/i }).click();
  await expect(page.getByRole('tab', { name: /Openstaand/i })).toBeVisible();
}

// ── Admin view ───────────────────────────────────────────────────────────────

test.describe('admin view', () => {
  test('shows login form at /admin', async ({ page }) => {
    await page.goto('/admin');
    await expect(page.getByPlaceholder('Gebruikersnaam')).toBeVisible();
    await expect(page.getByPlaceholder('Wachtwoord')).toBeVisible();
    await expect(page.getByRole('button', { name: /Inloggen/i })).toBeVisible();
  });

  test('shows error for wrong password', async ({ page }) => {
    await page.goto('/admin');
    await page.getByPlaceholder('Gebruikersnaam').fill('admin');
    await page.getByPlaceholder('Wachtwoord').fill('wrongpassword');
    await page.getByRole('button', { name: /Inloggen/i }).click();
    await expect(page.getByText(/ongeldige|mislukt/i)).toBeVisible();
    // Should still show login form
    await expect(page.getByPlaceholder('Gebruikersnaam')).toBeVisible();
  });

  test('can login and see dashboard tabs', async ({ page }) => {
    await adminLogin(page);
    await expect(page.getByRole('tab', { name: /Openstaand/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /Geschiedenis/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /Gebruikers/i })).toBeVisible();
  });

  test('can approve a pending session', async ({ page }) => {
    await submitChildSession(page);
    await adminLogin(page);
    // First card is the newest session — approve it
    const approveBtn = page.getByRole('button', { name: /Goedkeuren/i }).first();
    await expect(approveBtn).toBeVisible();
    await approveBtn.click();
    // The approved card disappears from the pending list
    await expect(approveBtn).not.toBeVisible({ timeout: 5_000 });
  });

  test('can reject a pending session with a note', async ({ page }) => {
    await submitChildSession(page);
    await adminLogin(page);
    // Open the reject form on the first card
    const rejectToggle = page.getByRole('button', { name: /Afwijzen/i }).first();
    await expect(rejectToggle).toBeVisible();
    await rejectToggle.click();
    // Fill in rejection note and confirm
    await page.getByPlaceholder(/berichtje voor het kind/i).fill('Probeer het morgen opnieuw');
    await page.getByRole('button', { name: /Bevestigen/i }).click();
    // Card disappears after rejection
    await expect(rejectToggle).not.toBeVisible({ timeout: 5_000 });
  });

  test('can switch to history tab and see past sessions', async ({ page }) => {
    await adminLogin(page);
    await page.getByRole('tab', { name: /Geschiedenis/i }).click();
    // History tab renders the session history section (may be empty, but should not error)
    await expect(page.getByRole('tab', { name: /Geschiedenis/i })).toHaveAttribute('aria-selected', 'true');
  });

  test('logout returns to login form', async ({ page }) => {
    await adminLogin(page);
    await page.getByRole('button', { name: /Uitloggen/i }).click();
    await expect(page.getByPlaceholder('Gebruikersnaam')).toBeVisible();
    await expect(page.getByRole('tab', { name: /Openstaand/i })).not.toBeVisible();
  });
});

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
