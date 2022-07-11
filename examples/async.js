import { check } from 'k6';
import launcher from 'k6/x/browser';

export default function() {
  const browser = launcher.launch('chromium', {
    headless: __ENV.XK6_HEADLESS ? true : false,
  });
  const context = browser.newContext();
  const page = context.newPage();

  page.goto('https://test.k6.io/', { waitUntil: 'networkidle' });
  const elem = page.$('a[href="/my_messages.php"]');
  
  elem.asyncClick().then(() => {    
    page.$('input[name="login"]').type('admin');
    page.$('input[name="password"]').type('123');

    // GOMAXPROCS=1 fails.
    // Reason: Two promise goroutines depend on each other.
    return Promise.all([
      page.$('input[type="submit"]').asyncClick(),
      page.asyncWaitForNavigation(),
    ]);
  }).then(() => {
    check(page, {
      'header': page.$('h2').textContent() == 'Welcome, admin!',
    });
    page.close();
    browser.close();
  }).catch(e => {
    console.error("ERROR:", e);
    page.close();
    browser.close();
  });
}
