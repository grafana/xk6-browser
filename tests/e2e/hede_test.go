package e2e

import (
	"bytes"
	"embed"
	"io"
	"strconv"
	"testing"

	"github.com/grafana/xk6-browser/browser"

	"go.k6.io/k6/cmd"
	"go.k6.io/k6/cmd/tests"
	"go.k6.io/k6/js/modules"
)

//go:embed tests
var testScripts embed.FS

/*
const (
	CloudTestRunFailed       ExitCode = 97 // This used to be 99 before k6 v0.33.0
	CloudFailedToGetProgress ExitCode = 98
	ThresholdsHaveFailed     ExitCode = 99
	SetupTimeout             ExitCode = 100
	TeardownTimeout          ExitCode = 101
	GenericTimeout           ExitCode = 102 // TODO: remove?
	ScriptStoppedFromRESTAPI ExitCode = 103
	InvalidConfig            ExitCode = 104
	ExternalAbort            ExitCode = 105
	CannotStartRESTAPI       ExitCode = 106
	ScriptException          ExitCode = 107
	ScriptAborted            ExitCode = 108
)
*/

func TestRunCurrentModule(t *testing.T) {
	t.Parallel()

	for i := 1; i <= 10; i++ {
		t.Run("fillform"+strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			ts := newBrowserTest(t, "tests/fillform.js")
			cmd.ExecuteWithGlobalState(ts.GlobalState)
		})
	}

	// logs := ts.LoggerHook.Drain()

	// t.Log("LOGS:")
	// t.Log("----------------------------------------------")
	// for _, e := range logs {
	// 	t.Log(e)
	// }
	// stdout := ts.Stdout.String()
	// t.Log("STDOUT:")
	// t.Log("----------------------------------------------")
	// t.Log(stdout)

	// stderr := ts.Stderr.String()
	// t.Log("STDERR:")
	// t.Log("----------------------------------------------")
	// t.Log(stderr)

	// assert.Contains(t, stderr, "Object has no member 'extContent'")
	// assert.Contains(t, logs, "Object has no member 'extContent'")
	// assert.True(t, testutils.LogContains(logs, logrus.ErrorLevel, "Object has no member 'extContent'"))
}

func init() {
	modules.Register("k6/x/browser", browser.New())
}

func newBrowserTest(t *testing.T, name string) *tests.GlobalTestState {
	t.Helper()

	ts := tests.NewGlobalTestState(t)
	ts.CmdArgs = []string{"k6", "run", "-q", "-"}
	// ts.CmdArgs = []string{"k6", "run", "-v", "--log-output=stdout", "-"}
	// ts.Env["XK6_BROWSER_LOG"] = "debug"

	script := readTestScript(t, name)
	ts.Stdin = bytes.NewBuffer(script)

	return ts
}

func readTestScript(t *testing.T, name string) []byte {
	t.Helper()

	f, err := testScripts.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Logf("readFile.closing %q: %v", name, err)
		}
	}()
	buf, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	return buf
}
