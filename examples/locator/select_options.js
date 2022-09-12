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

  const options = page.locator("#numbers-options");
  options.selectOption("two");

  check(page, {
    'selected option': page.locator("#select-multiple-info-display").innerText() === "Selected: two",
  });

  page.close();
  browser.close();
}
