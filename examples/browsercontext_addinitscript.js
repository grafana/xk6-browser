import { check } from 'k6';
import { chromium } from 'k6/x/browser';

export const options = {
  thresholds: {
    checks: ["rate==1.0"]
  }
}

export default async function() {
  const browser = chromium.launch({
    headless: __ENV.XK6_HEADLESS ? true : false,
  });
  const context = browser.newContext();
  await context.addInitScript(
    `(function () {
        document.open();
        document.write("k6Test");
        document.close();
    }());`
  );
  const page = context.newPage();
  try {
    await page.goto('https://test.k6.io/', { waitUntil: 'networkidle' });
    check(page, {
      'body': page.locator("body").innerText() == 'k6Test',
    });
  } finally {
    page.close();
    browser.close();
  }
}
