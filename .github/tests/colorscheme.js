import launcher from "k6/x/browser";
import { describe } from './lib/expect.js';
import { sleep } from "k6";

export default function() {
  const browser = launcher.launch('chromium', {
    colorScheme: 'dark', // Valid values are "light", "dark" or "no-preference"
    headless: true
  });
  const context = browser.newContext();
  const page = context.newPage();

  describe('Color scheme', (t) => {
    page.goto('https://googlechromelabs.github.io/dark-mode-toggle/demo/', { waitUntil: 'load' });

    // FIXME: getAttribute fails with:
    // unable to get node ID of element handle *dom.RequestNodeParams
    // t.expect(page.$('#dark-mode-toggle-3').getAttribute('mode'))
    //   .as('Mode').toEqual('dark');
    // FIXME: colorScheme doesn't actually work... The page loads in light mode
    // regardless of the colorScheme value.
  });

  page.close();
  browser.close();
}
