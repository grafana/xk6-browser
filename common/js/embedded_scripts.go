package js

import (
	_ "embed"
)

// WebVitalIIFEScript was downloaded from
// https://unpkg.com/web-vitals@3/dist/web-vitals.iife.js.
// Repo: https://github.com/GoogleChrome/web-vitals
//
//go:embed web_vital_iife.js
var WebVitalIIFEScript string

// WebVitalInitScript uses WebVitalIIFEScript
// and applies it to the current website that
// this init script is used against.
//
//go:embed web_vital_init.js
var WebVitalInitScript string

// SelectorEngineScript embeds a script that will highlight
// elements and return a selector to them when the mouse is
// hovered over the element.
//
//go:embed selector_engine.js
var SelectorEngineScript string

// InteractionHighlighterScript embeds a script that will
// highlight elements that have been interacted with, e.g.
// when an element has been clicked on.
//
//go:embed interaction_highlighter.js
var InteractionHighlighterScript string

// AutoScreenshotSignalScript embeds a script that will
// signal when the script has loaded and when an
// interaction on the page occurs.
//
//go:embed auto_screenshot.js
var AutoScreenshotSignalScript string
