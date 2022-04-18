import { check, fail } from 'k6';
import launcher from 'k6/x/browser';

export default function() {
  const browser = launcher.launch('chromium', {
    headless: true,
  });
  const context = browser.newContext();
  const page = context.newPage();

  page.evaluate(() => {
    setTimeout(() => {
      const el = document.createElement('h1');
      el.innerHTML = 'Hello';
      document.body.appendChild(el);
    }, 1000);
  });

  page.waitForFunction(() => document.querySelector('h1') !== null, {
    polling: 'mutation',
    timeout: 2000,
  }).then(ok => {
    check(ok, { 'waitForFunction successfully resolved': ok === null });
    page.close();
    browser.close();
  }, err => {
    fail(err);
  });
}
