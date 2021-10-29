import launcher from "k6/x/browser";
import { describe } from './lib/expect.js';

export default function() {
  const browser = launcher.launch('chromium', { headless: false });
  const context = browser.newContext();
  const page = context.newPage();

  describe('Querying', (t) => {
    page.goto('https://test.k6.io/');

    // Find element using CSS selector
    t.expect(page.$('header h1.title').textContent())
      .as('Title with CSS selector').toEqual('test.k6.io');

    // Find element using XPath expression
    t.expect(page.$("//header//h1[@class='title']").textContent())
      .as('Title with XPath selector').toEqual('test.k6.io');
  });

  page.close();
  browser.close();
}
