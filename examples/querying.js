import launcher from 'k6/x/browser';
import { check } from 'k6';

export default function() {
  const browser = launcher.launch('chromium', {
    headless: __ENV.XK6_HEADLESS ? true : false,
  });
  const context = browser.newContext();
  const page = context.newPage();

  page.goto('https://test.k6.io/');

  check(page, {
    'Title with CSS selector':
      p => p.$('header h1.title').textContent() == 'test.k6.io',
    'Title with XPath selector':
      p => p.$(`//header//h1[@class="title"]`).textContent() == 'test.k6.io',
  });

  page.close();
  browser.close();
}
