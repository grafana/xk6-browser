package common

import (
	"testing"

	"github.com/grafana/xk6-browser/k6ext/k6test"

	"github.com/stretchr/testify/assert"
)

func TestBrowserContextOptionsPermissions(t *testing.T) {
	vu := k6test.NewVU(t)

	var opts BrowserContextOptions
	err := opts.Parse(vu.Context(), vu.ToSobekValue((struct {
		Permissions []any `js:"permissions"`
	}{
		Permissions: []any{"accelerometer",
			"accessibility-events",
			"ambient-light-sensor",
			"background-sync",
			"camera",
			"clipboard-read",
			"clipboard-write",
			"geolocation",
			"gyroscope",
			"magnetometer",
			"microphone",
			"midi-sysex",
			"midi",
			"notifications",
			"payment-handler",
			"storage-access"},
	})))
	assert.NoError(t, err)
	assert.Len(t, opts.Permissions, 16)
	assert.Equal(t, opts.Permissions, []string{"accelerometer",
		"accessibility-events",
		"ambient-light-sensor",
		"background-sync",
		"camera",
		"clipboard-read",
		"clipboard-write",
		"geolocation",
		"gyroscope",
		"magnetometer",
		"microphone",
		"midi-sysex",
		"midi",
		"notifications",
		"payment-handler",
		"storage-access"})
}
