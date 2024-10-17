function faroInit () {
    // Create a script tag for loading the library
    var script = document.createElement('script');

    // Initialize the Web SDK at the onLoad event of the script element so it is called when the library is loaded.
    script.onload = () => {
      window.GrafanaFaroWebSdk.initializeFaro({
        // Mandatory, the URL of the Grafana Cloud collector with embedded application key.
        // Copy from the configuration page of your application in Grafana.
        url: 'http://faro-collector-us-central-0.grafana.net/collect/{app-key}',

        // Mandatory, the identification label(s) of your application
        app: {
          name: 'my-app',
          version: '1.0.0', // Optional, but recommended
        },
      });
    };

    // Set the source of the script tag to the CDN
    script.src = 'https://unpkg.com/@grafana/faro-web-sdk@^1.0.0-beta/dist/bundle/faro-web-sdk.iife.js';

    // Append the script tag to the head of the HTML document
    document.head.appendChild(script);
}

faroInit();
