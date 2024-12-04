(() => {
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

  function getXPath(element) {
      if (!element || element.nodeType !== Node.ELEMENT_NODE) return '';

      // 1. Check for data-testid
      if (element.hasAttribute('data-testid')) {
          return `//*[@data-testid="${element.getAttribute('data-testid')}"]`;
      }

      // 2. Check for id
      if (element.id) {
          return `//*[@id="${element.id}"]`;
      }

      // 3. Check for unique text content
      const tagName = element.nodeName.toLowerCase();
      if (['button', 'a', 'span', 'div'].includes(tagName) && element.textContent.trim()) {
          const text = element.textContent.trim();
          return `//${tagName}[text()="${text}"]`;
      }

      // 4. Check for unique href attribute
      if (element.hasAttribute('href')) {
          return `//${tagName}[@href="${element.getAttribute('href')}"]`;
      }

      // 5. Fallback to sibling index
      let index = 1;
      let sibling = element.previousElementSibling;
      while (sibling) {
          if (sibling.nodeName === element.nodeName) {
              index++;
          }
          sibling = sibling.previousElementSibling;
      }

      return `${getXPath(element.parentNode)}/${tagName}[${index}]`;
  }

  function highlightElement(event) {
      if (lastHighlightedElement) {
          lastHighlightedElement.style.outline = '';
      }
      const element = event.target;
      element.style.outline = '2px solid #FF671D';
      lastHighlightedElement = element;

      const xpath = getXPath(element);
      const rect = element.getBoundingClientRect();
      selectorOverlay.textContent = xpath;
      selectorOverlay.style.top = `${rect.top + window.scrollY}px`;
      selectorOverlay.style.left = `${rect.left + window.scrollX}px`;
  }

  function removeHighlight() {
      if (lastHighlightedElement) {
          lastHighlightedElement.style.outline = '';
          lastHighlightedElement = null;
      }
      selectorOverlay.textContent = '';
  }

  document.addEventListener('mouseover', highlightElement);
  document.addEventListener('mouseout', removeHighlight);
})();
