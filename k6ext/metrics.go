package k6ext

import k6metrics "go.k6.io/k6/metrics"

// CustomMetrics are the custom k6 metrics used by xk6-browser.
type CustomMetrics struct {
	BrowserDOMContentLoaded     *k6metrics.Metric
	BrowserFirstPaint           *k6metrics.Metric
	BrowserFirstMeaningfulPaint *k6metrics.Metric
	BrowserLoaded               *k6metrics.Metric

	WebVitals map[string]*k6metrics.Metric
}

// RegisterCustomMetrics creates and registers our custom metrics with the k6
// VU Registry and returns our internal struct pointer.
func RegisterCustomMetrics(registry *k6metrics.Registry) *CustomMetrics {
	wvs := map[string]string{
		"LCP":  "browser_largest_content_paint",
		"FID":  "browser_first_input_delay",
		"CLS":  "browser_cumulative_layout_shift",
		"FCP":  "browser_first_contentful_paint",
		"TTFB": "browser_time_to_first_byte",
		"INP":  "browser_interaction_to_next_paint",
	}
	webVitals := make(map[string]*k6metrics.Metric)

	for k, v := range wvs {
		t := k6metrics.Time
		if k == "CLS" {
			t = k6metrics.Default
		}

		webVitals[k] = registry.MustNewMetric(
			v, k6metrics.Trend, t)

		webVitals[k+":good"] = registry.MustNewMetric(
			v+"_good", k6metrics.Counter)
		webVitals[k+":needs-improvement"] = registry.MustNewMetric(
			v+"_needs_improvement", k6metrics.Counter)
		webVitals[k+":poor"] = registry.MustNewMetric(
			v+"_poor", k6metrics.Counter)
	}

	return &CustomMetrics{
		BrowserDOMContentLoaded: registry.MustNewMetric(
			"browser_dom_content_loaded", k6metrics.Trend, k6metrics.Time),
		BrowserFirstPaint: registry.MustNewMetric(
			"browser_first_paint", k6metrics.Trend, k6metrics.Time),
		BrowserFirstMeaningfulPaint: registry.MustNewMetric(
			"browser_first_meaningful_paint", k6metrics.Trend, k6metrics.Time),
		BrowserLoaded: registry.MustNewMetric(
			"browser_loaded", k6metrics.Trend, k6metrics.Time),

		WebVitals: webVitals,
	}
}
