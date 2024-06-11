package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSHandleEvaluate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pageFunc string
		args     []any
		expected string
	}{
		{
			name:     "no_args",
			pageFunc: `handle => handle.innerText`,
			args:     nil,
			expected: "Some title",
		},
		{
			name: "with_args",
			pageFunc: `(handle, a, b) => {
				const c = a + b;
				return handle.innerText + " " + c
			}`,
			args:     []any{1, 2},
			expected: "Some title 3",
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tb := newTestBrowser(t)
			p := tb.NewPage(nil)

			err := p.SetContent(`<html><head><title>Some title</title></head></html>`, nil)
			require.NoError(t, err)

			result, err := p.EvaluateHandle(`() => document.head`)
			require.NoError(t, err)
			require.NotNil(t, result)

			got, err := result.Evaluate(tt.pageFunc, tt.args...)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
