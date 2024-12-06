(() => {
    let interactionCount = 0; // Counter to track interaction order

    // Function to highlight interacted elements and add a counter
    function highlightInteractedElement(element, color = '#00FF00') {
        if (!element) return;

        interactionCount++; // Increment interaction count

        // Highlight the element with an outline
        element.style.outline = `2px solid ${color}`;

        // Create a new label showing the interaction count
        const label = document.createElement('span');
        label.className = 'interaction-label';
        label.style.position = 'absolute';
        label.style.backgroundColor = 'black'; // Black background
        label.style.color = 'white'; // White text
        label.style.fontSize = '12px';
        label.style.fontWeight = 'bold';
        label.style.padding = '2px 5px';
        label.style.borderRadius = '3px';
        label.style.pointerEvents = 'none';
        label.style.zIndex = '9999';
        label.style.boxShadow = '2px 2px 5px rgba(0, 0, 0, 0.5)'; // Drop shadow

        label.textContent = `${interactionCount}`;

        // Calculate position for the new label
        const rect = element.getBoundingClientRect();

        // Determine how many labels already exist for this element
        const existingLabels = Array.from(document.querySelectorAll('.interaction-label'))
            .filter((lbl) => lbl.dataset.targetElementId === getElementUniqueId(element));
        const offset = existingLabels.length * 20; // Offset each label by 20px vertically

        // Position the label above the element and offset each subsequent label
        label.style.top = `${rect.top + window.scrollY - 20}px`; // 20px above + offset
        label.style.left = `${rect.left + window.scrollX + offset}px`; // Align horizontally with the element

        // Associate the label with the element
        label.dataset.targetElementId = getElementUniqueId(element);

        document.body.appendChild(label);
    }

    // Generate a unique ID for each element
    function getElementUniqueId(element) {
        if (!element.dataset.uniqueId) {
            element.dataset.uniqueId = `uid-${Date.now()}-${Math.random().toString(36).substring(2, 8)}`;
        }
        return element.dataset.uniqueId;
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
