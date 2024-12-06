import { check } from 'https://jslib.k6.io/k6-utils/1.5.0/index.js';
import { browser } from 'k6/x/browser/async';

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
  const page = await browser.newPage();

  try {
    // Goto front page, find login link and click it
    await page.goto('https://test.k6.io/', { waitUntil: 'networkidle' });

    await page.locator('a[href="/my_messages.php"]').click()
    await page.waitForSelector('input[name="login"]')

    // Enter login credentials and login
    await page.locator('input[name="login"]').type('admin');
    await page.locator('input[name="password"]').type("123");

    // We expect the form submission to trigger a navigation, so to prevent a
    // race condition, setup a waiter concurrently while waiting for the click
    // to resolve.
    await Promise.all([
      page.waitForNavigation(), // TODO: Removing Promise.all should work
      page.locator('input[type="submit"]').click(),
    ]);

    await check(page.locator('h2'), {
      'header': async lo => {
        return await lo.textContent() == 'Welcome, admin!'
      }
    });

    // Check whether we receive cookies from the logged site.
    await check(browser.context(), {
      'session cookie is set': async ctx => {
        const cookies = await ctx.cookies();
        return cookies.find(c => c.name == 'sid') !== undefined;
      }
    });
  } finally {
    await page.close();
  }
}
