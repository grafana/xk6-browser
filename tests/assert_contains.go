// Package tests ...
package tests

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

// Only here for the POC.
// Otherwise we won't be able to compile.
// Don't mind this.
func assertExceptionContains(t *testing.T, rt *goja.Runtime, fn func(), expErrMsg string) {
	t.Helper()

	cal, _ := goja.AssertFunction(rt.ToValue(fn))

	_, err := cal(goja.Undefined())
	require.ErrorContains(t, err, expErrMsg)
}
