import launcher from 'k6/x/browser';
import { check } from 'k6';

export default function() {
  const preferredColorScheme = 'dark';

  const browser = launcher.launch('chromium', {
    headless: __ENV.XK6_HEADLESS ? true : false,
  });
  
  const context = browser.newContext({
    // valid values are "light", "dark" or "no-preference"
    colorScheme: preferredColorScheme,
  });
  const page = context.newPage();

  page.goto(
    'https://googlechromelabs.github.io/dark-mode-toggle/demo/',
    { waitUntil: 'load' },
  );

  const colorScheme = page.evaluate(() => {
    return {
      isDarkColorScheme: window.matchMedia('(prefers-color-scheme: dark)').matches
    };
  });
  check(colorScheme, {
    'isDarkColorScheme': cs => cs.isDarkColorScheme
  });

  page.close();
  browser.close();
}
