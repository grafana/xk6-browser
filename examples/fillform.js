import { check, sleep } from 'k6';
import { browser } from 'k6/x/browser';

export const options = {
  scenarios: {
    ui: {
      executor: "shared-iterations",
      vus: 1,
      iterations: 3,
      maxDuration: "5m",
      options: {
        browser: {
          type: "chromium",
        },
      },
    },
  },
  thresholds: {
    checks: ["rate==1.0"],
  },
};

export default async function () {
  const loggy = (msg) => {
    const vuid = (__VU + "").padStart(2, "0");
    const iterid = (__ITER + "").padStart(4, "0");
    const dtext = `VU: ${vuid} ITER: ${iterid}: `;
    console.log("debug", dtext, msg);
  };

  loggy("START: debug newContext");
  const context = browser.newContext();
  loggy("END  : newContext");
  context.setDefaultTimeout(30000);
  loggy("START: newPage");
  const page = context.newPage();
  loggy("END  : newPage");

  sleep(5)

  try {
    // Goto front page, find login link and click it
    loggy("START: pageGoto");
    await page.goto("https://test.k6.io/", { waitUntil: "networkidle" });
    loggy("END  : pageGoto");
    await Promise.all([
      page.waitForNavigation(),
      page.locator('a[href="/my_messages.php"]').click(),
    ]);
    loggy("END  : waitForNavigation");
    // Enter login credentials and login
    loggy("START: input");
    page.locator('input[name="login"]').type("admin");
    page.locator('input[name="password"]').type("123");
    loggy("END  : input");
    loggy("START: navigating");
    // We expect the form submission to trigger a navigation, so to prevent a
    // race condition, setup a waiter concurrently while waiting for the click
    // to resolve.
    await Promise.all([
      page.waitForNavigation(),
      page.locator('input[type="submit"]').click(),
    ]);
    loggy("END  : navigating");
    loggy("START: checking");
    check(page, {
      header: (p) => p.locator("h2").textContent() == "Welcome, admin!",
    });
    loggy("END  : checking");
  } finally {
    loggy("START: closing");
    page.close();
    loggy("END  : closing");
  }
}