import launcher from "k6/x/browser";

export default function() {
  const browser = launcher.launch('chromium', { headless: true });
  const context = browser.newContext();
  const page = context.newPage();
  page.goto('https://test.k6.io/');
  page.screenshot({ path: 'screenshot.png' });
  // TODO: Assert this somehow. Upload as CI artifact or just an external `ls`?
  page.close();
  browser.close();
}
