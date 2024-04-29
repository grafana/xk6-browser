import { check } from "k6";
import { browser } from "k6/x/browser";

export const options = {
  scenarios: {
    ui: {
      executor: "shared-iterations",
      options: {
        browser: {
          type: "chromium",
        },
      },
    },
  },
  thresholds: {
    checks: ["rate==1.0"],
  },
};

export default async function () {
  const page = browser.newPage();

  page.setContent(`
    <html>
      <head>
        <style></style>
      </head>
      <body>
        <div id="drag-source" draggable="true">Drag me!</div>
        <div id="drop-target">Drop here!</div>

        <script>
          const dragSource = document.getElementById('drag-source');
          const dropTarget = document.getElementById('drop-target');

          dragSource.addEventListener('dragstart', (event) => {
            console.log('dragstart');

            event.dataTransfer.setData('text/plain', 'Something dropped!');
          });

          dropTarget.addEventListener('dragover', (event) => {
            console.log("dragover");

            event.preventDefault();
          });

          dropTarget.addEventListener('drop', (event) => {
            console.log("drop");

            event.preventDefault();
            const data = event.dataTransfer.getData('text/plain');
            event.target.innerText = data;
          });
        </script>
      </body>
    </html>
  `);

  await page.dragAndDrop("#drag-source", "#drop-target");

  const dropEl = await page.waitForSelector("#drop-target");

  check(dropEl, {
    "source was dropped on target": (e) =>
      e.innerText() === "Something dropped!",
  });

  page.close();
}
