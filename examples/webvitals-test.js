import { chromium } from 'k6/x/browser';

export default async function () {
  const browser = chromium.launch({headless: true});
  const context = browser.newContext();
  context.addInitScript(`{
    function print(metric) {
      const m = {
        id: metric.id,
        name: metric.name,
        value: metric.value,
        rating: metric.rating,
        delta: metric.delta,
        numEntries: metric.entries.length,
        navigationType: metric.navigationType,
        url: window.location.href,
      }
      console.log('xk6-browser.web.vital.metric=' + JSON.stringify(m))
    }

    async function load() {
      let {
        onCLS, onFID, onLCP, onFCP, onINP, onTTFB
      } = await import('https://unpkg.com/web-vitals@3?module');

      onCLS(print);
      onFID(print);
      onLCP(print);
  
      onFCP(print);
      onINP(print);
      onTTFB(print);
    }

    load();
  }`);
  const page = context.newPage();

  try {
    await page.goto('https://grafana.com', { waitUntil: 'networkidle' })

    await Promise.all([
      page.waitForNavigation({ waitUntil: 'networkidle' }),
      page.locator('a[href="https://play.grafana.org/"]').click(),
    ]);

    page.screenshot({ path: 'screenshot.png' });
  } finally {
    page.close();
    browser.close();
  }
}
