package common

import (
	"testing"

	"github.com/grafana/xk6-browser/k6ext"
	"github.com/grafana/xk6-browser/k6ext/k6test"

	"github.com/stretchr/testify/assert"
)

func TestBrowserContextOptionsPermissions(t *testing.T) {
	vu := k6test.NewVU(t)

	var opts BrowserContextOptions
	ctx := k6ext.WithVU(vu.Context(), vu.TestRT.VU)
	err := opts.Parse(ctx, vu.ToSobekValue((struct {
		Permissions []any `js:"permissions"`
	}{
		Permissions: []any{"camera", "microphone"},
	})))
	assert.NoError(t, err)
	assert.Len(t, opts.Permissions, 2)
	assert.Equal(t, opts.Permissions, []string{"camera", "microphone"})
}
