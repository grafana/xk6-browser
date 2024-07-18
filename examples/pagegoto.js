import { browser } from "k6/browser";
import { sleep } from "k6";

export const options = {
  scenarios: {
    ui: {
      executor: "shared-iterations",
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
  const page = await browser.newPage();
  console.log("hello, world!");

  await page.goto("https://grafana.com/");

  sleep(1);
}
