package browser

import (
	"testing"

	"github.com/grafana/xk6-browser/k6ext/k6test"

	"github.com/stretchr/testify/assert"
)

func TestContextOptionsPermissions(t *testing.T) {
	vu := k6test.NewVU(t)

	var opts ContextOptions
	err := opts.Parse(vu.Context(), vu.ToGojaValue((struct {
		Permissions []any `js:"permissions"`
	}{
		Permissions: []any{"camera", "microphone"},
	})))
	assert.NoError(t, err)
	assert.Len(t, opts.Permissions, 2)
	assert.Equal(t, opts.Permissions, []string{"camera", "microphone"})
}
