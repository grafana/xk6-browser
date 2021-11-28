package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/target"
)

// TODO:
// -> enable httptrace
// -> close websocket connection on interrupt
// -> websocket
//    ws.SetReadDeadline(time.Now().Add(pongWait))
//    ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

func start(ctx context.Context, websocketURL string, log *log.Logger) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Printf("connecting to %q", websocketURL)
	var c *connection
	{
		if c, err = connect(ctx, websocketURL); err != nil {
			return fmt.Errorf("%w", err)
		}
		defer func() {
			if errClose := c.ws.Close(); err == nil && errClose != nil {
				err = fmt.Errorf("start:c.ws.Close: %w", errClose)
			}
		}()
	}
	log.Println("connected")

	errs := make(chan error, 2)
	{
		recv := func(ctx context.Context) {
			for {
				buf, err := c.recv(ctx)
				if err != nil {
					errs <- fmt.Errorf("start:c.recv: %w", err)
					return
				}
				prettyf(log.Printf, "<- %s", buf)

			}
		}
		processUserInput := func(ctx context.Context, r io.Reader) {
			sc := bufio.NewScanner(r)
			for sc.Scan() {
				select {
				case <-ctx.Done():
					errs <- fmt.Errorf("start:processUserInput:ctx.Done: %w", ctx.Err())
					return
				default:
				}
				if len(sc.Bytes()) == 0 {
					continue
				}
				prettyf(log.Printf, "-> %s", sc.Bytes())
				if err := c.send(ctx, sc.Bytes()); err != nil {
					errs <- fmt.Errorf("start:processUserInput:c.send %w", err)
					return
				}
			}
			if err := sc.Err(); err != nil {
				errs <- err
			}
		}

		go recv(ctx)
		go processUserInput(ctx, os.Stdin)
	}

	{
		getVersion := browser.GetVersion()
		_, _, _, _, _, err := getVersion.Do(cdp.WithExecutor(ctx, c))
		if err != nil {
			return fmt.Errorf("start:%T: %w", getVersion, err)
		}
	}

	{
		autoAttach := target.SetAutoAttach(true, true).WithFlatten(true)
		err = autoAttach.Do(cdp.WithExecutor(ctx, c))
		if err != nil {
			return fmt.Errorf("start:%T: %w", autoAttach, err)
		}

		// Target.setAutoAttach has a bug where it does not wait for new Targets being attached.
		// However making a dummy call afterwards fixes this.
		// This can be removed after https://chromium-review.googlesource.com/c/chromium/src/+/2885888 lands in stable.
		getTargetInfo := target.GetTargetInfo()
		_, err = getTargetInfo.Do(cdp.WithExecutor(ctx, c))
		if err != nil {
			return fmt.Errorf("start:%T: %w", getTargetInfo, err)
		}
	}

	// action := page.Navigate("https://duckduckgo.com")
	// fid, lid, errt, err := action.Do(cdp.WithExecutor(ctx, c))
	// log.Println(fid, lid, errt, err)
	// if err != nil {
	// 	return err
	// }

	// {"id":1, "method":"Browser.getVersion"}

	// start receiving events:
	// {"id":100, "method":"Target.createTarget", "params":{"url": "https://duckduckgo.com"}}
	// stop receiving events:
	// {"id":101, "method": "Target.detachFromTarget", "params":{"sessionId": "0479DCBFC35D9B062F09FD1EAEE2639D"}}

	// page.Enable()
	// action := target.SetAutoAttach(true, true).WithFlatten(true)
	// if err := action.Do(cdp.WithExecutor(ctx, &fuck{})); err != nil {
	// 	return fmt.Errorf("unable to execute %T: %w", action, err)
	// }

	return <-errs
}

func prettyf(printf func(string, ...interface{}), format string, args ...interface{}) {
	b, ok := args[0].([]byte)
	if !ok {
		printf(format, args...)
		return
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, b, "", "  "); err != nil {
		printf(format, args...)
		return
	}
	printf(format, append([]interface{}{pretty.Bytes()}, args[1:]...)...)
}
