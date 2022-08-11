import { check } from 'k6';
import { chromium } from 'k6/x/browser';

export default function() {
  const browser = chromium.launch({
    headless: __ENV.XK6_HEADLESS ? true : false,
  });

  check(browser, {
    'should be connected after launch': browser.isConnected(),
  });

  const handlerCalled = Symbol();

  browser.on('disconnected').then((val) => {
    // The promise resolve/success handler
    check(browser, {
      'should be disconnected on event': !browser.isConnected(),
    });
    return handlerCalled;
  }, (val) => {
    // The promise reject/failure handler
    console.error(`promise rejected: ${val}`);
  }).then((val) => {
    check(val, {
      'the browser.on success handler should be called': val === handlerCalled,
    });
  });

  check(browser, {
    'should be connected before ending iteration': browser.isConnected(),
  });

  // Disconnect from the browser instance, and trigger the Promise chain to
  // resolve.
  browser.close();
}
