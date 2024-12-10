(() => {
    // Check if the script has already been executed
    if (window.__k6BrowserInteractionScriptRan) {
        return;
    }

    window.__k6BrowserInteractionScriptRan = true;

    const handledTargets = new Set();

    function once(element, event) {
        // Check if the event target is already handled
        if (handledTargets.has(element)) {
            return; // Ignore if already handled
        }

        // Add the target to the Set
        handledTargets.add(element);

        // Optional: Remove the element from the Set after a timeout (if necessary)
        setTimeout(() => {
            handledTargets.delete(element);
        }, 500); // Allow reprocessing after 500ms

        window.k6browserInteractionOccurred(JSON.stringify({ event: event }))
    }

    window.k6browserInteractionOccurred(JSON.stringify({ event: "domcontentloaded" }))

    document.addEventListener('click', (event) => {
        once(event.target, "interact");
    });

    document.addEventListener('select', (event) => {
        once(event.target, "interact");
    });

    document.addEventListener('input', (event) => {
        once(event.target, "interact");
    });
})();
