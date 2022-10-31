package common

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockReader struct {
	lines []string
	err   error
}

func (r mockReader) Read(p []byte) (n int, err error) {
	for _, l := range r.lines {
		n += copy(p, []byte(l+"\n"))
	}
	return n, r.err
}

func TestParseDevToolsURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		stderr             []string
		readErr            error
		prematureCtxCancel bool
		prematureCmdDone   bool
		assert             func(t *testing.T, wsURL string, err error)
	}{
		{
			name: "ok/no_error",
			stderr: []string{
				`DevTools listening on ws://127.0.0.1:41315/devtools/browser/d1d3f8eb-b362-4f12-9370-bd25778d0da7`,
			},
			assert: func(t *testing.T, wsURL string, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, "ws://127.0.0.1:41315/devtools/browser/d1d3f8eb-b362-4f12-9370-bd25778d0da7", wsURL)
			},
		},
		{
			name: "ok/non-fatal_error",
			stderr: []string{
				`[23400:23418:1028/115455.877614:ERROR:bus.cc(399)] Failed to ` +
					`connect to the bus: Could not parse server address: ` +
					`Unknown address type (examples of valid types are "tcp" ` +
					`and on UNIX "unix")`,
				"",
				`DevTools listening on ws://127.0.0.1:41315/devtools/browser/d1d3f8eb-b362-4f12-9370-bd25778d0da7`,
			},
			assert: func(t *testing.T, wsURL string, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, "ws://127.0.0.1:41315/devtools/browser/d1d3f8eb-b362-4f12-9370-bd25778d0da7", wsURL)
			},
		},
		{
			name: "err/fatal-eof",
			stderr: []string{
				`[6497:6497:1013/103521.932979:ERROR:ozone_platform_x11` +
					`.cc(247)] Missing X server or $DISPLAY` + "\n",
			},
			readErr: io.ErrUnexpectedEOF,
			assert: func(t *testing.T, wsURL string, err error) {
				t.Helper()
				require.Empty(t, wsURL)
				assert.EqualError(t, err, "Missing X server or $DISPLAY")
			},
		},
		{
			name:    "err/fatal-eof-no_stderr",
			readErr: io.ErrUnexpectedEOF,
			assert: func(t *testing.T, wsURL string, err error) {
				t.Helper()
				require.Empty(t, wsURL)
				assert.EqualError(t, err, "unexpected EOF")
			},
		},
		{
			name:             "err/fatal-premature_cmd_done",
			stderr:           []string{""},
			prematureCmdDone: true,
			assert: func(t *testing.T, wsURL string, err error) {
				t.Helper()
				require.Empty(t, wsURL)
				assert.EqualError(t, err, "browser process ended unexpectedly")
			},
		},
		{
			name:               "err/fatal-premature_ctx_cancel",
			stderr:             []string{""},
			prematureCtxCancel: true,
			assert: func(t *testing.T, wsURL string, err error) {
				t.Helper()
				require.Empty(t, wsURL)
				assert.EqualError(t, err, "context canceled")
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mr := mockReader{lines: tc.stderr, err: tc.readErr}
			cmdDone := make(chan struct{})
			cmd := command{done: cmdDone, stderr: mr}

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			timeout := time.Second
			timer := time.NewTimer(timeout)
			t.Cleanup(func() { _ = timer.Stop() })

			var (
				done  = make(chan struct{})
				wsURL string
				err   error
			)

			go func() {
				wsURL, err = parseDevToolsURL(ctx, cmd)
				close(done)
			}()

			if tc.prematureCmdDone {
				time.Sleep(200 * time.Millisecond)
				close(cmdDone)
			}

			if tc.prematureCtxCancel {
				time.Sleep(200 * time.Millisecond)
				cancel()
			}

			select {
			case <-done:
				tc.assert(t, wsURL, err)
			case <-timer.C:
				t.Errorf("test timed out after %s", timeout)
			}
		})
	}
}
