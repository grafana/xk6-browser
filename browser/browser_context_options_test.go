package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/xk6-browser/k6ext/k6test"
)

func TestBrowserContextOptionsPermissions(t *testing.T) {
	vu := k6test.NewVU(t)

	opts, err := ParseBrowserContextOptions(vu.Context(),
		vu.ToSobekValue((struct {
			Permissions []any `js:"permissions"`
		}{
			Permissions: []any{"camera", "microphone"},
		})))
	assert.NoError(t, err)
	assert.Len(t, opts.Permissions, 2)
	assert.Equal(t, opts.Permissions, []string{"camera", "microphone"})
}
