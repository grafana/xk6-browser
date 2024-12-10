(() => {
  // Selector Finder Function
  function findBestSelector(element) {
    // Prefer aria-label or aria-labelledby
    if (!element.hasAttribute('aria-label')) {
      // 1. Check for `data-testid`
      if (element.hasAttribute('data-testid')) {
        return `'[data-testid="${element.getAttribute('data-testid')}"]'`;
      }

      // 2. Check for `id`
      if (element.id) {
        return `'#${element.id}'`;
      }
    }

    // 3. Check for role and accessible name (explicit or implicit roles)
    const role = getRole(element);
    if (role) {
      const name = getAccessibleName(element);
      if (name) {
        return `'role=${role}[name="${name}"]'`;
      }
      return `'role=${role}'`;
    }

    // 4. Check for visible text
    const text = element.textContent.trim();
    if (text) {
      return `'text="${text}"'`;
    }

    // 5. Fallback to XPath
    return generateXPath(element);
  }

  // Helper function to compute the role (explicit or implicit)
  function getRole(element) {
    // Check for explicit role
    if (element.hasAttribute('role')) {
      return element.getAttribute('role');
    }

    // Implicit role mapping
    const implicitRoles = {
      button: ['button', "input[type='button']", "input[type='submit']", "input[type='reset']"],
      link: ['a[href]'],
      checkbox: ["input[type='checkbox']"],
      heading: ['h1', 'h2', 'h3', 'h4', 'h5', 'h6'],
      dialog: ['dialog'],
      img: ['img[alt]'],
      textbox: ["input[type='text']", "input[type='email']", "input[type='password']", 'textarea'],
      radio: ["input[type='radio']"],
      // Add more implicit roles if needed
    };

    for (const [role, selectors] of Object.entries(implicitRoles)) {
      for (const selector of selectors) {
        if (element.matches(selector)) {
          return role;
        }
      }
    }

    return null;
  }

  // Helper function to compute the accessible name of an element
  function getAccessibleName(element) {
    // Prefer aria-label or aria-labelledby
    if (element.hasAttribute('aria-label')) {
      return element.getAttribute('aria-label');
    }
    if (element.hasAttribute('aria-labelledby')) {
      const labelId = element.getAttribute('aria-labelledby');
      const labelElement = element.ownerDocument.getElementById(labelId);
      return labelElement ? labelElement.textContent.trim() : '';
    }
    // Use text content as a fallback
    return element.textContent.trim();
  }

  // Helper function to generate XPath as a fallback
  function generateXPath(element) {
    if (element.id) {
      return `'//*[@id="${element.id}"]'`;
    }
    const siblings = Array.from(element.parentNode.children).filter(
      (el) => el.nodeName === element.nodeName
    );
    const index = siblings.indexOf(element) + 1;
    const tagName = element.nodeName.toLowerCase();
    if (element.parentNode === document) {
      return `'/${tagName}[${index}]'`;
    }
    return `'${generateXPath(element.parentNode)}/${tagName}[${index}]'`;
  }

  // Highlight and Selector Display
  let lastHighlightedElement = null;
  const selectorOverlay = document.createElement('div');
  selectorOverlay.style.position = 'absolute';
  selectorOverlay.style.background = 'rgba(0, 0, 0, 0.8)';
  selectorOverlay.style.color = '#fff';
  selectorOverlay.style.padding = '5px';
  selectorOverlay.style.fontSize = '12px';
  selectorOverlay.style.borderRadius = '5px';
  selectorOverlay.style.pointerEvents = 'none';
  selectorOverlay.style.zIndex = '9999';
  document.body.appendChild(selectorOverlay);

  // Helper to copy text to clipboard
  function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
      console.log(`Copied to clipboard: ${text}`);
      showTemporaryMessage('Copied!', selectorOverlay);
    }).catch((err) => {
      console.error('Failed to copy text: ', err);
      showTemporaryMessage('Failed to copy', selectorOverlay);
    });
  }

  // Show a temporary message in the overlay
  function showTemporaryMessage(message, overlay) {
    const originalText = overlay.textContent;
    overlay.textContent = message;
    setTimeout(() => {
      overlay.textContent = originalText;
    }, 1000); // Reset after 1 second
  }

  // Highlight the element and show selector
  function highlightElement(event) {
    if (lastHighlightedElement) {
      lastHighlightedElement.style.outline = '';
    }
    const element = event.target;
    element.style.outline = '2px solid #FF671D';
    lastHighlightedElement = element;

    const selector = findBestSelector(element);
    const rect = element.getBoundingClientRect();
    selectorOverlay.textContent = selector;
    selectorOverlay.style.top = `${rect.top + window.scrollY}px`;
    selectorOverlay.style.left = `${rect.left + window.scrollX}px`;

    // Copy to clipboard on Command + C
    document.onkeydown = (e) => {
      if (e.metaKey && e.key === 'c') { // Press Command + C
        e.preventDefault();
        copyToClipboard(selector);
      }
    };
  }

  function removeHighlight() {
    if (lastHighlightedElement) {
      lastHighlightedElement.style.outline = '';
      lastHighlightedElement = null;
    }
    selectorOverlay.textContent = '';
    document.onkeydown = null; // Remove keydown listener
  }

  document.addEventListener('mouseover', highlightElement);
  document.addEventListener('mouseout', removeHighlight);
})();