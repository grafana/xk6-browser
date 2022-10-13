package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/grafana/xk6-browser/k6ext"

	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/dop251/goja"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

type objectOverflowError struct{}

// Error returns the description of the overflow error.
func (*objectOverflowError) Error() string {
	return "object is too large and will be parsed partially"
}

type objectPropertyParseError struct {
	error
	property string
}

// Error returns the reason of the failure, including the wrapper parsing error
// message.
func (pe *objectPropertyParseError) Error() string {
	return fmt.Sprintf("parsing object property %q: %s", pe.property, pe.error)
}

// Unwrap returns the wrapped parsing error.
func (pe *objectPropertyParseError) Unwrap() error {
	return pe.error
}

func parseRemoteObjectPreview(op *cdpruntime.ObjectPreview) (map[string]interface{}, error) {
	obj := make(map[string]interface{})
	var result error
	if op.Overflow {
		result = multierror.Append(result, &objectOverflowError{})
	}

	for _, p := range op.Properties {
		val, err := parseRemoteObjectValue(p.Type, p.Value, p.ValuePreview)
		if err != nil {
			result = multierror.Append(result, &objectPropertyParseError{err, p.Name})
			continue
		}
		obj[p.Name] = val
	}

	return obj, result
}

func parseRemoteObjectValue(t cdpruntime.Type, val string, op *cdpruntime.ObjectPreview) (interface{}, error) {
	switch t {
	case cdpruntime.TypeAccessor:
		return "accessor", nil
	case cdpruntime.TypeBigint:
		n, err := strconv.ParseInt(strings.Replace(val, "n", "", -1), 10, 64)
		if err != nil {
			return nil, BigIntParseError{err}
		}
		return n, nil
	case cdpruntime.TypeFunction:
		return "function()", nil
	case cdpruntime.TypeString:
		if !strings.HasPrefix(val, `"`) {
			return val, nil
		}
	case cdpruntime.TypeSymbol:
		return val, nil
	case cdpruntime.TypeObject:
		if op != nil {
			return parseRemoteObjectPreview(op)
		}
		if val == "Object" {
			return val, nil
		}
	case cdpruntime.TypeUndefined:
		return "undefined", nil
	}

	var v interface{}
	if err := json.Unmarshal([]byte(val), &v); err != nil {
		return nil, err
	}

	return v, nil
}

func parseExceptionDetails(exc *cdpruntime.ExceptionDetails) string {
	if exc == nil {
		return ""
	}
	var errMsg string
	if exc.Exception != nil {
		errMsg = exc.Exception.Description
		if errMsg == "" {
			if o, _ := parseRemoteObject(exc.Exception); o != nil {
				errMsg = fmt.Sprintf("%s", o)
			}
		}
	}
	return errMsg
}

func parseRemoteObject(obj *cdpruntime.RemoteObject) (interface{}, error) {
	if obj.UnserializableValue == "" {
		return parseRemoteObjectValue(obj.Type, string(obj.Value), obj.Preview)
	}

	switch obj.UnserializableValue.String() {
	case "-0": // To handle +0 divided by negative number
		return math.Float64frombits(0 | (1 << 63)), nil
	case "NaN":
		return math.NaN(), nil
	case "Infinity":
		return math.Inf(0), nil
	case "-Infinity":
		return math.Inf(-1), nil
	}

	return nil, UnserializableValueError{obj.UnserializableValue}
}

func valueFromRemoteObject(ctx context.Context, robj *cdpruntime.RemoteObject) (goja.Value, error) {
	val, err := parseRemoteObject(robj)
	if val == "undefined" {
		return goja.Undefined(), err
	}
	return k6ext.Runtime(ctx).ToValue(val), err
}

func handleParseRemoteObjectErr(ctx context.Context, err error, logger *logrus.Entry) {
	var (
		ooe *objectOverflowError
		ope *objectPropertyParseError
	)
	merr, ok := err.(*multierror.Error)
	if !ok {
		// If this panics it's a bug :)
		k6ext.Panic(ctx, "parsing remote object value: %w", err)
	}
	for _, e := range merr.Errors {
		switch {
		case errors.As(e, &ooe):
			logger.Warn(ooe)
		case errors.As(e, &ope):
			logger.WithError(ope).Error()
		default:
			// If this panics it's a bug :)
			k6ext.Panic(ctx, "parsing remote object value: %w", e)
		}
	}
}
