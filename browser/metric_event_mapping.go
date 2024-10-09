package browser

import (
	"fmt"

	"github.com/grafana/xk6-browser/common"
)

// mapMetricEvent to the JS module.
func mapMetricEvent(vu moduleVU, cm *common.MetricEvent) (mapping, error) {
	rt := vu.VU.Runtime()

	// We're setting up the function in the Sobek context that will be reused
	// for this VU.
	_, err := rt.RunString(`
	function _k6BrowserCheckRegEx(pattern, url) {
		let r = pattern;
		if (typeof pattern === 'string') {
			r = new RegExp(pattern);
		}
		return r.test(url);
	}`)
	if err != nil {
		return nil, fmt.Errorf("evaluating regex function: %w", err)
	}

	return mapping{
		"tag": func(urls common.URLTagPatterns) error {
			callback := func(pattern, url string) (bool, error) {
				js := fmt.Sprintf(`_k6BrowserCheckRegEx(%s, '%s')`, pattern, url)

				matched, err := rt.RunString(js)
				if err != nil {
					return false, fmt.Errorf("matching url with regex: %w", err)
				}

				return matched.ToBoolean(), nil
			}

			return cm.Tag(callback, urls)
		},
	}, nil
}
