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

  const inputField = page.locator("#text1");

  if (inputField.isEditable() && inputField.isEnabled()) {
    inputField.type("Hello World!");
  }

  page.close();
  browser.close();
}
