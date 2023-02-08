// Package osext is in charge of the interaction
// with the OS process that is running the browser.
package osext

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strings"

	"github.com/grafana/xk6-browser/log"
	"github.com/grafana/xk6-browser/storage"
)

// Process is a wrapper around the os.Process.
// We can keep track of and control its lifecycle.
// It's crucial in setting up and connecting to
// the running browser.
type Process struct {
	ctx    context.Context
	cancel context.CancelFunc

	// The process of the browser, if running locally.
	process *os.Process

	// Channels for managing termination.
	LostConnection             chan struct{}
	processIsGracefullyClosing chan struct{}
	ProcessDone                chan struct{}

	// Browser's WebSocket URL to speak CDP
	wsURL string

	// The directory where user data for the browser is stored.
	UserDataDir *storage.Dir

	logger *log.Logger
}

// NewProcess creates a new osext. It will start
// a child process where the internet browser will
// run.
func NewProcess(
	ctx context.Context, path string, args, env []string, dataDir *storage.Dir,
	ctxCancel context.CancelFunc, logger *log.Logger,
) (*Process, error) {
	cmd, err := execute(ctx, path, args, env, dataDir, logger)
	if err != nil {
		return nil, err
	}

	wsURL, err := parseDevToolsURL(ctx, cmd)
	if err != nil {
		return nil, err
	}

	register(ctx, logger, cmd.Process.Pid)

	p := Process{
		ctx:                        ctx,
		cancel:                     ctxCancel,
		process:                    cmd.Process,
		LostConnection:             make(chan struct{}),
		processIsGracefullyClosing: make(chan struct{}),
		ProcessDone:                cmd.done,
		wsURL:                      wsURL,
		UserDataDir:                dataDir,
	}

	go func() {
		// If we lose connection to the browser and we're not in-progress with clean
		// browser-initiated termination then cancel the context to clean up.
		select {
		case <-p.LostConnection:
		case <-ctx.Done():
		}

		select {
		case <-p.processIsGracefullyClosing:
		default:
			p.cancel()
		}
	}()

	return &p, nil
}

// DidLoseConnection should be called
// when we get a EventConnectionClose
// or when the context is closed.
func (p *Process) DidLoseConnection() {
	close(p.LostConnection)
}

// IsConnected returns whether the WebSocket
// connection to the browser process is active
// or not.
func (p *Process) IsConnected() bool {
	var ok bool
	select {
	case _, ok = <-p.LostConnection:
	default:
		ok = true
	}
	return ok
}

// GracefulClose triggers a graceful closing of the browser process.
func (p *Process) GracefulClose() {
	p.logger.Debugf("Browser:GracefulClose", "")
	close(p.processIsGracefullyClosing)
}

// Terminate triggers the termination of the browser process.
func (p *Process) Terminate() {
	p.logger.Debugf("Browser:Close", "browserProc terminate")
	p.cancel()
}

// WsURL returns the Websocket URL that the browser is listening on for CDP clients.
func (p *Process) WsURL() string {
	return p.wsURL
}

// Pid returns the browser process ID.
func (p *Process) Pid() int {
	return p.process.Pid
}

// AttachLogger attaches a logger to the browser process.
func (p *Process) AttachLogger(logger *log.Logger) {
	p.logger = logger
}

type command struct {
	*exec.Cmd
	done           chan struct{}
	stdout, stderr io.Reader
}

func execute(
	ctx context.Context, path string, args, env []string,
	dataDir *storage.Dir, logger *log.Logger,
) (command, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	killAfterParent(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return command{}, fmt.Errorf("%w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return command{}, fmt.Errorf("%w", err)
	}

	// Set up environment variable for process
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	// We must start the cmd before calling cmd.Wait, as otherwise the two
	// can run into a data race.
	err = cmd.Start()
	if os.IsNotExist(err) {
		return command{}, fmt.Errorf("file does not exist: %s", path)
	}
	if err != nil {
		return command{}, fmt.Errorf("%w", err)
	}
	if ctx.Err() != nil {
		return command{}, fmt.Errorf("%w", ctx.Err())
	}

	done := make(chan struct{})
	go func() {
		// TODO: How to handle these errors?
		defer func() {
			if err := dataDir.Cleanup(); err != nil {
				logger.Errorf("browser", "cleaning up the user data directory: %v", err)
			}
			close(done)
		}()

		if err := cmd.Wait(); err != nil {
			logger.Errorf("browser",
				"process with PID %d unexpectedly ended: %v",
				cmd.Process.Pid, err)
		}
	}()

	return command{cmd, done, stdout, stderr}, nil
}

// parseDevToolsURL grabs the WebSocket address from Chrome's output and returns
// it. If the process ends abruptly, it will return the first error from stderr.
func parseDevToolsURL(ctx context.Context, cmd command) (_ string, err error) {
	parser := &devToolsURLParser{
		sc: bufio.NewScanner(cmd.stderr),
	}
	done := make(chan struct{})
	go func() {
		for parser.scan() {
		}
		close(done)
	}()
	for err == nil {
		select {
		case <-done:
			err = parser.err()
		case <-ctx.Done():
			err = ctx.Err()
		case <-cmd.done:
			err = errors.New("browser process ended unexpectedly")
		}
	}
	if parser.url != "" {
		err = nil
	}

	return parser.url, err
}

type devToolsURLParser struct {
	sc *bufio.Scanner

	errs []error
	url  string
}

func (p *devToolsURLParser) scan() bool {
	if !p.sc.Scan() {
		return false
	}

	const urlPrefix = "DevTools listening on "

	line := p.sc.Text()
	if strings.HasPrefix(line, urlPrefix) {
		p.url = strings.TrimPrefix(strings.TrimSpace(line), urlPrefix)
	}
	if strings.Contains(line, ":ERROR:") {
		if i := strings.Index(line, "] "); i > 0 {
			p.errs = append(p.errs, errors.New(line[i+2:]))
		}
	}

	return p.url == ""
}

func (p *devToolsURLParser) err() error {
	if p.url != "" {
		return io.EOF
	}
	if len(p.errs) > 0 {
		return p.errs[0]
	}

	err := p.sc.Err()
	if errors.Is(err, fs.ErrClosed) {
		return fmt.Errorf("browser process shutdown unexpectedly before establishing a connection: %w", err)
	}
	if err != nil {
		return err //nolint:wrapcheck
	}

	return nil
}
