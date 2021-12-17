/*
 *
 * xk6-browser - a browser automation extension for k6
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package tests

import (
	"testing"
	"time"

	"github.com/grafana/xk6-browser/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k6metrics "go.k6.io/k6/lib/metrics"
	k6stats "go.k6.io/k6/stats"
)

func TestDataURLSkipRequest(t *testing.T) {
	tb := newTestBrowser(t)
	p := tb.NewPage(nil)

	lc := attachLogCache(tb.state.Logger)

	p.Goto("data:text/html,hello", nil)

	assert.True(t, lc.contains("skipped request handling of data URL"))
}

func TestMetricsEmission(t *testing.T) {
	tb := newTestBrowser(t, withHTTPServer())

	url := tb.URL("/get")
	browserLoadedTags := map[string]string{
		"group": "",
		"url":   "about:blank",
	}
	browserTags := map[string]string{
		"group": "",
		"url":   url,
	}
	httpTags := map[string]string{
		"method":              "GET",
		"url":                 url,
		"status":              "200",
		"group":               "",
		"proto":               "http/1.1",
		"from_cache":          "false",
		"from_prefetch_cache": "false",
		"from_service_worker": "false",
	}
	expMetricTags := map[string]map[string]string{
		common.BrowserDOMContentLoaded.Name:     browserLoadedTags,
		common.BrowserLoaded.Name:               browserLoadedTags,
		common.BrowserFirstPaint.Name:           browserTags,
		common.BrowserFirstContentfulPaint.Name: browserTags,
		k6metrics.DataSentName: map[string]string{
			"group":  "",
			"method": "GET",
			"url":    url,
		},
		k6metrics.HTTPReqsName:              httpTags,
		k6metrics.HTTPReqDurationName:       httpTags,
		k6metrics.DataReceivedName:          httpTags,
		k6metrics.HTTPReqConnectingName:     httpTags,
		k6metrics.HTTPReqTLSHandshakingName: httpTags,
		k6metrics.HTTPReqSendingName:        httpTags,
		k6metrics.HTTPReqReceivingName:      httpTags,
	}

	p := tb.NewPage(nil)
	resp := p.Goto(url, tb.rt.ToValue(struct {
		WaitUntil string `js:"waitUntil"`
	}{WaitUntil: "networkidle"}))
	require.NotNil(t, resp)

	// TODO: Remove this sleep. It's only needed to wait for all metrics to be
	// emitted, but this should be synchronized with waitUntil
	// load/networkidle/domcontentloaded. Without this the test would be flaky.
	time.Sleep(100 * time.Millisecond)

	bufSamples := k6stats.GetBufferedSamples(tb.samples)

	var reqsCount int
	cb := func(sample k6stats.Sample) {
		switch sample.Metric.Name {
		case k6metrics.HTTPReqsName:
			reqsCount += int(sample.Value)
		case k6metrics.DataSentName, k6metrics.DataReceivedName:
			assert.Greaterf(t, int(sample.Value), 0,
				"metric %s", sample.Metric.Name)
		}
	}

	assertMetricsEmitted(t, bufSamples, expMetricTags, cb)
	assert.Equal(t, 1, reqsCount)
}
