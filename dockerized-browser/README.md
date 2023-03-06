# Dockerized browser

This directory contains a Dockerfile that can be used to build a docker image with a Chrome browser installed within it. This does not package the k6-browser binary with the browser; that Dockerfile can be found in the root of this project.

## Build

```dockerfile
docker build -t {image-tag} -f Dockerfile-chrome .
```

## Run

```bash
docker run -p {port}:{port}/tcp --cap-drop all --security-opt=no-new-privileges --security-opt \
seccomp=chrome.json --read-only --tmpfs /tmp --tmpfs /home/browser/.cache --tmpfs /home/browser/.pki {image-tag} \
--headless --show-component-extension-options --enable-gpu-rasterization --no-default-browser-check \
--disable-pings --media-router=0 --enable-remote-extensions --load-extension= --no-first-run --window-size=800,600 \
--disable-extensions --disable-renderer-backgrounding --force-color-profile=srgb --hide-scrollbars \
--disable-component-extensions-with-background-pages --no-service-autorun --no-default-browser-check \
--blink-settings=primaryHoverType=2,availableHoverTypes=2,primaryPointerType=4,availablePointerTypes=4 \
--disable-popup-blocking --enable-automation --password-store=basic --disable-background-networking \
--disable-default-apps --disable-hang-monitor --disable-backgrounding-occluded-windows \
--disable-features=ImprovedCookieControls,LazyFrameLoading,GlobalMediaControls,DestroyProfileOnBrowserClose,MediaRouter,AcceptCHFrame \
--use-mock-keychain --enable-features=NetworkService,NetworkServiceInProcess --disable-background-timer-throttling \
--disable-prompt-on-repost --metrics-recording-only --no-startup-window --mute-audio \
--user-data-dir=/tmp/browser/user --disk-cache-dir=/tmp/browser/cache --disable-breakpad --disable-dev-shm-usage \
--disable-ipc-flooding-protection --remote-debugging-port={port} --remote-debugging-address=0.0.0.0
```

## Credits

The `chrome.json` seccomp file was downloaded from https://raw.githubusercontent.com/jfrazelle/dotfiles/master/etc/docker/seccomp/chrome.json.
