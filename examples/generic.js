import { check } from 'k6';
import { browser } from 'k6/browser';

export const options = {
  scenarios: {
    ui: {
      executor: 'shared-iterations',
      options: {
        browser: {
            type: 'chromium',
        },
      },
    },
  },
  thresholds: {
    checks: ["rate==1.0"]
  }
}

export default async function() {
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    // Goto front page, find login link and click it
    await page.goto('https://test.k6.io/my_messages.php');

    // Enter login credentials and login
    await page.locator('input[name="login"]').type('admin');
    await page.locator('input[name="password"]').type('123');

    // Submit and wait for navigation
    await Promise.all([
      page.waitForNavigation(),
      page.locator('input[type="submit"]').click(),
    ]);

    // Check we've logged in
    const h2 = page.locator('h2');
    const headerOK = await h2.textContent() == 'Welcome, admin!';
    check(headerOK, { 'header': headerOK });
  } finally {
    await page.close();
  }
}
