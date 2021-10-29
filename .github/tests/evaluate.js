import launcher from "k6/x/browser";
import { describe } from './lib/expect.js';

export default function() {
  const browser = launcher.launch('chromium', { headless: true });
  const context = browser.newContext();
  const page = context.newPage();

  describe('Page dimensions', (t) => {
    page.goto('https://test.k6.io/', { waitUntil: 'load' });
    const dimensions = page.evaluate(() => {
      return {
        width: document.documentElement.clientWidth,
        height: document.documentElement.clientHeight,
        deviceScaleFactor: window.devicePixelRatio
      };
    });

    t.expect(dimensions.width).as('Width').toEqual(1265);
    t.expect(dimensions.height).as('Height').toEqual(720);
    t.expect(dimensions.deviceScaleFactor).as('Scale factor').toEqual(1);
  });

  page.close();
  browser.close();
}
