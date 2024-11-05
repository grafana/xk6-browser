package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/xk6-browser/common"
	"github.com/grafana/xk6-browser/k6ext/k6test"
)

func TestBrowserContextOptionsPermissions(t *testing.T) {
	vu := k6test.NewVU(t)

	opts, err := parseBrowserContextOptions(vu.Context(), vu.ToSobekValue((struct {
		Permissions []any `js:"permissions"`
	}{
		Permissions: []any{"camera", "microphone"},
	})))
	assert.NoError(t, err)
	assert.Len(t, opts.Permissions, 2)
	assert.Equal(t, opts.Permissions, []string{"camera", "microphone"})
}

func TestBrowserContextSetGeolocation(t *testing.T) {
	vu := k6test.NewVU(t)

	opts, err := parseBrowserContextOptions(vu.Context(), vu.ToSobekValue((struct {
		GeoLocation *common.Geolocation `js:"geolocation"`
	}{
		GeoLocation: &common.Geolocation{
			Latitude:  1.0,
			Longitude: 2.0,
			Accuracy:  3.0,
		},
	})))
	assert.NoError(t, err)
	assert.NotNil(t, opts)
	assert.Equal(t, 1.0, opts.Geolocation.Latitude)
	assert.Equal(t, 2.0, opts.Geolocation.Longitude)
	assert.Equal(t, 3.0, opts.Geolocation.Accuracy)
}