package browser

import (
	"context"
	"errors"

	"github.com/grafana/sobek"

	"github.com/grafana/xk6-browser/k6error"
	"github.com/grafana/xk6-browser/k6ext"
)

func panicIfFatalError(ctx context.Context, err error) {
	if errors.Is(err, k6error.ErrFatal) {
		k6ext.Abort(ctx, err.Error())
	}
}

// exportArg exports the value and returns it.
// It returns nil if the value is undefined or null.
func exportArg(gv sobek.Value) any {
	if !gojaValueExists(gv) {
		return nil
	}
	return gv.Export()
}

// exportArgs returns a slice of exported Goja values.
func exportArgs(gargs []sobek.Value) []any {
	args := make([]any, 0, len(gargs))
	for _, garg := range gargs {
		// leaves a nil garg in the array since users might want to
		// pass undefined or null as an argument to a function
		args = append(args, exportArg(garg))
	}
	return args
}

// gojaValueExists returns true if a given value is not nil and exists
// (defined and not null) in the goja runtime.
func gojaValueExists(v sobek.Value) bool {
	return v != nil && !sobek.IsUndefined(v) && !sobek.IsNull(v)
}
