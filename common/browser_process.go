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
	"context"
	"os"

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

	// Browser's WebSocket URL to speak CDP
	wsURL string

	// The directory where user data for the browser is stored.
	userDataDir *storage.Dir

	logger *log.Logger
}

func NewBrowserProcess(
	ctx context.Context, cancel context.CancelFunc, process *os.Process, wsURL string, dataDir *storage.Dir,
) *BrowserProcess {
	p := BrowserProcess{
		ctx:                        ctx,
		cancel:                     cancel,
		process:                    process,
		lostConnection:             make(chan struct{}),
		processIsGracefullyClosing: make(chan struct{}),
		wsURL:                      wsURL,
		userDataDir:                dataDir,
	}
	go func() {
		// If we lose connection to the browser and we're not in-progress with clean
		// browser-initiated termination then cancel the context to clean up.
		<-p.lostConnection
		select {
		case <-p.processIsGracefullyClosing:
		default:
			p.cancel()
		}
	}()
	return &p
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
