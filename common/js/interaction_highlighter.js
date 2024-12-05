(() => {
    function highlightInteractedElement(element, color = 'green') {
        if (!element) return;
        element.style.outline = `4px solid ${color}`;
        element.style.transition = 'outline 0.3s ease-in-out';

        setTimeout(() => {
            element.style.outline = '';
        }, 2000);
    }

    document.addEventListener('click', (event) => {
        const element = event.target;
        highlightInteractedElement(element, '#00FF00');
    });

    document.addEventListener('change', (event) => {
        const element = event.target;
        highlightInteractedElement(element, '#00FF00');
    });
})();
