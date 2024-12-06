(() => {
    window.k6browserInteractionOccurred(JSON.stringify({ event: "domcontentloaded" }))

    document.addEventListener('click', (event) => {
        window.k6browserInteractionOccurred(JSON.stringify({ event: "interact" }))
    });

    document.addEventListener('change', (event) => {
        window.k6browserInteractionOccurred(JSON.stringify({ event: "interact" }))
    });
})();
