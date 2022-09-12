import { check } from 'k6';
import { chromium } from 'k6/x/browser';

export default function () {
  const browser = chromium.launch({
    headless: __ENV.XK6_HEADLESS ? true : false,
  });
  const context = browser.newContext();
  const page = context.newPage();
  page.goto("https://test.k6.io/browser.php", {
    waitUntil: "networkidle",
  });

  const disabledField = page.locator("#input-text-disabled");
  disabledField.waitFor({
    state: "visible"
  });

  check(page, {
    'selector state': disabledField.isVisible() === true
  })

  page.close();
  browser.close();
}
