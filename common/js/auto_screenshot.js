(() => {
    document.addEventListener("DOMContentLoaded", (event) => {
        window.k6browserInteractionOccurred(JSON.stringify({ load: true }))
    });

    document.addEventListener('click', (event) => {
        window.k6browserInteractionOccurred(JSON.stringify({ interact: true }))
    });

    document.addEventListener('change', (event) => {
        window.k6browserInteractionOccurred(JSON.stringify({ interact: true }))
    });
})();
