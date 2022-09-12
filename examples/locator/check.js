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

  check(page, {
    'selector state': checkbox.isChecked() === true,
    'checked message': page.locator("#checkbox-info-display").textContent() === "Thanks for checking the box",
  });

  page.close();
  browser.close();
}
