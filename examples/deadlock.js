import { browser } from 'k6/x/browser';
import { check } from 'k6';

export const options = {
    ext: {
        loadimpact: {
            distribution: {
                frankfurt: { loadZone: 'amazon:de:frankfurt', percent: 100 },
            }
        }
    },
    scenarios: {
        browser: {
            executor: 'shared-iterations',
            options: {
                browser: {
                    type: 'chromium',
                },
            },
            vus: 1,
            iterations: 1,
        },
    },
    thresholds: {
        checks: ["rate==1.0"]
    }
}

export default async function () {
    const page = browser.newPage();
    try {
        await page.goto("http://localhost:81")
        await Promise.all([
            page.click('a[href="/redirect"]'),
            page.waitForNavigation({
                waitUntil: 'domcontentloaded',
                // timeout: 1000,
            }),
        ]);
        // await page.goto("https://mmc-b-uat.media-server.com/mmc/p/rd4b29qo"); //data.playerPath); //'https://mmc-b-uat.media-server.com/mmc/console/client/');
        // page.locator('input[name="firstname"]').type('Tester');
        // page.locator('input[name="lastname"]').type('Tester');
        // page.locator('input[name="email"]').type('tester@tester.com');
        // const submitButton = page.locator('//button[text()="Submit"]')
        // await Promise.all([page.waitForNavigation({
        //     waitUntil: 'domcontentloaded',
        //     timeout: 750,
        // }), submitButton.click()]);
        // check(page, {
        //     'header': p => p.locator('header').textContent().includes("M6 ONLY"),
        // });
    } finally {
        page.close();
    }
}