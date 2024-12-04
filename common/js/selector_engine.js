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
      if (element.id) {
          // If the element has an ID, use it directly
          return `//*[@id="${element.id}"]`;
      }
      if (element === document.body) {
          // Special case for <body>
          return '/html/body';
      }

      let index = 1;
      let sibling = element.previousElementSibling;
      while (sibling) {
          if (sibling.nodeName === element.nodeName) {
              index++;
          }
          sibling = sibling.previousElementSibling;
      }

      const tagName = element.nodeName.toLowerCase();
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
