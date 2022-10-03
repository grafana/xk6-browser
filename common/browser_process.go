/*
 *
 * xk6-browser - a browser automation extension for k6
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package common

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/grafana/xk6-browser/log"
	"github.com/grafana/xk6-browser/storage"
)

type BrowserProcess struct {
	ctx    context.Context
	cancel context.CancelFunc

	// The process of the browser, if running locally.
	process *os.Process

	// Channels for managing termination.
	lostConnection             chan struct{}
	processIsGracefullyClosing chan struct{}
	processDone                chan struct{}

	// Browser's WebSocket URL to speak CDP
	wsURL string

	// The directory where user data for the browser is stored.
	userDataDir *storage.Dir

	logger *log.Logger
}

func NewBrowserProcess(
	ctx context.Context, path string, args, env []string, dataDir *storage.Dir,
	ctxCancel context.CancelFunc, logger *log.Logger,
) (*BrowserProcess, error) {
	cmd, procDone, err := execute(ctx, path, args, env, dataDir, logger)
	if err != nil {
		return nil, err
	}

	wsURL, err := getDevToolsURL(dataDir.Dir)
	if err != nil {
		return nil, fmt.Errorf("getting DevTools URL: %w", err)
	}

	p := BrowserProcess{
		ctx:                        ctx,
		cancel:                     ctxCancel,
		process:                    cmd.Process,
		lostConnection:             make(chan struct{}),
		processIsGracefullyClosing: make(chan struct{}),
		processDone:                procDone,
		wsURL:                      wsURL,
		userDataDir:                dataDir,
	}

	go func() {
		// If we lose connection to the browser and we're not in-progress with clean
		// browser-initiated termination then cancel the context to clean up.
		select {
		case <-p.lostConnection:
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

func (p *BrowserProcess) didLoseConnection() {
	close(p.lostConnection)
}

func (p *BrowserProcess) isConnected() bool {
	var ok bool
	select {
	case _, ok = <-p.lostConnection:
	default:
		ok = true
	}
	return ok
}

// GracefulClose triggers a graceful closing of the browser process.
func (p *BrowserProcess) GracefulClose() {
	p.logger.Debugf("Browser:GracefulClose", "")
	close(p.processIsGracefullyClosing)
}

// Terminate triggers the termination of the browser process.
func (p *BrowserProcess) Terminate() {
	p.logger.Debugf("Browser:Close", "browserProc terminate")
	p.cancel()
}

// WsURL returns the Websocket URL that the browser is listening on for CDP clients.
func (p *BrowserProcess) WsURL() string {
	return p.wsURL
}

// Pid returns the browser process ID.
func (p *BrowserProcess) Pid() int {
	return p.process.Pid
}

// AttachLogger attaches a logger to the browser process.
func (p *BrowserProcess) AttachLogger(logger *log.Logger) {
	p.logger = logger
}

func execute(
	ctx context.Context, path string, args, env []string, dataDir *storage.Dir,
	logger *log.Logger,
) (*exec.Cmd, chan struct{}, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	killAfterParent(cmd)

	// Set up environment variable for process
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	// We must start the cmd before calling cmd.Wait, as otherwise the two
	// can run into a data race.
	err := cmd.Start()
	if os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("file does not exist: %s", path)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("%w", err)
	}
	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("%w", ctx.Err())
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

	return cmd, done, nil
}

// getDevToolsURL returns the DevTools WebSocket address by reading the
// DevToolsActivePort file in the data directory.
func getDevToolsURL(dataDir string) (wsURL string, rerr error) {
	var (
		f                  *os.File
		fpath              = filepath.Join(dataDir, "DevToolsActivePort")
		maxReadAttempts    = 10
		readAttemptDelayMS = 50
	)

	// The browser might not have created the file yet, so try reading it
	// multiple times after a slight delay.
	for readAttempts := 0; readAttempts < maxReadAttempts; readAttempts++ {
		var err error
		f, err = os.Open(fpath) //nolint:gosec  // false positive, https://github.com/securego/gosec/issues/439
		if errors.Is(err, os.ErrNotExist) {
			time.Sleep(time.Duration(readAttemptDelayMS) * time.Millisecond)
			continue
		}
		if err != nil {
			return "", fmt.Errorf("reading %q: %w", fpath, err)
		}
		defer func() { rerr = f.Close() }()

		break
	}

	if f == nil {
		return "", fmt.Errorf("unable to read file %q in %s", fpath,
			time.Duration(maxReadAttempts*readAttemptDelayMS)*time.Millisecond)
	}

	fs := bufio.NewScanner(f)
	fs.Split(bufio.ScanLines)
	portURI := make([]string, 0, 2)

	for fs.Scan() {
		portURI = append(portURI, fs.Text())
	}

	return fmt.Sprintf("ws://127.0.0.1:%s%s", portURI[0], portURI[1]), nil
}
