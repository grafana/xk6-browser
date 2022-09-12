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

  check(page, {
    'options innertext': page.locator("#numbers-options").innerText() === "Zero\nOne\nTwo\nThree\nFour\nFive",
  });

  page.close();
  browser.close();
}
