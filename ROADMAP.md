xk6-browser roadmap
===================

xk6-browser is a [k6](https://k6.io/) extension that will become part of the k6 core once it reaches its stability goals. The project adds browser automation support to k6, expanding the testing capabilities of the k6 ecosystem to include real-world user simulation in addition to API/performance testing. This allows web developers to test their applications fully end-to-end in a way that previously wasn't possible with k6 alone.

We consider browser automation to be an important part of web-application testing, and we have big goals in mind for xk6-browser. In the spirit of transparency, we'd like to share our roadmap for the project with the k6 community. We hope that users can plan ahead, trusting the k6 and Grafana's commitment to its success. With that in mind, we'll detail some of our important status updates, our short, mid and long-term goals. Most of these will be worked on concurrently, and reaching them will be a gradual process. The timeframes are also not set in stone, but rather serve as tentative targets that the team is aiming for.

Status updates
----------------

- **Is this production ready?**<br>
   xk6-browser is ready to be used in production. However, be warned that our API is still undergoing a few changes so expect a few breaking changes and bugs 🐞.

- **Is this extension supported in k6 Cloud?**<br>
    No, not yet. We take the security of our customer data very seriously and currently, we are analyzing the implications of running browser instances in the cloud.

- **It doesn't work with my Chromium/Chrome version, why?**<br>
    CDP evolves and there are differences between different versions of Chromium, sometimes quite subtle. The codebase is continuously tested with the two latest major releases of Google Chrome.

- **Are Firefox or WebKit-based browsers supported?**<br>
    Not yet. There are differences in CDP coverage between Chromium, Firefox, and WebKit-based browsers. xk6-browser is initially only targetting Chromium-based browsers.

- **Are all features of Playwright supported?**<br>
    No. Playwright's API is pretty large and some of the functionality only makes sense if it's implemented using async operations: event listening, request interception, waiting for events, etc. This requires the existence of an event loop per VU in k6, which was only [recently added](https://github.com/grafana/k6/issues/882). Most of the current xk6-browser API is synchronous and thus lacks some of the functionality that requires asynchronicity, but we're gradually migrating existing methods to return a `Promise`, and adding new ones that will follow the same API.

    Expect breaking changes during this transition. We'll point them out in the release notes as well as proposed migration plan.

    Note that `async`/`await` is still under development and is not supported in k6 scripts. If you wish to use this syntax you'll have to transform your script beforehand with an updated Babel version. See the [k6-template-es6 project](https://github.com/grafana/k6-template-es6) and [this comment](https://github.com/grafana/k6/issues/779#issuecomment-964027280) for details.

Short-term goals
----------------

- **Transition our API to be async/`Promise` based.**<br>
  At the moment, most of our API is synchronous. This is due to the historical fact that k6 didn't support async behavior because of a missing per-VU event loop.
[This event loop is now available](https://github.com/grafana/k6/pull/2228).
  Async APIs are important for a browser-testing tool, since most browser behavior and [CDP](https://chromedevtools.github.io/devtools-protocol/) (the protocol we use to communicate with the browser) is event-based. We need to expose an async API to implement this missing functionality and reach feature parity with tools like Playwright or Puppeteer.

  *How will we achieve this?*<br>
  By gradually transitioning the relevant methods of current API to an async implementation, and implementing new features as async when needed.

  *Definition of Done*<br>
  When all of currently existing relevant API methods can be executed async.


Mid-term goals
--------------

- **Global availability of browser-based tests in k6 Cloud for all users.**<br>

  *How will we achieve this?*<br>
  The deployment should be optimized and the extension thoroughly tested before making it available to all users. Frontend changes should be done at this point, and usage costs (CPU, RAM, storage) and pricing details should be determined, followed by public announcements of the availability.

  *Definition of Done*<br>
  - The import path for xk6-browser is updated to `k6/browser` and the API is considered to be stable.
  - Tests can be executed using multiple browser VUs.

Long-term goals
---------------

These are goals achievable after a year, and don't have a clear date of delivery yet.

- **Add support for other browsers.**<br>
  Currently, our main focus is supporting Chromium-based browsers. We should expand support to include other browsers as well. The main challenges here will be around CDP and the behavior differences between browsers.

  *How will we achieve this?*<br>
  By testing other browsers and fixing issues as they arise.

  *Definition of Done*<br>
  When a user can choose and execute a test in a non-Chrome browser.


- **Reach rough compatibility with Playwright.**<br>
  Currently, our functionality is limited compared to more mature projects like Playwright. We plan to expand this gradually and reach or exceed the features offered by other browser automation tools, such as screen capture, video recording, downloading, and file uploading.

  *How will we achieve this?*<br>
  By prioritizing new features to add based on API importance and user feedback. Once our standalone and Cloud users are able to execute feasible complex scenarios, our main focus will be to add more missing features and close the current feature gap.

  *Definition of Done*<br>
  When we implement the selected scope of functionality found in Playwright that makes sense for xk6-browser. This is intentionally vague at the moment, and we'll refine it as we make progress.

- **Support the usage of k6 browser for end-to-end tests.**<br>

  *How will we achieve this?*<br>
  Stabilize API, provide the ability to catch and expose JS errors and exceptions that our users can track.

  *Definition of Done*<br>
  - There are no planned changes to the existing API method signatures.
  - xk6-browser provides an API to catch JS exceptions and expose them as metrics.

- **Full integration with other Grafana products.*

  *How will we achieve this?*<br>
  _TBD_
  
  *Definition of Done*<br>
  - Users are able to seamlessly navigate between frontend-facing Grafana products.
  - The data collected from running browser tests can be accessed in products like Faro, Synthetic Monitoring, etc.
  - The data collected by other frontend-facing Grafana products can be used to author and run browser-based tests in Grafana k6.
