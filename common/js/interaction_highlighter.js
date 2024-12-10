(() => {
    let interactionCount = 0; // Counter to track interaction order

    const handledTargets = new Set();
    const labelMap = new Map(); // Map to associate elements with their labels

    // Function to highlight interacted elements and add a counter
    function highlightInteractedElement(element, color = '#00FF00') {
        if (!element) return;

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

        // Get the labels already associated with the element
        const labels = labelMap.get(element) || [];
        const offset = labels.length * 20; // Offset each label by 20px vertically

        // Calculate position for the new label
        const rect = element.getBoundingClientRect();
        label.style.top = `${rect.top + window.scrollY - 20 - offset}px`; // Place above and offset
        label.style.left = `${rect.left + window.scrollX}px`; // Align horizontally

        // Add the label to the Map
        labels.push(label);
        labelMap.set(element, labels);

        document.body.appendChild(label);
    }

    // Sync label positions to stay with their elements
    function syncLabelPositions() {
        labelMap.forEach((labels, element) => {
            const rect = element.getBoundingClientRect();
            labels.forEach((label, index) => {
                const offset = index * 20; // Offset each label by 20px vertically
                label.style.top = `${rect.top + window.scrollY - 20 - offset}px`;
                label.style.left = `${rect.left + window.scrollX}px`;
            });
        });

        requestAnimationFrame(syncLabelPositions); // Continuously sync positions
    }

    // Start syncing label positions
    syncLabelPositions();

    // Generate a unique ID for each element
    function getElementUniqueId(element) {
        if (!element.dataset.uniqueId) {
            element.dataset.uniqueId = `uid-${Date.now()}-${Math.random().toString(36).substring(2, 8)}`;
        }
        return element.dataset.uniqueId;
    }

    // Event listeners
    document.addEventListener('click', (event) => {
        const element = event.target;
        highlightInteractedElement(element, '#00FF00');
    });

    document.addEventListener('select', (event) => {
        const element = event.target;
        highlightInteractedElement(element, '#00FF00');
    });

    document.addEventListener('input', (event) => {
        const element = event.target;
        highlightInteractedElement(element, '#00FF00');
    });
})();
