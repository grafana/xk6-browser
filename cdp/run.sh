#!/bin/bash
#       --single-process \
        # --disable-background-timer-throttling \

'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' \
        --remote-debugging-port=9100 \
        --window-position=100,0 \
        --window-size=640,480 \
        --force-device-scale-factor=1 \
        --enable-features=NetworkService,NetworkServiceInProcess \
        --enable-automation \
        --force-color-profile=srgb \
        --metrics-recording-only \
        --use-mock-keychain \
        --no-first-run \
        --no-sandbox \
        --no-default-browser-check \
        --disable-background-networking \
        --disable-backgrounding-occluded-windows \
        --disable-breakpad \
        --disable-client-side-phishing-detection \
        --disable-component-extensions-with-background-pages \
        --disable-default-apps \
        --disable-dev-shm-usage \
        --disable-extensions \
        --disable-hang-monitor \
        --disable-ipc-flooding-protection \
        --disable-popup-blocking \
        --disable-prompt-on-repost \
        --disable-renderer-backgrounding \
        --disable-sync \
        --disable-component-update \
        --disable-domain-reliability \
        --safebrowsing-dixsable-auto-update \
        --disable-features=TranslateUI,BlinkGenPropertyTrees,ImprovedCookieControls,SameSiteByDefaultCookies,LazyFrameLoading \
        --no-startup-window \
        --password-store=basic

# --log-level=0 \
# --auto-open-devtools-for-tabs \

# --remote-debugging-pipe: more secure than using protocol over a websocket

# https://chromium.googlesource.com/chromium/src/+/HEAD/chrome/common/chrome_switches.cc

# https://peter.sh/experiments/chromium-command-line-switches/

# https://github.com/GoogleChrome/chrome-launcher/blob/master/docs/chrome-flags-for-tools.md

# https://www.microfocus.com/documentation/silk-test/195/en/silktestworkbench-195-help-en/GUID-EF7996FE-B3CA-4B12-9E97-E413DA3D57D2.html