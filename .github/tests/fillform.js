import launcher from "k6/x/browser";
import { describe } from './lib/expect.js';
import { sleep } from 'k6';

export default function() {
  const browser = launcher.launch('chromium', {
    headless: true,
	slowMo: '500ms' // slow down by 500ms
  });
  const context = browser.newContext();
  const page = context.newPage();

  describe('Fill form', (t) => {
    // Goto front page, find login link and click it
    page.goto('https://test.k6.io/', { waitUntil: 'networkidle' });
    const elem = page.$('a[href="/my_messages.php"]');
    elem.click();

    // Enter login credentials and login
    page.$('input[name="login"]').type('admin');
    page.$('input[name="password"]').type('123');
    page.$('input[type="submit"]').click();

    sleep(5);
    // Wait for next page to load
    // FIXME: This doesn't work without the above sleep.
    // Same with state 'domcontentloaded'.
    page.waitForLoadState('networkidle');

    t.expect(page.$('h2').textContent()).as('Header').toEqual('Welcome, admin!');
  });

  page.close();
  browser.close();
}
