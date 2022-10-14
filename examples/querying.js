import { check } from 'k6';
import { chromium } from 'k6/x/browser';

export const options = {
  thresholds: {
    checks: ["rate==1.0"]
  }
}

export default function() {
  const browser = chromium.launch({
    headless: __ENV.XK6_HEADLESS ? true : false,
  });
  const context = browser.newContext();
  const page = context.newPage();

  page.goto('https://test.k6.io/').then(() => {
    check(page, {
      'Title with CSS selector':
        p => p.$('header h1.title').textContent() == 'test.k6.io',
      'Title with XPath selector':
        p => p.$(`//header//h1[@class="title"]`).textContent() == 'test.k6.io',
    });
  }).finally(() => {
    page.close();
    browser.close();
  });
}
