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

  const checkbox = page.locator("#checkbox1");
  checkbox.check();
  checkbox.uncheck();

  check(page, {
    'unchecked message': page.locator("#checkbox-info-display").textContent() === "You've just unchecked the box",
  });

  page.close();
  browser.close();
}
